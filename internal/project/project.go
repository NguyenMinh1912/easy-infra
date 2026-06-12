// Package project ties together the project config, profiles, state, and the
// service registry into a single facade for the command layer. Commands depend
// on this package rather than re-implementing the load/validate/resolve
// sequence, keeping the cmd/ layer thin and the workflow logic in one place.
package project

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/minhnc/easy-infra/internal/config"
	"github.com/minhnc/easy-infra/internal/profile"
	"github.com/minhnc/easy-infra/internal/service"
	"github.com/minhnc/easy-infra/internal/state"
)

// ErrNotInitialized signals that the current folder has no easy-infra project.
// Commands surface it as a hint to run `easy-infra init`.
var ErrNotInitialized = errors.New("project not initialized")

// LocalProfile is the conventional name of the profile that holds a project's
// locally-forked services. "fork to local" writes the forked service's
// localised env here so it appears in the sidebar as a managed profile.
const LocalProfile = "local"

// Paths locates a project's config, state, and profile files.
type Paths struct {
	Config      string
	State       string
	ProfilesDir string
}

// DefaultPaths returns the conventional file locations for a project rooted at
// the current working directory.
func DefaultPaths() Paths {
	return Paths{
		Config:      config.DefaultPath,
		State:       state.DefaultPath,
		ProfilesDir: profile.DefaultDir,
	}
}

// ProfilePath returns the file path for the named profile.
func (p Paths) ProfilePath(name string) string {
	return profile.Path(p.ProfilesDir, name)
}

// Project is a loaded, validated project config plus its current state and the
// registry used to interpret service config. Profiles are loaded on demand.
type Project struct {
	Config   *config.Config
	State    *state.State
	Registry *service.Registry
	Paths    Paths
}

// Load reads and validates the project config, then loads (or initializes) the
// state. It returns ErrNotInitialized if the config is absent.
func Load(paths Paths, reg *service.Registry) (*Project, error) {
	cfg, err := config.Load(paths.Config)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrNotInitialized
		}
		return nil, err
	}
	if err := cfg.Validate(reg); err != nil {
		return nil, err
	}

	st, err := state.Load(paths.State)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			st = &state.State{}
		} else {
			return nil, err
		}
	}

	return &Project{Config: cfg, State: st, Registry: reg, Paths: paths}, nil
}

// Profiles lists the available profile names.
func (p *Project) Profiles() ([]string, error) {
	return profile.List(p.Paths.ProfilesDir)
}

// LoadProfile loads and validates the named profile against the project's
// service definitions.
func (p *Project) LoadProfile(name string) (*profile.Profile, error) {
	prof, err := profile.Load(p.Paths.ProfilePath(name))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("profile %q does not exist", name)
		}
		return nil, err
	}
	if err := prof.Validate(p.Registry, p.Config.ServiceNames()); err != nil {
		return nil, fmt.Errorf("profile %q: %w", name, err)
	}
	return prof, nil
}

// ActiveProfile resolves and loads the currently active profile. It returns an
// actionable error when no profile is active.
func (p *Project) ActiveProfile() (string, *profile.Profile, error) {
	name := p.State.ActiveProfile
	if name == "" {
		return "", nil, fmt.Errorf("no active profile; run `easy-infra use <profile>` first")
	}
	prof, err := p.LoadProfile(name)
	if err != nil {
		return "", nil, err
	}
	return name, prof, nil
}

// SetActiveProfile records name as the active profile after confirming the
// profile exists and is valid, then persists the state.
func (p *Project) SetActiveProfile(name string) error {
	if _, err := p.LoadProfile(name); err != nil {
		return err
	}
	p.State.ActiveProfile = name
	return p.State.Save(p.Paths.State)
}

// SaveProfile writes a profile to its conventional path.
func (p *Project) SaveProfile(name string, prof *profile.Profile) error {
	return prof.Save(p.Paths.ProfilePath(name))
}

// ProfileConfig returns the named profile's per-service environment config for
// display or editing. Unlike LoadProfile it does not validate, so a profile
// momentarily out of sync with the service definitions can still be opened and
// fixed. A missing profile is reported as an actionable error.
func (p *Project) ProfileConfig(name string) (map[string]service.Config, error) {
	prof, err := profile.Load(p.Paths.ProfilePath(name))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("profile %q does not exist", name)
		}
		return nil, err
	}
	return prof.Services, nil
}

// UpdateProfile replaces the named profile's per-service environment config,
// after validating it against the project's defined services, then saves it. It
// reports a missing profile as an actionable error.
func (p *Project) UpdateProfile(name string, services map[string]service.Config) error {
	if _, err := p.ProfileConfig(name); err != nil {
		return err
	}
	prof := &profile.Profile{Services: services}
	if err := prof.Validate(p.Registry, p.Config.ServiceNames()); err != nil {
		return fmt.Errorf("profile %q: %w", name, err)
	}
	return p.SaveProfile(name, prof)
}

// AddProfile scaffolds a new profile with default environment config for every
// service the project defines, then saves it. It errors if a profile with that
// name already exists.
func (p *Project) AddProfile(name string) (*profile.Profile, error) {
	path := p.Paths.ProfilePath(name)
	if _, err := profile.Load(path); err == nil {
		return nil, fmt.Errorf("profile %q already exists", name)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	prof, err := profile.Scaffold(p.Registry, p.Config.ServiceNames()...)
	if err != nil {
		return nil, err
	}
	if err := p.SaveProfile(name, prof); err != nil {
		return nil, err
	}
	return prof, nil
}

// ForkLocalProfile builds (or updates) the conventional "local" profile from a
// source profile, replacing the forked service's env with its localised form
// (localEnv) while leaving every other service as the source defines it. An
// existing local profile's already-localised services are preserved, so forking
// services one at a time accumulates them. The result is validated and saved,
// and its name (LocalProfile) is returned.
func (p *Project) ForkLocalProfile(source, svcName string, localEnv service.Config) (string, error) {
	// Load the source through LoadProfile so an invalid source is rejected before
	// we derive anything from it.
	src, err := p.LoadProfile(source)
	if err != nil {
		return "", err
	}

	services := make(map[string]service.Config, len(src.Services))
	for name, cfg := range src.Services {
		services[name] = cfg
	}
	// Preserve services already localised by a previous fork.
	if existing, err := profile.Load(p.Paths.ProfilePath(LocalProfile)); err == nil {
		for name, cfg := range existing.Services {
			if _, defined := services[name]; defined {
				services[name] = cfg
			}
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	services[svcName] = localEnv

	prof := &profile.Profile{Services: services}
	if err := prof.Validate(p.Registry, p.Config.ServiceNames()); err != nil {
		return "", fmt.Errorf("profile %q: %w", LocalProfile, err)
	}
	if err := p.SaveProfile(LocalProfile, prof); err != nil {
		return "", err
	}
	return LocalProfile, nil
}

// RemoveProfile deletes the named profile. It refuses to remove the active
// profile and reports a missing profile as an actionable error.
func (p *Project) RemoveProfile(name string) error {
	if p.State.ActiveProfile == name {
		return fmt.Errorf("cannot remove active profile %q; switch with `easy-infra use <other>` first", name)
	}
	if err := profile.Remove(p.Paths.ProfilePath(name)); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("profile %q does not exist", name)
		}
		return err
	}
	return nil
}
