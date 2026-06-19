// Console endpoints: execute ad-hoc statements against a profile's service and
// describe its schema for the UI's SQL editor autocomplete. Only services
// implementing service.Querier (postgres today) support a console.
package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/minhnc/easy-infra/internal/service"
)

// queryTimeout bounds one console statement; long-running SQL fails with a
// clear error instead of hanging the request.
const queryTimeout = 30 * time.Second

// schemaTimeout bounds schema introspection. It is shorter than queryTimeout:
// autocomplete is an enhancement, so an unreachable database should give up
// quickly and let the editor fall back to keyword completion.
const schemaTimeout = 10 * time.Second

// queryRequest carries the statement to execute.
type queryRequest struct {
	SQL string `json:"sql"`
}

// queryResponse is the JSON shape of a console execution. Like the connection
// check, a failing statement is an expected outcome: OK stays 200 and the
// reason lands in Error with the result fields empty.
type queryResponse struct {
	Columns    []string `json:"columns"`
	Rows       [][]any  `json:"rows"`
	RowCount   int      `json:"rowCount"`
	Command    string   `json:"command"`
	Truncated  bool     `json:"truncated"`
	DurationMs int64    `json:"durationMs"`
	Error      string   `json:"error,omitempty"`
}

// schemaResponse is the JSON shape of a schema introspection. Introspection
// failing (e.g. database unreachable) is likewise reported in Error on a 200,
// so the UI can degrade to keyword-only completion.
type schemaResponse struct {
	Tables []service.TableInfo `json:"tables"`
	// CurrentSchema is the connection's search_path-resolved schema, so the
	// editor can default its suggestions to where statements execute.
	CurrentSchema string `json:"currentSchema"`
	Error         string `json:"error,omitempty"`
}

// handleConsoleQuery executes one statement against the named profile's
// service. The env comes from the saved profile config on disk — the console
// always talks to what the profile actually points at. There is deliberately
// no statement filtering: this is a dev tool and the database is the user's;
// the cleanable flag only governs Clean.
func (s *Server) handleConsoleQuery(w http.ResponseWriter, r *http.Request) {
	var req queryRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.SQL) == "" {
		writeError(w, http.StatusBadRequest, "sql must not be empty")
		return
	}
	querier, spec, ok := s.resolveQuerier(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), queryTimeout)
	defer cancel()
	start := time.Now()
	res, err := querier.Query(ctx, spec, req.SQL)
	elapsed := time.Since(start).Milliseconds()
	if err != nil {
		writeJSON(w, http.StatusOK, queryResponse{
			Columns: []string{}, Rows: [][]any{}, DurationMs: elapsed, Error: err.Error(),
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
	})
}

// handleConsoleSchema describes the named profile's service for autocomplete.
func (s *Server) handleConsoleSchema(w http.ResponseWriter, r *http.Request) {
	querier, spec, ok := s.resolveQuerier(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), schemaTimeout)
	defer cancel()
	info, err := querier.Schema(ctx, spec)
	if err != nil {
		writeJSON(w, http.StatusOK, schemaResponse{Tables: []service.TableInfo{}, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, schemaResponse{Tables: info.Tables, CurrentSchema: info.CurrentSchema})
}

// resolveQuerier maps the {name}/{service} path onto a console-capable service
// and the Spec for the profile's saved env. On failure it writes the error
// response and returns ok=false.
func (s *Server) resolveQuerier(w http.ResponseWriter, r *http.Request) (service.Querier, service.Spec, bool) {
	profileName := r.PathValue("name")
	svcID := r.PathValue("service")

	proj, err := s.activeProject()
	if err != nil {
		s.writeProjectError(w, err)
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
		writeError(w, http.StatusBadRequest, fmt.Sprintf("service %q does not support a console", svcID))
		return nil, service.Spec{}, false
	}
	return querier, service.Spec{Profile: profileName, Env: entry.Config}, true
}
