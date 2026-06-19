// Package server exposes the project over HTTP: a small JSON API describing the
// project's profiles and services, plus the embedded single-page UI. It is the
// backend behind `easy-infra serve`, keeping all request handling out of the
// thin cmd/ layer.
package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/minhnc/easy-infra/internal/project"
	"github.com/minhnc/easy-infra/internal/service"
	"github.com/minhnc/easy-infra/internal/store"
)

// Server answers API requests and serves the embedded UI. All data lives in the
// central SQLite store; handlers open the active workspace fresh on each request
// so the API always reflects what is persisted. The user creates and switches
// workspaces from the UI.
type Server struct {
	reg     *service.Registry
	ui      fs.FS
	store   *store.Store
	backups *backupManager
}

// New builds a Server from the injected service registry, the store, and the
// embedded UI filesystem (see the ui package).
func New(reg *service.Registry, st *store.Store, ui fs.FS) *Server {
	return &Server{reg: reg, store: st, ui: ui, backups: newBackupManager()}
}

// activeProject opens the active workspace as a Project. When no workspace is
// active it returns project.ErrNotInitialized, so handlers behave as "not
// initialized" (the UI then offers to create a workspace).
func (s *Server) activeProject() (*project.Project, error) {
	ws, ok, err := s.store.ActiveWorkspace()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, project.ErrNotInitialized
	}
	return project.Open(s.store, s.reg, ws.ID)
}

// Handler returns the HTTP handler tree: the JSON API under /api and the SPA on
// everything else.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("GET /api/workspaces", s.handleListWorkspaces)
	mux.HandleFunc("POST /api/workspaces", s.handleCreateWorkspace)
	mux.HandleFunc("PUT /api/workspaces/{id}", s.handleRenameWorkspace)
	mux.HandleFunc("POST /api/workspaces/activate", s.handleActivateWorkspace)
	mux.HandleFunc("DELETE /api/workspaces/{id}", s.handleRemoveWorkspace)
	mux.HandleFunc("GET /api/profiles", s.handleListProfiles)
	mux.HandleFunc("POST /api/profiles", s.handleCreateProfile)
	mux.HandleFunc("GET /api/profiles/{name}", s.handleGetProfile)
	mux.HandleFunc("PUT /api/profiles/{name}", s.handleUpdateProfile)
	mux.HandleFunc("DELETE /api/profiles/{name}", s.handleDeleteProfile)
	mux.HandleFunc("POST /api/profiles/{name}/activate", s.handleActivateProfile)
	mux.HandleFunc("POST /api/profiles/{name}/services", s.handleCreateProfileService)
	mux.HandleFunc("PUT /api/profiles/{name}/services/{service}", s.handleUpdateProfileService)
	mux.HandleFunc("DELETE /api/profiles/{name}/services/{service}", s.handleDeleteProfileService)
	mux.HandleFunc("POST /api/profiles/{name}/services/{service}/check", s.handleCheckConnection)
	mux.HandleFunc("POST /api/profiles/{name}/services/{service}/query", s.handleConsoleQuery)
	mux.HandleFunc("GET /api/profiles/{name}/services/{service}/schema", s.handleConsoleSchema)
	mux.HandleFunc("PATCH /api/profiles/{name}/services/{service}/row", s.handleRowUpdate)
	mux.HandleFunc("DELETE /api/profiles/{name}/services/{service}/row", s.handleRowDelete)
	mux.HandleFunc("GET /api/profiles/{name}/services/{service}/buckets", s.handleBrowseBuckets)
	mux.HandleFunc("GET /api/profiles/{name}/services/{service}/objects", s.handleBrowseObjects)
	mux.HandleFunc("GET /api/profiles/{name}/services/{service}/objects/archive", s.handleBrowseArchive)
	mux.HandleFunc("DELETE /api/profiles/{name}/services/{service}/objects", s.handleBrowseDelete)
	mux.HandleFunc("GET /api/profiles/{name}/services/{service}/object", s.handleBrowseObject)
	mux.HandleFunc("PUT /api/profiles/{name}/services/{service}/object", s.handleBrowseUpload)
	mux.HandleFunc("GET /api/profiles/{name}/services/{service}/databases", s.handleKeyspaceDatabases)
	mux.HandleFunc("GET /api/profiles/{name}/services/{service}/keys", s.handleKeyspaceKeys)
	mux.HandleFunc("GET /api/profiles/{name}/services/{service}/key", s.handleKeyspaceValue)
	mux.HandleFunc("GET /api/profiles/{name}/services/{service}/health", s.handleCloudHealth)
	mux.HandleFunc("GET /api/profiles/{name}/services/{service}/queues", s.handleCloudQueues)
	mux.HandleFunc("POST /api/profiles/{name}/services/{service}/queues", s.handleCloudCreateQueue)
	mux.HandleFunc("DELETE /api/profiles/{name}/services/{service}/queues", s.handleCloudDeleteQueue)
	mux.HandleFunc("POST /api/profiles/{name}/services/{service}/queues/purge", s.handleCloudPurgeQueue)
	mux.HandleFunc("GET /api/profiles/{name}/services/{service}/queues/messages", s.handleCloudQueueMessages)
	mux.HandleFunc("GET /api/profiles/{name}/services/{service}/identities", s.handleCloudIdentities)
	mux.HandleFunc("POST /api/profiles/{name}/services/{service}/identities", s.handleCloudCreateIdentity)
	mux.HandleFunc("DELETE /api/profiles/{name}/services/{service}/identities", s.handleCloudDeleteIdentity)
	mux.HandleFunc("GET /api/profiles/{name}/services/{service}/messages", s.handleCloudMessages)
	mux.HandleFunc("GET /api/services/catalog", s.handleServiceCatalog)
	mux.HandleFunc("POST /api/services/{name}/backup", s.handleStartBackup)
	mux.HandleFunc("GET /api/services/{name}/backup-options", s.handleBackupOptions)
	mux.HandleFunc("GET /api/services/{name}/snapshots", s.handleListSnapshots)
	mux.HandleFunc("POST /api/services/{name}/apply", s.handleStartApply)
	mux.HandleFunc("POST /api/services/{name}/fork", s.handleStartFork)
	mux.HandleFunc("GET /api/backups", s.handleListBackups)
	mux.HandleFunc("GET /api/backups/{id}", s.handleGetBackup)
	mux.HandleFunc("POST /api/backups/{id}/cancel", s.handleCancelBackup)
	mux.HandleFunc("DELETE /api/backups/{id}", s.handleDeleteBackup)
	mux.Handle("/", s.spaHandler())
	return mux
}

