package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"time"

	profilepkg "github.com/minhnc/easy-infra/internal/profile"
	"github.com/minhnc/easy-infra/internal/project"
	"github.com/minhnc/easy-infra/internal/service"
)

// checkConnectionTimeout bounds a connection check so an unreachable host
// fails fast rather than hanging the request.
const checkConnectionTimeout = 10 * time.Second

// profilesResponse is the JSON shape returned by the /api/profiles endpoints:
// the project's profiles plus which one is active.
type profilesResponse struct {
	ActiveProfile string    `json:"activeProfile"`
	Profiles      []profile `json:"profiles"`
}

// profileRequest is the request body for creating a profile.
type profileRequest struct {
	Name string `json:"name"`
}

// profileConfigResponse is the JSON shape returned by GET /api/profiles/{name}:
// the profile's per-service environment config.
type profileConfigResponse struct {
	Name     string                 `json:"name"`
	Services []profileServiceConfig `json:"services"`
}

// profileServiceConfig describes one service instance within a profile: its
// unique id, its service type, its display name, and its environment config. A
// profile may hold several instances of the same type, so id (not type) is the
// stable identifier the UI routes and per-service endpoints use.
type profileServiceConfig struct {
	ID     string         `json:"id"`
	Type   string         `json:"type"`
	Name   string         `json:"name"`
	Config service.Config `json:"config"`
}

// profileConfigRequest is the request body for updating a profile's config.
type profileConfigRequest struct {
	Services []profileServiceConfig `json:"services"`
}

// checkConnectionRequest carries the (possibly unsaved) service env config the
// UI wants to test, so the user can verify a connection before saving it.
type checkConnectionRequest struct {
	Config service.Config `json:"config"`
}

// checkConnectionResponse reports whether the service was reachable. The probe
// failing is an expected outcome, not an HTTP error, so OK is false with the
// reason in Error rather than a non-2xx status.
type checkConnectionResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// validProfileName guards the profile name taken from the request path/body.
// Profile names become file names under .easy-infra/profiles, so we keep them
// to a safe, predictable character set and reject path-traversal attempts.
var validProfileName = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// handleListProfiles returns the project's profiles. As with /api/status, an
// uninitialized folder yields a 200 with an empty list.
func (s *Server) handleListProfiles(w http.ResponseWriter, _ *http.Request) {
	proj, err := project.Load(s.paths, s.reg)
	if err != nil {
		if errors.Is(err, project.ErrNotInitialized) {
			writeJSON(w, http.StatusOK, profilesResponse{Profiles: []profile{}})
			return
		}
		s.writeProjectError(w, err)
		return
	}
	s.writeProfiles(w, http.StatusOK, proj)
}

// handleCreateProfile adds a profile scaffolded with default config for every
// service the project defines.
func (s *Server) handleCreateProfile(w http.ResponseWriter, r *http.Request) {
	var req profileRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if !validProfileName.MatchString(req.Name) {
		writeError(w, http.StatusBadRequest, "profile name must be non-empty and contain only letters, digits, '.', '_' or '-'")
		return
	}
	proj, err := project.Load(s.paths, s.reg)
	if err != nil {
		s.writeProjectError(w, err)
		return
	}
	if _, err := proj.AddProfile(req.Name); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	s.writeProfiles(w, http.StatusCreated, proj)
}

// handleDeleteProfile removes a profile. The backend refuses to remove the
// active one and reports a missing profile.
func (s *Server) handleDeleteProfile(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	proj, err := project.Load(s.paths, s.reg)
	if err != nil {
		s.writeProjectError(w, err)
		return
	}
	if err := proj.RemoveProfile(name); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleGetProfile returns the named profile's per-service environment config.
func (s *Server) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	proj, err := project.Load(s.paths, s.reg)
	if err != nil {
		s.writeProjectError(w, err)
		return
	}
	services, err := proj.ProfileConfig(name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, profileConfigResponse{
		Name:     name,
		Services: profileServiceConfigs(services),
	})
}

// handleUpdateProfile replaces the named profile's per-service environment
// config, validating it against the project's defined services.
func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var req profileConfigRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	proj, err := project.Load(s.paths, s.reg)
	if err != nil {
		s.writeProjectError(w, err)
		return
	}
	services := make(map[string]profilepkg.ServiceEntry, len(req.Services))
	for _, sc := range req.Services {
		// Fall back to the id for an absent type/name so a client may post a
		// minimal {id, config} block, matching the on-disk default behaviour.
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
		services[id] = profilepkg.ServiceEntry{Type: svcType, Name: svcName, Config: sc.Config}
	}
	if err := proj.UpdateProfile(name, services); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, profileConfigResponse{
		Name:     name,
		Services: profileServiceConfigs(services),
	})
}

// serviceNameRequest is the request body for adding a service to a profile. It
// names the service type to add and, optionally, a display name and a starting
// config. Type defaults from the legacy `name` field so an older client posting
// just a service type still works; name and config fall back to the service's
// defaults when omitted.
type serviceNameRequest struct {
	Type   string         `json:"type"`
	Name   string         `json:"name"`
	Config service.Config `json:"config"`
}

// serviceConfigRequest is the request body for updating a single service
// instance within a profile: its config and, optionally, a new display name.
type serviceConfigRequest struct {
	Name   string         `json:"name"`
	Config service.Config `json:"config"`
}

