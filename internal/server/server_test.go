package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/minhnc/easy-infra/internal/config"
	"github.com/minhnc/easy-infra/internal/project"
	"github.com/minhnc/easy-infra/internal/service"
)

// emptyUI is a UI filesystem with no built bundle.
var emptyUI = fstest.MapFS{}

// newPaths returns project paths rooted at a fresh temp dir.
func newPaths(t *testing.T) project.Paths {
	t.Helper()
	dir := t.TempDir()
	return project.Paths{
		Config:      filepath.Join(dir, "easy-infra.yml"),
		State:       filepath.Join(dir, "state.json"),
		ProfilesDir: filepath.Join(dir, "profiles"),
	}
}

// initProject scaffolds a config on disk plus an active profile, returning the
// paths the server should read from.
func initProject(t *testing.T, services ...string) (project.Paths, *service.Registry) {
	t.Helper()
	reg := service.DefaultRegistry()
	paths := newPaths(t)

	cfg, err := config.Scaffold(reg, services...)
	if err != nil {
		t.Fatalf("Scaffold: %v", err)
	}
	if err := cfg.Save(paths.Config); err != nil {
		t.Fatalf("Save config: %v", err)
	}

	proj, err := project.Load(paths, reg)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := proj.AddProfile("default"); err != nil {
		t.Fatalf("AddProfile: %v", err)
	}
	if err := proj.SetActiveProfile("default"); err != nil {
		t.Fatalf("SetActiveProfile: %v", err)
	}
	return paths, reg
}

func getStatus(t *testing.T, srv *Server) (*httptest.ResponseRecorder, statusResponse) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	var got statusResponse
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode status: %v (body %q)", err, rec.Body.String())
		}
	}
	return rec, got
}

func TestStatusNotInitialized(t *testing.T) {
	srv := New(service.DefaultRegistry(), newPaths(t), emptyUI)
	rec, got := getStatus(t, srv)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200", rec.Code)
	}
	if got.Initialized {
		t.Error("Initialized = true, want false for an uninitialized folder")
	}
	if len(got.Profiles) != 0 || len(got.Services) != 0 {
		t.Errorf("expected empty profiles/services, got %+v", got)
	}
}

func TestStatusInitialized(t *testing.T) {
	paths, reg := initProject(t, "postgres", "redis")
	srv := New(reg, paths, emptyUI)
	rec, got := getStatus(t, srv)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200", rec.Code)
	}
	if !got.Initialized {
		t.Fatal("Initialized = false, want true")
	}
	if got.ActiveProfile != "default" {
		t.Errorf("ActiveProfile = %q, want \"default\"", got.ActiveProfile)
	}
	if len(got.Profiles) != 1 || got.Profiles[0].Name != "default" || !got.Profiles[0].Active {
		t.Errorf("Profiles = %+v, want one active \"default\"", got.Profiles)
	}
	wantServices := map[string]bool{"postgres": true, "redis": true}
	if len(got.Services) != len(wantServices) {
		t.Errorf("Services = %v, want %v", got.Services, wantServices)
	}
	for _, s := range got.Services {
		if !wantServices[s] {
			t.Errorf("unexpected service %q in %v", s, got.Services)
		}
	}
}

func TestStatusMethodNotAllowed(t *testing.T) {
	srv := New(service.DefaultRegistry(), newPaths(t), emptyUI)
	req := httptest.NewRequest(http.MethodPost, "/api/status", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code = %d, want 405", rec.Code)
	}
}

func TestSPAUnbuilt(t *testing.T) {
	srv := New(service.DefaultRegistry(), newPaths(t), emptyUI)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200", rec.Code)
	}
	if body, _ := io.ReadAll(rec.Body); !strings.Contains(string(body), "make ui") {
		t.Errorf("unbuilt page missing build hint, got %q", body)
	}
}

func TestSPAServesIndexAndFallback(t *testing.T) {
	ui := fstest.MapFS{
		"index.html": {Data: []byte("<!doctype html><title>app</title>")},
	}
	srv := New(service.DefaultRegistry(), newPaths(t), ui)

	// Root serves index.html, and an unknown client-route falls back to it too.
	for _, path := range []string{"/", "/profiles"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s code = %d, want 200", path, rec.Code)
		}
		if body, _ := io.ReadAll(rec.Body); !strings.Contains(string(body), "<title>app</title>") {
			t.Errorf("GET %s did not serve index.html, got %q", path, body)
		}
	}
}
