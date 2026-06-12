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

// doRequest issues a request against the server's handler and returns the
// recorder.
func doRequest(t *testing.T, srv *Server, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

func decodeProfiles(t *testing.T, rec *httptest.ResponseRecorder) profilesResponse {
	t.Helper()
	var got profilesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode profiles: %v (body %q)", err, rec.Body.String())
	}
	return got
}

func TestListProfiles(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, paths, emptyUI)

	rec := doRequest(t, srv, http.MethodGet, "/api/profiles", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", rec.Code)
	}
	got := decodeProfiles(t, rec)
	if got.ActiveProfile != "default" {
		t.Errorf("ActiveProfile = %q, want \"default\"", got.ActiveProfile)
	}
	if len(got.Profiles) != 1 || got.Profiles[0].Name != "default" || !got.Profiles[0].Active {
		t.Errorf("Profiles = %+v, want one active \"default\"", got.Profiles)
	}
}

func TestListProfilesNotInitialized(t *testing.T) {
	srv := New(service.DefaultRegistry(), newPaths(t), emptyUI)
	rec := doRequest(t, srv, http.MethodGet, "/api/profiles", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", rec.Code)
	}
	if got := decodeProfiles(t, rec); len(got.Profiles) != 0 {
		t.Errorf("Profiles = %+v, want empty", got.Profiles)
	}
}

func TestCreateProfile(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, paths, emptyUI)

	rec := doRequest(t, srv, http.MethodPost, "/api/profiles", `{"name":"staging"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("code = %d, want 201 (body %q)", rec.Code, rec.Body.String())
	}
	got := decodeProfiles(t, rec)
	names := map[string]bool{}
	for _, p := range got.Profiles {
		names[p.Name] = true
	}
	if !names["default"] || !names["staging"] {
		t.Errorf("Profiles = %+v, want default and staging", got.Profiles)
	}
}

func TestCreateProfileDuplicate(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, paths, emptyUI)

	rec := doRequest(t, srv, http.MethodPost, "/api/profiles", `{"name":"default"}`)
	if rec.Code != http.StatusConflict {
		t.Errorf("code = %d, want 409", rec.Code)
	}
}

func TestCreateProfileInvalidName(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, paths, emptyUI)

	for _, name := range []string{"", "../escape", "has space", "with/slash"} {
		rec := doRequest(t, srv, http.MethodPost, "/api/profiles", `{"name":"`+name+`"}`)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("name %q: code = %d, want 400", name, rec.Code)
		}
	}
}

func TestCreateProfileNotInitialized(t *testing.T) {
	srv := New(service.DefaultRegistry(), newPaths(t), emptyUI)
	rec := doRequest(t, srv, http.MethodPost, "/api/profiles", `{"name":"staging"}`)
	if rec.Code != http.StatusConflict {
		t.Errorf("code = %d, want 409", rec.Code)
	}
}

func TestActivateProfile(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, paths, emptyUI)

	if rec := doRequest(t, srv, http.MethodPost, "/api/profiles", `{"name":"staging"}`); rec.Code != http.StatusCreated {
		t.Fatalf("create staging: code = %d", rec.Code)
	}

	rec := doRequest(t, srv, http.MethodPost, "/api/profiles/staging/activate", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	got := decodeProfiles(t, rec)
	if got.ActiveProfile != "staging" {
		t.Errorf("ActiveProfile = %q, want \"staging\"", got.ActiveProfile)
	}
}

func TestActivateProfileMissing(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, paths, emptyUI)

	rec := doRequest(t, srv, http.MethodPost, "/api/profiles/ghost/activate", "")
	if rec.Code != http.StatusConflict {
		t.Errorf("code = %d, want 409", rec.Code)
	}
}

func TestDeleteProfile(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, paths, emptyUI)

	if rec := doRequest(t, srv, http.MethodPost, "/api/profiles", `{"name":"staging"}`); rec.Code != http.StatusCreated {
		t.Fatalf("create staging: code = %d", rec.Code)
	}

	rec := doRequest(t, srv, http.MethodDelete, "/api/profiles/staging", "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("code = %d, want 204 (body %q)", rec.Code, rec.Body.String())
	}

	list := decodeProfiles(t, doRequest(t, srv, http.MethodGet, "/api/profiles", ""))
	for _, p := range list.Profiles {
		if p.Name == "staging" {
			t.Errorf("staging still present after delete: %+v", list.Profiles)
		}
	}
}

func TestDeleteActiveProfileRefused(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, paths, emptyUI)

	rec := doRequest(t, srv, http.MethodDelete, "/api/profiles/default", "")
	if rec.Code != http.StatusConflict {
		t.Errorf("code = %d, want 409 (refuse active delete)", rec.Code)
	}
}

func TestDeleteProfileMissing(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, paths, emptyUI)

	rec := doRequest(t, srv, http.MethodDelete, "/api/profiles/ghost", "")
	if rec.Code != http.StatusConflict {
		t.Errorf("code = %d, want 409", rec.Code)
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