// handleCreateProfileService adds an instance of a service type to a profile,
// then returns the profile's updated service config. A profile may hold several
// instances of the same type; the backend assigns each a unique id.
func (s *Server) handleCreateProfileService(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var req serviceNameRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	// Accept the service type from `type`, falling back to the legacy `name`
	// field which used to carry it.
	svcType := req.Type
	if svcType == "" {
		svcType = req.Name
	}
	// A display name only defaults when type was given explicitly; when the
	// legacy single-field form is used, `name` is the type, not a label.
	displayName := ""
	if req.Type != "" {
		displayName = req.Name
	}
	proj, err := project.Load(s.paths, s.reg)
	if err != nil {
		s.writeProjectError(w, err)
		return
	}
	if _, err := proj.AddProfileService(name, svcType, displayName, req.Config); err != nil {
		s.writeProjectError(w, err)
		return
	}
	s.writeProfileConfig(w, http.StatusCreated, proj, name)
}

// handleUpdateProfileService replaces a single service instance's config within
// a profile and optionally renames it. The {service} path segment is the
// instance id.
func (s *Server) handleUpdateProfileService(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	svcID := r.PathValue("service")
	var req serviceConfigRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	proj, err := project.Load(s.paths, s.reg)
	if err != nil {
		s.writeProjectError(w, err)
		return
	}
	if err := proj.UpdateProfileService(name, svcID, req.Name, req.Config); err != nil {
		s.writeProjectError(w, err)
		return
	}
	s.writeProfileConfig(w, http.StatusOK, proj, name)
}

// handleDeleteProfileService removes a service instance from a profile. The
// {service} path segment is the instance id.
func (s *Server) handleDeleteProfileService(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	svcID := r.PathValue("service")
	proj, err := project.Load(s.paths, s.reg)
	if err != nil {
		s.writeProjectError(w, err)
		return
	}
	if err := proj.RemoveProfileService(name, svcID); err != nil {
		s.writeProjectError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// writeProfileConfig re-reads a profile and writes its per-service config at the
// given status, so a mutation can echo the resulting state back to the UI.
func (s *Server) writeProfileConfig(w http.ResponseWriter, status int, proj *project.Project, name string) {
	services, err := proj.ProfileConfig(name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, status, profileConfigResponse{
		Name:     name,
		Services: profileServiceConfigs(services),
	})
}

// handleActivateProfile sets the named profile as the active one.
func (s *Server) handleActivateProfile(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	proj, err := project.Load(s.paths, s.reg)
	if err != nil {
		s.writeProjectError(w, err)
		return
	}
	if err := proj.SetActiveProfile(name); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	s.writeProfiles(w, http.StatusOK, proj)
}

// handleCheckConnection probes a service with the env config posted in the
// request body, letting the UI verify connectivity for the config currently in
// the form without first saving it. The profile name scopes the check (it is
// recorded on the Spec) but the config comes from the body, not disk.
func (s *Server) handleCheckConnection(w http.ResponseWriter, r *http.Request) {
	profileName := r.PathValue("name")
	svcName := r.PathValue("service")
	var req checkConnectionRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	svc, ok := s.reg.Get(svcName)
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("unknown service %q", svcName))
		return
	}
	if err := svc.ValidateEnv(req.Config); err != nil {
		writeJSON(w, http.StatusOK, checkConnectionResponse{Error: err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), checkConnectionTimeout)
	defer cancel()
	if err := svc.Health(ctx, service.Spec{Profile: profileName, Env: req.Config}); err != nil {
		writeJSON(w, http.StatusOK, checkConnectionResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, checkConnectionResponse{OK: true})
}

// writeProfiles writes the project's current profile list at the given status.
func (s *Server) writeProfiles(w http.ResponseWriter, status int, proj *project.Project) {
	profiles, err := listProfiles(proj)
	if err != nil {
		s.writeProjectError(w, err)
		return
	}
	writeJSON(w, status, profilesResponse{
		ActiveProfile: proj.State.ActiveProfile,
		Profiles:      profiles,
	})
}

// profileServiceConfigs maps a profile's service instances onto their JSON
// shape, sorted by id for a stable response. Type and name fall back to the id
// for an entry that stores neither (the backward-compatible single-instance
// case), so the UI always receives a resolved type and name.
func profileServiceConfigs(services map[string]profilepkg.ServiceEntry) []profileServiceConfig {
	ids := make([]string, 0, len(services))
	for id := range services {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]profileServiceConfig, 0, len(ids))
	for _, id := range ids {
		entry := services[id]
		out = append(out, profileServiceConfig{
			ID:     id,
			Type:   entry.ResolveType(id),
			Name:   entry.ResolveName(id),
			Config: entry.Config,
		})
	}
	return out
}

// listProfiles builds the API profile list for a loaded project, flagging the
// active one.
func listProfiles(proj *project.Project) ([]profile, error) {
	names, err := proj.Profiles()
	if err != nil {
		return nil, err
	}
	active := proj.State.ActiveProfile
	profiles := make([]profile, 0, len(names))
	for _, name := range names {
		profiles = append(profiles, profile{Name: name, Active: name == active})
	}
	return profiles, nil
}
