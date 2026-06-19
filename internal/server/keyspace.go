// Key-browser endpoints: list a profile's Redis keyspace and read individual
// key values, for the UI's service detail page. Only services implementing
// service.KeyBrowser (redis today) support browsing keys.
package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/minhnc/easy-infra/internal/service"
)

// keyspaceTimeout bounds one keyspace request so an unreachable server fails
// with a clear error instead of hanging the page. It matches browseTimeout.
const keyspaceTimeout = 15 * time.Second

// databasesResponse is the JSON shape of the logical-database count. Like the
// console and browser, an unreachable server is an expected outcome: OK stays
// 200 and the reason lands in Error with Count zero.
type databasesResponse struct {
	Count int    `json:"count"`
	Error string `json:"error,omitempty"`
}

// keysResponse is the JSON shape of one keyspace scan page.
type keysResponse struct {
	Keys   []service.KeyEntry `json:"keys"`
	Cursor uint64             `json:"cursor"`
	Error  string             `json:"error,omitempty"`
}

// handleKeyspaceDatabases reports the named profile's Redis logical-database
// count, for the UI's database selector.
func (s *Server) handleKeyspaceDatabases(w http.ResponseWriter, r *http.Request) {
	browser, spec, ok := s.resolveKeyBrowser(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), keyspaceTimeout)
	defer cancel()
	count, err := browser.Databases(ctx, spec)
	if err != nil {
		writeJSON(w, http.StatusOK, databasesResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, databasesResponse{Count: count})
}

// handleKeyspaceKeys lists one SCAN page of the named profile's keyspace. The
// `db` query selects the logical database, `pattern` filters keys (defaults to
// "*"), and `cursor` continues a previous scan.
func (s *Server) handleKeyspaceKeys(w http.ResponseWriter, r *http.Request) {
	browser, spec, ok := s.resolveKeyBrowser(w, r)
	if !ok {
		return
	}
	db, ok := parseDBParam(w, r)
	if !ok {
		return
	}
	pattern := r.URL.Query().Get("pattern")
	cursor, err := parseCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "cursor must be a whole number")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), keyspaceTimeout)
	defer cancel()
	page, err := browser.Keys(ctx, spec, db, pattern, cursor)
	if err != nil {
		writeJSON(w, http.StatusOK, keysResponse{Keys: []service.KeyEntry{}, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, keysResponse{Keys: page.Keys, Cursor: page.Cursor})
}

// handleKeyspaceValue reads one key's value from the named profile's keyspace.
// The `db` query selects the logical database and `key` names the key.
func (s *Server) handleKeyspaceValue(w http.ResponseWriter, r *http.Request) {
	browser, spec, ok := s.resolveKeyBrowser(w, r)
	if !ok {
		return
	}
	db, ok := parseDBParam(w, r)
	if !ok {
		return
	}
	key := r.URL.Query().Get("key")
	if key == "" {
		writeError(w, http.StatusBadRequest, "key must not be empty")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), keyspaceTimeout)
	defer cancel()
	value, err := browser.Value(ctx, spec, db, key)
	if err != nil {
		writeJSON(w, http.StatusOK, valueResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, valueResponse{KeyValue: value})
}

// valueResponse wraps a key value with the same 200-on-logical-error convention
// as the other keyspace endpoints.
type valueResponse struct {
	*service.KeyValue
	Error string `json:"error,omitempty"`
}

// parseDBParam reads the `db` query as a non-negative integer, defaulting to 0
// when absent. On a malformed value it writes a 400 and returns ok=false.
func parseDBParam(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := r.URL.Query().Get("db")
	if raw == "" {
		return 0, true
	}
	db, err := strconv.Atoi(raw)
	if err != nil || db < 0 {
		writeError(w, http.StatusBadRequest, "db must be a non-negative whole number")
		return 0, false
	}
	return db, true
}

// parseCursor reads a SCAN cursor, treating an empty value as the start (0).
func parseCursor(raw string) (uint64, error) {
	if raw == "" {
		return 0, nil
	}
	return strconv.ParseUint(raw, 10, 64)
}

// resolveKeyBrowser maps the {name}/{service} path onto a key-browse-capable
// service and the Spec for the profile's saved env. On failure it writes the
// error response and returns ok=false. It mirrors resolveBrowser.
func (s *Server) resolveKeyBrowser(w http.ResponseWriter, r *http.Request) (service.KeyBrowser, service.Spec, bool) {
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
	browser, ok := svc.(service.KeyBrowser)
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("service %q does not support a key browser", svcID))
		return nil, service.Spec{}, false
	}
	return browser, service.Spec{Profile: profileName, Env: entry.Config}, true
}
