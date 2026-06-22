// SQL template endpoints: CRUD over a workspace's saved, parameterized SQL
// scripts plus a run endpoint that renders a template's {{variables}} and
// executes the result against a profile's queryable service. Execution reuses
// the console's Querier path — a template is a saved, parameterized console
// query.
package server

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"regexp"
	"time"

	"github.com/minhnc/easy-infra/internal/project"
	"github.com/minhnc/easy-infra/internal/service"
	"github.com/minhnc/easy-infra/internal/sqltemplate"
)

// validTemplateName guards the template name taken from the request path/body.
// It mirrors validProfileName: a safe, predictable character set that is also
// path-segment friendly.
var validTemplateName = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// templateSummary is the JSON shape of a template in a list: everything but the
// SQL body, plus the variables parsed from that body.
type templateSummary struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Variables   []string `json:"variables"`
	UpdatedAt   string   `json:"updatedAt"`
}

// templateResponse is the JSON shape of a single template, including its SQL
// body and the variables parsed from it.
type templateResponse struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	SQL         string   `json:"sql"`
	Variables   []string `json:"variables"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
}

// templateRequest is the body for creating or updating a template. Name is
// ignored on update (the path identifies the template).
type templateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	SQL         string `json:"sql"`
}

// templateRunRequest is the body for running a template: which profile/service
// to run against, an optional logical-database override (Redis), and the values
// to substitute for the template's variables.
type templateRunRequest struct {
	Profile   string            `json:"profile"`
	Service   string            `json:"service"`
	DB        *int              `json:"db,omitempty"`
	Variables map[string]string `json:"variables"`
}

// handleListTemplates returns the workspace's templates (summaries). As with
// /api/status, an uninitialized workspace yields a 200 with an empty list.
func (s *Server) handleListTemplates(w http.ResponseWriter, _ *http.Request) {
	proj, err := s.activeProject()
	if err != nil {
		if errors.Is(err, project.ErrNotInitialized) {
			writeJSON(w, http.StatusOK, []templateSummary{})
			return
		}
		s.writeProjectError(w, err)
		return
	}
	tmpls, err := proj.Templates()
	if err != nil {
		s.writeProjectError(w, err)
		return
	}
	out := make([]templateSummary, 0, len(tmpls))
	for _, t := range tmpls {
		out = append(out, templateSummary{
			Name:        t.Name,
			Description: t.Description,
			Variables:   sqltemplate.Variables(t.SQL),
			UpdatedAt:   t.UpdatedAt.Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// handleGetTemplate returns a single template with its SQL body.
func (s *Server) handleGetTemplate(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	proj, err := s.activeProject()
	if err != nil {
		s.writeProjectError(w, err)
		return
	}
	t, err := proj.Template(name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, templateResponseFor(t))
}

// handleCreateTemplate creates a new template from the request body.
func (s *Server) handleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	var req templateRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if !validTemplateName.MatchString(req.Name) {
		writeError(w, http.StatusBadRequest, "template name must be non-empty and contain only letters, digits, '.', '_' or '-'")
		return
	}
	proj, err := s.activeProject()
	if err != nil {
		s.writeProjectError(w, err)
		return
	}
	t := sqltemplate.Template{Name: req.Name, Description: req.Description, SQL: req.SQL}
	if err := proj.CreateTemplate(t); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	// Re-read so the response carries the assigned timestamps.
	saved, err := proj.Template(req.Name)
	if err != nil {
		s.writeProjectError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, templateResponseFor(saved))
}

// handleUpdateTemplate replaces the named template's description and SQL.
func (s *Server) handleUpdateTemplate(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var req templateRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	proj, err := s.activeProject()
	if err != nil {
		s.writeProjectError(w, err)
		return
	}
	t := sqltemplate.Template{Name: name, Description: req.Description, SQL: req.SQL}
	if err := proj.UpdateTemplate(name, t); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	saved, err := proj.Template(name)
	if err != nil {
		s.writeProjectError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, templateResponseFor(saved))
}

// handleDeleteTemplate removes the named template.
func (s *Server) handleDeleteTemplate(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	proj, err := s.activeProject()
	if err != nil {
		s.writeProjectError(w, err)
		return
	}
	if err := proj.RemoveTemplate(name); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleRunTemplate renders the named template with the supplied variables and
// executes the result against the chosen profile/service. Like the console, a
// failing statement is an expected outcome: OK stays 200 with the reason in
// Error. A missing variable is a client error (the form is incomplete), so it
// returns 400 before any database work.
func (s *Server) handleRunTemplate(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var req templateRunRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	proj, err := s.activeProject()
	if err != nil {
		s.writeProjectError(w, err)
		return
	}
	rendered, err := proj.RenderTemplate(name, req.Variables)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	querier, spec, ok := s.resolveQuerierFor(w, proj, req.Profile, req.Service)
	if !ok {
		return
	}
	if req.DB != nil {
		if *req.DB < 0 {
			writeError(w, http.StatusBadRequest, "db must be a non-negative whole number")
			return
		}
		// Override the saved logical database without mutating the cached config,
		// mirroring handleConsoleQuery.
		env := make(service.Config, len(spec.Env)+1)
		maps.Copy(env, spec.Env)
		env["db"] = *req.DB
		spec.Env = env
	}

	ctx, cancel := context.WithTimeout(r.Context(), queryTimeout)
	defer cancel()
	start := time.Now()
	res, qerr := querier.Query(ctx, spec, rendered)
	elapsed := time.Since(start).Milliseconds()
	if qerr != nil {
		writeJSON(w, http.StatusOK, queryResponse{
			Columns: []string{}, Rows: [][]any{}, DurationMs: elapsed, Error: qerr.Error(),
		})
		return
	}
	writeJSON(w, http.StatusOK, queryResponse{
		Columns:    res.Columns,
		Rows:       res.Rows,
		RowCount:   res.RowCount,
		Command:    res.Command,
		Truncated:  res.Truncated,
		DurationMs: elapsed,
		Editable:   res.Editable,
	})
}

// resolveQuerierFor maps an explicit profile/service pair onto a console-capable
// service and the Spec for the profile's saved env. It mirrors resolveQuerier,
// which takes the pair from the request path; the run endpoint takes it from the
// body. On failure it writes the error response and returns ok=false.
func (s *Server) resolveQuerierFor(w http.ResponseWriter, proj *project.Project, profileName, svcID string) (service.Querier, service.Spec, bool) {
	if profileName == "" || svcID == "" {
		writeError(w, http.StatusBadRequest, "profile and service are required to run a template")
		return nil, service.Spec{}, false
	}
	services, err := proj.ProfileConfig(profileName)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return nil, service.Spec{}, false
	}
	entry, ok := services[svcID]
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("service %q is not defined in profile %q", svcID, profileName))
		return nil, service.Spec{}, false
	}
	svcType := entry.ResolveType(svcID)
	svc, ok := s.reg.Get(svcType)
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("unknown service %q", svcType))
		return nil, service.Spec{}, false
	}
	querier, ok := svc.(service.Querier)
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("service %q does not support running SQL", svcID))
		return nil, service.Spec{}, false
	}
	return querier, service.Spec{Profile: profileName, Env: entry.Config}, true
}

// templateResponseFor maps a stored template onto its single-template JSON
// shape, parsing the variables from its SQL body.
func templateResponseFor(t sqltemplate.Template) templateResponse {
	return templateResponse{
		Name:        t.Name,
		Description: t.Description,
		SQL:         t.SQL,
		Variables:   sqltemplate.Variables(t.SQL),
		CreatedAt:   t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   t.UpdatedAt.Format(time.RFC3339),
	}
}
