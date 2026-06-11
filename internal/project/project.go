// Package project ties together config, state, and the service registry into
// a single facade for the command layer. Commands depend on this package
// rather than re-implementing the load/validate/resolve sequence, keeping the
// cmd/ layer thin and the workflow logic in one place.
package project

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/minhnc/easy-infra/internal/config"
	"github.com/minhnc/easy-infra/internal/service"
	"github.com/minhnc/easy-infra/internal/state"
)

// ErrNotInitialized signals that the current folder has no easy-infra project.
// Commands surface it as a hint to run `easy-infra init`.
var ErrNotInitialized = errors.New("project not initialized")

// Paths locates a project's config and state files.
type Paths struct {
	Config string
	State  string
}

// DefaultPaths returns the conventional file locations for a project rooted at
// the current working directory.
func DefaultPaths() Paths {
	return Paths{Config: config.DefaultPath, State: state.DefaultPath}
}

// Project is a loaded, validated project: its config, its current state, and
// the registry used to interpret service configuration.
type Project struct {
	Config   *config.Config
	State    *state.State
	Registry *service.Registry
	Paths    Paths
}

// Load reads and validates the config, then loads (or initializes) the state.
// It returns ErrNotInitialized if the config is absent.
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

// ActiveProfile resolves the currently active profile from state and config.
// It returns an actionable error when no profile is active or the active
// profile no longer exists in config.
func (p *Project) ActiveProfile() (string, config.Profile, error) {
	name := p.State.ActiveProfile
	if name == "" {
		return "", config.Profile{}, fmt.Errorf("no active profile; run `easy-infra use <profile>` first")
	}
	profile, ok := p.Config.Profile(name)
	if !ok {
		return "", config.Profile{}, fmt.Errorf("active profile %q no longer exists in config", name)
	}
	return name, profile, nil
}

// SetActiveProfile records name as the active profile, validating that it
// exists in config, and persists the state.
func (p *Project) SetActiveProfile(name string) error {
	if _, ok := p.Config.Profile(name); !ok {
		return fmt.Errorf("unknown profile %q (available: %v)", name, p.Config.ProfileNames())
	}
	p.State.ActiveProfile = name
	return p.State.Save(p.Paths.State)
}
