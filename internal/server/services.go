package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/minhnc/easy-infra/internal/project"
	"github.com/minhnc/easy-infra/internal/service"
)

// serviceDefinition is the JSON shape of one project-level service definition.
type serviceDefinition struct {
	Name       string         `json:"name"`
	Definition service.Config `json:"definition"`
}

// servicesResponse is returned by GET /api/services. Initialized is false for a
// folder with no easy-infra project, in which case Services is empty.
type servicesResponse struct {
	Initialized bool                `json:"initialized"`
	Services    []serviceDefinition `json:"services"`
}

// catalogEntry describes a service easy-infra supports and the default
// definition used when adding it. The UI derives which entries are still
// addable by diffing against the defined services.
type catalogEntry struct {
	Name              string         `json:"name"`
	DefaultDefinition service.Config `json:"defaultDefinition"`
}

// catalogResponse is returned by GET /api/services/catalog.
type catalogResponse struct {
	Services []catalogEntry `json:"services"`
}

// serviceRequest is the request body for creating or updating a service.
type serviceRequest struct {
	Name       string         `json:"name"`
	Definition service.Config `json:"definition"`
}

// handleServiceCatalog lists every service easy-infra supports, from the
// registry. It does not require an initialized project.
func (s *Server) handleServiceCatalog(w http.ResponseWriter, _ *http.Request) {
	entries := make([]catalogEntry, 0)
	for _, name := range s.reg.Names() {
		svc, _ := s.reg.Get(name)
		entries = append(entries, catalogEntry{Name: name, DefaultDefinition: svc.DefaultDefinition()})
	}
	writeJSON(w, http.StatusOK, catalogResponse{Services: entries})
}

// handleListServices returns the project's service definitions. As with
// /api/status, an uninitialized folder yields a 200 with Initialized false.
func (s *Server) handleListServices(w http.ResponseWriter, _ *http.Request) {
	proj, err := project.Load(s.paths, s.reg)
	if err != nil {
		if errors.Is(err, project.ErrNotInitialized) {
			writeJSON(w, http.StatusOK, servicesResponse{Services: []serviceDefinition{}})
			return
		}
		s.writeProjectError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, servicesResponse{Initialized: true, Services: serviceDefinitions(proj)})
}

// handleCreateService adds a service to the project using its default
// definition, scaffolding it into existing profiles.
func (s *Server) handleCreateService(w http.ResponseWriter, r *http.Request) {
	var req serviceRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	proj, err := project.Load(s.paths, s.reg)
	if err != nil {
		s.writeProjectError(w, err)
		return
	}
	if err := proj.AddService(req.Name); err != nil {
		s.writeProjectError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, serviceDefinition{
		Name:       req.Name,
		Definition: proj.Config.Services[req.Name],
	})
}

// handleUpdateService replaces a service's project-level definition.
func (s *Server) handleUpdateService(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var req serviceRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	proj, err := project.Load(s.paths, s.reg)
	if err != nil {
		s.writeProjectError(w, err)
		return
	}
	if err := proj.UpdateService(name, req.Definition); err != nil {
		s.writeProjectError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, serviceDefinition{Name: name, Definition: req.Definition})
}

// handleDeleteService removes a service from the project and every profile.
func (s *Server) handleDeleteService(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	proj, err := project.Load(s.paths, s.reg)
	if err != nil {
		s.writeProjectError(w, err)
		return
	}
	if err := proj.RemoveService(name); err != nil {
		s.writeProjectError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// serviceDefinitions maps the project's services onto their JSON shape.
func serviceDefinitions(proj *project.Project) []serviceDefinition {
	defs := proj.Services()
	out := make([]serviceDefinition, 0, len(defs))
	for _, d := range defs {
		out = append(out, serviceDefinition{Name: d.Name, Definition: d.Definition})
	}
	return out
}

// decodeJSON parses the request body into v, writing a 400 and returning false
// on malformed input.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return false
	}
	return true
}

// writeProjectError maps a project/service error onto an HTTP status and a JSON
// error envelope, so the UI can surface an actionable message.
func (s *Server) writeProjectError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, project.ErrNotInitialized):
		status = http.StatusConflict
	case errors.Is(err, project.ErrUnknownService),
		errors.Is(err, project.ErrInvalidDefinition):
		status = http.StatusBadRequest
	case errors.Is(err, project.ErrServiceNotDefined):
		status = http.StatusNotFound
	case errors.Is(err, project.ErrServiceExists),
		errors.Is(err, project.ErrLastService):
		status = http.StatusConflict
	}
	writeError(w, status, err.Error())
}

// writeError writes a JSON error envelope: {"error": "..."}.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
