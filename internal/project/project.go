// Package project ties the store, the service registry, and a workspace into a
// single facade for the HTTP server. Handlers depend on this package rather than
// re-implementing the load/validate/resolve sequence, keeping request handling
// thin and the workflow logic in one place.
//
// A Project is scoped to one workspace. Storage lives in package store (a single
// SQLite database); this package layers validation and the profile/service
// workflows on top of it.
package project

import (
	"errors"
	"fmt"

	"github.com/minhnc/easy-infra/internal/profile"
	"github.com/minhnc/easy-infra/internal/service"
	"github.com/minhnc/easy-infra/internal/store"
)

// ErrNotInitialized signals that there is no workspace to operate on (none
// active, or the requested one is gone). Handlers surface it as "create a
// workspace first".
var ErrNotInitialized = errors.New("no active workspace")

// LocalProfile is the conventional name of the profile that holds a workspace's
// locally-forked services. "fork to local" writes the forked service's localised
// env here so it appears in the sidebar as a managed profile.
const LocalProfile = "local"

// DefaultProfile is the profile a freshly created workspace starts with.
const DefaultProfile = "default"

// DefaultServices is the service set a freshly scaffolded profile starts with,
// so a new profile is immediately valid (a profile must define at least one
// service).
var DefaultServices = []string{"postgres", "redis"}

// Project is a workspace plus the store and registry needed to read and validate
// its profiles and services.
type Project struct {
	Store     *store.Store
	Registry  *service.Registry
	Workspace store.Workspace
}

// Open loads the workspace with id and returns a Project scoped to it. An
// unknown workspace yields ErrNotInitialized.
func Open(st *store.Store, reg *service.Registry, wsID int64) (*Project, error) {
	ws, err := st.GetWorkspace(wsID)
	if err != nil {
		if errors.Is(err, store.ErrWorkspaceNotFound) {
			return nil, ErrNotInitialized
		}
		return nil, err
	}
	return &Project{Store: st, Registry: reg, Workspace: ws}, nil
}

// CreateWorkspace creates a workspace, scaffolds its default profile (owning the
// conventional starter services), and records that profile as active — so a new
// workspace is immediately usable. It does not make the workspace itself active;
// the caller decides that.
func CreateWorkspace(st *store.Store, reg *service.Registry, name string) (store.Workspace, error) {
	prof, err := profile.Scaffold(reg, DefaultServices...)
	if err != nil {
		return store.Workspace{}, err
	}
	ws, err := st.CreateWorkspace(name)
	if err != nil {
		return store.Workspace{}, err
	}
	if err := st.CreateProfile(ws.ID, DefaultProfile, prof.Services); err != nil {
		return store.Workspace{}, err
	}
	if err := st.SetWorkspaceActiveProfile(ws.ID, DefaultProfile); err != nil {
		return store.Workspace{}, err
	}
	ws.ActiveProfile = DefaultProfile
	return ws, nil
}

// ActiveProfileName returns the workspace's active profile name ("" when none).
func (p *Project) ActiveProfileName() string {
	return p.Workspace.ActiveProfile
}

// Profiles lists the workspace's profile names, sorted.
func (p *Project) Profiles() ([]string, error) {
	return p.Store.ListProfiles(p.Workspace.ID)
}

// LoadProfile loads and validates the named profile, checking each service it
// owns against the registry.
func (p *Project) LoadProfile(name string) (*profile.Profile, error) {
	services, err := p.ProfileConfig(name)
	if err != nil {
		return nil, err
	}
	prof := &profile.Profile{Services: services}
	if err := prof.Validate(p.Registry); err != nil {
		return nil, fmt.Errorf("profile %q: %w", name, err)
	}
	return prof, nil
}

// ActiveProfile resolves and loads the currently active profile. It returns an
// actionable error when no profile is active.
func (p *Project) ActiveProfile() (string, *profile.Profile, error) {
	name := p.Workspace.ActiveProfile
	if name == "" {
		return "", nil, fmt.Errorf("no active profile; choose one first")
	}
	prof, err := p.LoadProfile(name)
	if err != nil {
		return "", nil, err
	}
	return name, prof, nil
}