// ListenAndServe serves the handler on addr until the process is stopped.
func (s *Server) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, s.Handler())
}

// statusResponse is the JSON shape returned by GET /api/status. Initialized is
// false for a folder with no easy-infra project; the remaining fields are then
// empty.
type statusResponse struct {
	Initialized   bool      `json:"initialized"`
	ActiveProfile string    `json:"activeProfile"`
	Profiles      []profile `json:"profiles"`
	Services      []string  `json:"services"`
}

// profile pairs a profile name with whether it is the active one.
type profile struct {
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	proj, err := s.activeProject()
	if err != nil {
		if errors.Is(err, project.ErrNotInitialized) {
			writeJSON(w, http.StatusOK, statusResponse{Profiles: []profile{}, Services: []string{}})
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	names, err := proj.Profiles()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	active := proj.ActiveProfileName()
	profiles := make([]profile, 0, len(names))
	for _, name := range names {
		profiles = append(profiles, profile{Name: name, Active: name == active})
	}

	// Services now belong to a profile, so report the active profile's set. With
	// no active profile (or one that fails to load) the list is simply empty.
	services := []string{}
	if active != "" {
		if defs, err := proj.ProfileServices(active); err == nil {
			for _, d := range defs {
				services = append(services, d.Name)
			}
		}
	}

	writeJSON(w, http.StatusOK, statusResponse{
		Initialized:   true,
		ActiveProfile: active,
		Profiles:      profiles,
		Services:      services,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// spaHandler serves the embedded single-page app, falling back to index.html so
// client-side routes resolve. When the bundle has not been built it serves a
// short page pointing at `make ui` rather than a bare 404.
func (s *Server) spaHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.hasIndex() {
			writeUnbuilt(w)
			return
		}
		if _, err := fs.Stat(s.ui, pathForRequest(r.URL.Path)); err != nil {
			// Unknown path: hand it to the SPA so its router can handle it.
			r = r.Clone(r.Context())
			r.URL.Path = "/"
		}
		http.FileServer(http.FS(s.ui)).ServeHTTP(w, r)
	})
}

// pathForRequest maps a request path to a lookup key within the UI filesystem.
func pathForRequest(p string) string {
	if p == "/" || p == "" {
		return "index.html"
	}
	return p[1:]
}

func (s *Server) hasIndex() bool {
	_, err := fs.Stat(s.ui, "index.html")
	return err == nil
}

func writeUnbuilt(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, unbuiltPage)
}

const unbuiltPage = `<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>easy-infra</title></head>
<body style="font-family: system-ui, sans-serif; max-width: 40rem; margin: 4rem auto; padding: 0 1rem;">
  <h1>easy-infra UI not built</h1>
  <p>The frontend bundle has not been built yet. From the repository root run:</p>
  <pre style="background:#f4f4f5;padding:1rem;border-radius:6px;">make ui</pre>
  <p>then restart <code>easy-infra serve</code>. The JSON API at
  <code>/api/status</code> is available now.</p>
</body>
</html>
`
