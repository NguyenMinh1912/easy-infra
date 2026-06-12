package server

import (
	"errors"
	"net/http"
	"regexp"

	"github.com/minhnc/easy-infra/internal/project"
)

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
