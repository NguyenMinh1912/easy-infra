package project

import (
	"fmt"
	"sort"

	"github.com/minhnc/easy-infra/internal/profile"
	"github.com/minhnc/easy-infra/internal/service"
	"github.com/minhnc/easy-infra/internal/store"
)

// exportVersion identifies the workspace export format. Bumping it lets the
// import side reject a payload it does not understand instead of silently
// importing the wrong shape.
const exportVersion = 1

// WorkspaceExport is the portable representation of a workspace: its profiles
// and the service instances they own, plus which profile was active. It is what
// the export endpoint serialises and the import endpoint consumes, so a user can
// move a workspace between machines or share it with a teammate.
type WorkspaceExport struct {
	Version       int             `json:"version"`
	Name          string          `json:"name"`
	ActiveProfile string          `json:"activeProfile"`
	Profiles      []ProfileExport `json:"profiles"`
}

// ProfileExport is one profile within an export: its name and its service
// instances.
type ProfileExport struct {
	Name     string          `json:"name"`
	Services []ServiceExport `json:"services"`
}

// ServiceExport is one service instance within an exported profile. It mirrors
// the API's profileServiceConfig shape so the format is human-readable and
// stable across the export/import round trip.
type ServiceExport struct {
	ID     string         `json:"id"`
	Type   string         `json:"type"`
	Name   string         `json:"name"`
	Config service.Config `json:"config"`
}

// ExportWorkspace reads the workspace with id and returns its portable
// representation: every profile and the service instances it owns, with the
// active profile recorded. An unknown workspace yields ErrNotInitialized.
func ExportWorkspace(st *store.Store, wsID int64) (WorkspaceExport, error) {
	proj, err := Open(st, nil, wsID)
	if err != nil {
		return WorkspaceExport{}, err
	}
	names, err := proj.Profiles()
	if err != nil {
		return WorkspaceExport{}, err
	}
	profiles := make([]ProfileExport, 0, len(names))
	for _, name := range names {
		services, err := proj.ProfileConfig(name)
		if err != nil {
			return WorkspaceExport{}, err
		}
		profiles = append(profiles, ProfileExport{
			Name:     name,
			Services: exportServices(services),
		})
	}
	return WorkspaceExport{
		Version:       exportVersion,
		Name:          proj.Workspace.Name,
		ActiveProfile: proj.Workspace.ActiveProfile,
		Profiles:      profiles,
	}, nil
}

// ImportWorkspace creates a new workspace from a previously exported payload. To
// avoid clobbering existing data it never reuses an id and resolves a unique
// name (appending " (n)" when the exported name is taken), returning the created
// workspace. Each imported service is validated against the registry so a
// malformed or unknown-service payload is rejected up front rather than
// producing a broken workspace.
func ImportWorkspace(st *store.Store, reg *service.Registry, exp WorkspaceExport) (store.Workspace, error) {
	if exp.Version != exportVersion {
		return store.Workspace{}, fmt.Errorf("unsupported export version %d (expected %d)", exp.Version, exportVersion)
	}
	if err := validateImport(reg, exp); err != nil {
		return store.Workspace{}, err
	}

	name, err := uniqueWorkspaceName(st, exp.Name)
	if err != nil {
		return store.Workspace{}, err
	}
	ws, err := st.CreateWorkspace(name)
	if err != nil {
		return store.Workspace{}, err
	}
	for _, p := range exp.Profiles {
		if err := st.CreateProfile(ws.ID, p.Name, importServices(p.Services)); err != nil {
			return store.Workspace{}, err
		}
	}
	// Only restore the active profile if it actually came over, so a workspace
	// exported with no active profile imports the same way.
	if exp.ActiveProfile != "" {
		if err := st.SetWorkspaceActiveProfile(ws.ID, exp.ActiveProfile); err != nil {
			return store.Workspace{}, err
		}
		ws.ActiveProfile = exp.ActiveProfile
	}
	return ws, nil
}

// validateImport checks the payload is coherent before any rows are written:
// every profile must be named and every service must be a type the registry
// knows with a valid config.
func validateImport(reg *service.Registry, exp WorkspaceExport) error {
	for _, p := range exp.Profiles {
		if p.Name == "" {
			return fmt.Errorf("imported profile is missing a name")
		}
		for _, sc := range p.Services {
			svcType := sc.Type
			if svcType == "" {
				svcType = sc.ID
			}
			svc, ok := reg.Get(svcType)
			if !ok {
				return fmt.Errorf("profile %q: unknown service %q", p.Name, svcType)
			}
			if err := service.ValidateConfig(svc, sc.Config); err != nil {
				return fmt.Errorf("profile %q service %q: %w", p.Name, sc.ID, err)
			}
		}
	}
	return nil
}

// exportServices maps a profile's stored service instances onto the export
// shape, sorted by id for a stable, diff-friendly payload.
func exportServices(services map[string]profile.ServiceEntry) []ServiceExport {
	ids := make([]string, 0, len(services))
	for id := range services {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]ServiceExport, 0, len(ids))
	for _, id := range ids {
		entry := services[id]
		out = append(out, ServiceExport{
			ID:     id,
			Type:   entry.ResolveType(id),
			Name:   entry.ResolveName(id),
			Config: entry.Config,
		})
	}
	return out
}

// importServices turns exported service instances back into the store's keyed
// map. Type and name fall back to the id, mirroring the update-profile handler.
func importServices(services []ServiceExport) map[string]profile.ServiceEntry {
	out := make(map[string]profile.ServiceEntry, len(services))
	for _, sc := range services {
		id := sc.ID
		if id == "" {
			id = sc.Name
		}
		svcType := sc.Type
		if svcType == "" {
			svcType = id
		}
		svcName := sc.Name
		if svcName == "" {
			svcName = id
		}
		out[id] = profile.ServiceEntry{Type: svcType, Name: svcName, Config: sc.Config}
	}
	return out
}

// uniqueWorkspaceName returns base if no workspace already uses it, otherwise
// "base (2)", "base (3)", … so an import never collides with an existing
// workspace.
func uniqueWorkspaceName(st *store.Store, base string) (string, error) {
	existing, err := st.ListWorkspaces()
	if err != nil {
		return "", err
	}
	taken := make(map[string]bool, len(existing))
	for _, w := range existing {
		taken[w.Name] = true
	}
	if !taken[base] {
		return base, nil
	}
	for n := 2; ; n++ {
		candidate := fmt.Sprintf("%s (%d)", base, n)
		if !taken[candidate] {
			return candidate, nil
		}
	}
}
