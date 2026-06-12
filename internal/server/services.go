package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/minhnc/easy-infra/internal/project"
	"github.com/minhnc/easy-infra/internal/service"
)

// catalogEntry describes a service easy-infra supports and the default config
// used when adding it to a profile. The UI derives which entries are still
// addable by diffing against a profile's services.
type catalogEntry struct {
	Name          string         `json:"name"`
	DefaultConfig service.Config `json:"defaultConfig"`
}

// catalogResponse is returned by GET /api/services/catalog.
type catalogResponse struct {
	Services []catalogEntry `json:"services"`
}

// handleServiceCatalog lists every service easy-infra supports, from the
// registry. It does not require an initialized project.
func (s *Server) handleServiceCatalog(w http.ResponseWriter, _ *http.Request) {
	entries := make([]catalogEntry, 0)
	for _, name := range s.reg.Names() {
		svc, _ := s.reg.Get(name)
		entries = append(entries, catalogEntry{Name: name, DefaultConfig: service.DefaultConfig(svc)})
	}
	writeJSON(w, http.StatusOK, catalogResponse{Services: entries})
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