// SetActiveProfile records name as the active profile after confirming it exists
// and is valid, then persists it.
func (p *Project) SetActiveProfile(name string) error {
	if _, err := p.LoadProfile(name); err != nil {
		return err
	}
	if err := p.Store.SetWorkspaceActiveProfile(p.Workspace.ID, name); err != nil {
		return err
	}
	p.Workspace.ActiveProfile = name
	return nil
}

// ProfileConfig returns the named profile's per-service config for display or
// editing. Unlike LoadProfile it does not validate, so a profile momentarily out
// of sync with the service definitions can still be opened and fixed. A missing
// profile is reported as an actionable error.
func (p *Project) ProfileConfig(name string) (map[string]profile.ServiceEntry, error) {
	services, err := p.Store.ProfileServices(p.Workspace.ID, name)
	if err != nil {
		if errors.Is(err, store.ErrProfileNotFound) {
			return nil, fmt.Errorf("profile %q does not exist", name)
		}
		return nil, err
	}
	return services, nil
}

// UpdateProfile replaces the named profile's per-service config, after
// validating each service against the registry, then saves it. It reports a
// missing profile as an actionable error.
func (p *Project) UpdateProfile(name string, services map[string]profile.ServiceEntry) error {
	if _, err := p.ProfileConfig(name); err != nil {
		return err
	}
	prof := &profile.Profile{Services: services}
	if err := prof.Validate(p.Registry); err != nil {
		return fmt.Errorf("profile %q: %w", name, err)
	}
	return p.Store.ReplaceProfileServices(p.Workspace.ID, name, services)
}

// AddProfile scaffolds a new profile with default config for the conventional
// starter services, then saves it. It errors if a profile with that name already
// exists.
func (p *Project) AddProfile(name string) (*profile.Profile, error) {
	exists, err := p.Store.ProfileExists(p.Workspace.ID, name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("profile %q already exists", name)
	}
	prof, err := profile.Scaffold(p.Registry, DefaultServices...)
	if err != nil {
		return nil, err
	}
	if err := p.Store.CreateProfile(p.Workspace.ID, name, prof.Services); err != nil {
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
func (p *Project) ForkLocalProfile(source, svcID string, localEnv service.Config) (string, error) {
	// Load the source through LoadProfile so an invalid source is rejected before
	// we derive anything from it.
	src, err := p.LoadProfile(source)
	if err != nil {
		return "", err
	}

	services := make(map[string]profile.ServiceEntry, len(src.Services))
	for id, entry := range src.Services {
		services[id] = entry
	}
	// Preserve services already localised by a previous fork.
	localExists, err := p.Store.ProfileExists(p.Workspace.ID, LocalProfile)
	if err != nil {
		return "", err
	}
	if localExists {
		existing, err := p.ProfileConfig(LocalProfile)
		if err != nil {
			return "", err
		}
		for id, entry := range existing {
			if _, defined := services[id]; defined {
				services[id] = entry
			}
		}
	}
	// Overlay the localised connection onto the source's block so the forked
	// service keeps its definition fields (e.g. version) while pointing at the
	// local container, preserving its type and name.
	srcEntry := src.Services[svcID]
	forkedCfg := service.Config{}
	for k, v := range srcEntry.Config {
		forkedCfg[k] = v
	}
	for k, v := range localEnv {
		forkedCfg[k] = v
	}
	services[svcID] = profile.ServiceEntry{Type: srcEntry.Type, Name: srcEntry.Name, Config: forkedCfg}

	prof := &profile.Profile{Services: services}
	if err := prof.Validate(p.Registry); err != nil {
		return "", fmt.Errorf("profile %q: %w", LocalProfile, err)
	}
	if localExists {
		if err := p.Store.ReplaceProfileServices(p.Workspace.ID, LocalProfile, services); err != nil {
			return "", err
		}
	} else if err := p.Store.CreateProfile(p.Workspace.ID, LocalProfile, services); err != nil {
		return "", err
	}
	return LocalProfile, nil
}

// RemoveProfile deletes the named profile. It refuses to remove the active
// profile and reports a missing profile as an actionable error.
func (p *Project) RemoveProfile(name string) error {
	if p.Workspace.ActiveProfile == name {
		return fmt.Errorf("cannot remove active profile %q; switch to another first", name)
	}
	if err := p.Store.RemoveProfile(p.Workspace.ID, name); err != nil {
		if errors.Is(err, store.ErrProfileNotFound) {
			return fmt.Errorf("profile %q does not exist", name)
		}
		return err
	}
	return nil
}
