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
)

// Server answers API requests and serves the embedded UI. It loads the project
// fresh on each request so the API always reflects what is on disk.
type Server struct {
	reg   *service.Registry
	paths project.Paths
	ui    fs.FS
}

// New builds a Server from the injected service registry, project paths, and the
// embedded UI filesystem (see the ui package).
func New(reg *service.Registry, paths project.Paths, ui fs.FS) *Server {
	return &Server{reg: reg, paths: paths, ui: ui}
}

// Handler returns the HTTP handler tree: the JSON API under /api and the SPA on
// everything else.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", s.handleStatus)
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

	proj, err := project.Load(s.paths, s.reg)
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

	active := proj.State.ActiveProfile
	profiles := make([]profile, 0, len(names))
	for _, name := range names {
		profiles = append(profiles, profile{Name: name, Active: name == active})
	}

	writeJSON(w, http.StatusOK, statusResponse{
		Initialized:   true,
		ActiveProfile: active,
		Profiles:      profiles,
		Services:      proj.Config.ServiceNames(),
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
