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
	profilepkg "github.com/minhnc/easy-infra/internal/profile"
	"github.com/minhnc/easy-infra/internal/project"
	"github.com/minhnc/easy-infra/internal/service"
	"github.com/minhnc/easy-infra/internal/workspace"
)

// emptyUI is a UI filesystem with no built bundle.
var emptyUI = fstest.MapFS{}

// newPaths returns project paths rooted at a fresh temp dir, in the conventional
// workspace layout the server uses.
func newPaths(t *testing.T) project.Paths {
	t.Helper()
	return project.PathsFor(t.TempDir())
}

// regFrom builds a single-workspace registry whose active workspace is the root
// of paths, so a Server constructed with it operates on that folder.
func regFrom(t *testing.T, paths project.Paths) *workspace.Registry {
	t.Helper()
	root := filepath.Dir(paths.Config)
	r := &workspace.Registry{}
	if err := r.Add("test", root); err != nil {
		t.Fatalf("Add workspace: %v", err)
	}
	if err := r.SetActive("test"); err != nil {
		t.Fatalf("SetActive: %v", err)
	}
	return r
}

// initProject scaffolds a config marker on disk plus an active "default"
// profile owning the given services, returning the paths the server reads from.
func initProject(t *testing.T, services ...string) (project.Paths, *service.Registry) {
	t.Helper()
	reg := service.DefaultRegistry()
	paths := newPaths(t)

	if err := config.Scaffold().Save(paths.Config); err != nil {
		t.Fatalf("Save config: %v", err)
	}

	prof, err := profilepkg.Scaffold(reg, services...)
	if err != nil {
		t.Fatalf("Scaffold profile: %v", err)
	}
	if err := prof.Save(paths.ProfilePath("default")); err != nil {
		t.Fatalf("Save profile: %v", err)
	}

	proj, err := project.Load(paths, reg)
	if err != nil {
		t.Fatalf("Load: %v", err)
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
	srv := New(service.DefaultRegistry(), regFrom(t, newPaths(t)), emptyUI)
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
	srv := New(reg, regFrom(t, paths), emptyUI)
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
	srv := New(service.DefaultRegistry(), regFrom(t, newPaths(t)), emptyUI)
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
	srv := New(reg, regFrom(t, paths), emptyUI)

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
	srv := New(service.DefaultRegistry(), regFrom(t, newPaths(t)), emptyUI)
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
	srv := New(reg, regFrom(t, paths), emptyUI)

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
	srv := New(reg, regFrom(t, paths), emptyUI)

	rec := doRequest(t, srv, http.MethodPost, "/api/profiles", `{"name":"default"}`)
	if rec.Code != http.StatusConflict {
		t.Errorf("code = %d, want 409", rec.Code)
	}
}

func TestCreateProfileInvalidName(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, regFrom(t, paths), emptyUI)

	for _, name := range []string{"", "../escape", "has space", "with/slash"} {
		rec := doRequest(t, srv, http.MethodPost, "/api/profiles", `{"name":"`+name+`"}`)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("name %q: code = %d, want 400", name, rec.Code)
		}
	}
}

func TestCreateProfileNotInitialized(t *testing.T) {
	srv := New(service.DefaultRegistry(), regFrom(t, newPaths(t)), emptyUI)
	rec := doRequest(t, srv, http.MethodPost, "/api/profiles", `{"name":"staging"}`)
	if rec.Code != http.StatusConflict {
		t.Errorf("code = %d, want 409", rec.Code)
	}
}

func TestActivateProfile(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, regFrom(t, paths), emptyUI)

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
	srv := New(reg, regFrom(t, paths), emptyUI)

	rec := doRequest(t, srv, http.MethodPost, "/api/profiles/ghost/activate", "")
	if rec.Code != http.StatusConflict {
		t.Errorf("code = %d, want 409", rec.Code)
	}
}

func TestDeleteProfile(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, regFrom(t, paths), emptyUI)

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
	srv := New(reg, regFrom(t, paths), emptyUI)

	rec := doRequest(t, srv, http.MethodDelete, "/api/profiles/default", "")
	if rec.Code != http.StatusConflict {
		t.Errorf("code = %d, want 409 (refuse active delete)", rec.Code)
	}
}

func TestDeleteProfileMissing(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, regFrom(t, paths), emptyUI)

	rec := doRequest(t, srv, http.MethodDelete, "/api/profiles/ghost", "")
	if rec.Code != http.StatusConflict {
		t.Errorf("code = %d, want 409", rec.Code)
	}
}

func decodeProfileConfig(t *testing.T, rec *httptest.ResponseRecorder) profileConfigResponse {
	t.Helper()
	var got profileConfigResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode profile config: %v (body %q)", err, rec.Body.String())
	}
	return got
}

func TestGetProfileConfig(t *testing.T) {
	paths, reg := initProject(t, "postgres", "redis")
	srv := New(reg, regFrom(t, paths), emptyUI)

	rec := doRequest(t, srv, http.MethodGet, "/api/profiles/default", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	got := decodeProfileConfig(t, rec)
	if got.Name != "default" {
		t.Errorf("Name = %q, want \"default\"", got.Name)
	}
	// Scaffolded profile carries env config for every defined service, sorted.
	if len(got.Services) != 2 || got.Services[0].Name != "postgres" || got.Services[1].Name != "redis" {
		t.Errorf("Services = %+v, want sorted postgres,redis", got.Services)
	}
	if len(got.Services[0].Config) == 0 {
		t.Error("postgres config is empty, want default env keys")
	}
}

func TestGetProfileConfigMissing(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, regFrom(t, paths), emptyUI)

	rec := doRequest(t, srv, http.MethodGet, "/api/profiles/ghost", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("code = %d, want 404", rec.Code)
	}
}

func TestUpdateProfileConfig(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, regFrom(t, paths), emptyUI)

	// Load the scaffolded config, change one value, and write it back.
	current := decodeProfileConfig(t, doRequest(t, srv, http.MethodGet, "/api/profiles/default", ""))
	current.Services[0].Config["host"] = "db.internal"
	body, err := json.Marshal(profileConfigRequest{Services: current.Services})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	rec := doRequest(t, srv, http.MethodPut, "/api/profiles/default", string(body))
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}

	// The change is persisted and visible on the next read.
	got := decodeProfileConfig(t, doRequest(t, srv, http.MethodGet, "/api/profiles/default", ""))
	if got.Services[0].Config["host"] != "db.internal" {
		t.Errorf("host = %v, want \"db.internal\"", got.Services[0].Config["host"])
	}
}

func TestUpdateProfileConfigMissingService(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, regFrom(t, paths), emptyUI)

	// Omitting the defined service leaves the profile invalid; the update is
	// refused rather than silently dropping config.
	rec := doRequest(t, srv, http.MethodPut, "/api/profiles/default", `{"services":[]}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("code = %d, want 400 (body %q)", rec.Code, rec.Body.String())
	}
}

func TestUpdateProfileConfigMissingProfile(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, regFrom(t, paths), emptyUI)

	rec := doRequest(t, srv, http.MethodPut, "/api/profiles/ghost", `{"services":[]}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("code = %d, want 400", rec.Code)
	}
}

func TestSPAUnbuilt(t *testing.T) {
	srv := New(service.DefaultRegistry(), regFrom(t, newPaths(t)), emptyUI)
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
	srv := New(service.DefaultRegistry(), regFrom(t, newPaths(t)), ui)

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
