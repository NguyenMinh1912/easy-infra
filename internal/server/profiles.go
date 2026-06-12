package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"time"

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

// profileServiceConfig pairs a service name with its environment config within
// a profile.
type profileServiceConfig struct {
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
	services := make(map[string]service.Config, len(req.Services))
	for _, sc := range req.Services {
		services[sc.Name] = sc.Config
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

// serviceNameRequest is the request body for adding a service to a profile.
type serviceNameRequest struct {
	Name string `json:"name"`
}

// serviceConfigRequest is the request body for updating a single service's
// config within a profile.
type serviceConfigRequest struct {
	Config service.Config `json:"config"`
}

// handleCreateProfileService adds a service to a profile using its default
// config, then returns the profile's updated service config.
func (s *Server) handleCreateProfileService(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var req serviceNameRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	proj, err := project.Load(s.paths, s.reg)
	if err != nil {
		s.writeProjectError(w, err)
		return
	}
	if err := proj.AddProfileService(name, req.Name); err != nil {
		s.writeProjectError(w, err)
		return
	}
	s.writeProfileConfig(w, http.StatusCreated, proj, name)
}

// handleUpdateProfileService replaces a single service's config within a
// profile.
func (s *Server) handleUpdateProfileService(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	svcName := r.PathValue("service")
	var req serviceConfigRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	proj, err := project.Load(s.paths, s.reg)
	if err != nil {
		s.writeProjectError(w, err)
		return
	}
	if err := proj.UpdateProfileService(name, svcName, req.Config); err != nil {
		s.writeProjectError(w, err)
		return
	}
	s.writeProfileConfig(w, http.StatusOK, proj, name)
}

// handleDeleteProfileService removes a service from a profile.
func (s *Server) handleDeleteProfileService(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	svcName := r.PathValue("service")
	proj, err := project.Load(s.paths, s.reg)
	if err != nil {
		s.writeProjectError(w, err)
		return
	}
	if err := proj.RemoveProfileService(name, svcName); err != nil {
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

// profileServiceConfigs maps a profile's per-service env config onto its JSON
// shape, sorted by service name for a stable response.
func profileServiceConfigs(services map[string]service.Config) []profileServiceConfig {
	names := make([]string, 0, len(services))
	for name := range services {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]profileServiceConfig, 0, len(names))
	for _, name := range names {
		out = append(out, profileServiceConfig{Name: name, Config: services[name]})
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
