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
