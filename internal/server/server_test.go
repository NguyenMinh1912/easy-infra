package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	profilepkg "github.com/minhnc/easy-infra/internal/profile"
	"github.com/minhnc/easy-infra/internal/project"
	"github.com/minhnc/easy-infra/internal/service"
	"github.com/minhnc/easy-infra/internal/store"
)

// emptyUI is a UI filesystem with no built bundle.
var emptyUI = fstest.MapFS{}

// newStore opens a store in a fresh temp config dir with no workspace, so a
// Server built from it behaves as "not initialized".
func newStore(t *testing.T) *store.Store {
	t.Helper()
	t.Setenv("EASY_INFRA_CONFIG_DIR", t.TempDir())
	st, err := store.Open()
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

// initProject opens a store with one active workspace whose "default" profile
// owns the given services and is active, returning the store and registry the
// server reads from.
func initProject(t *testing.T, services ...string) (*store.Store, *service.Registry) {
	t.Helper()
	reg := service.DefaultRegistry()
	st := newStore(t)

	ws, err := project.CreateWorkspace(st, reg, "test")
	if err != nil {
		t.Fatalf("CreateWorkspace: %v", err)
	}
	if err := st.SetActiveWorkspace(ws.ID); err != nil {
		t.Fatalf("SetActiveWorkspace: %v", err)
	}
	// A new workspace has no profiles; create the "default" profile owning
	// exactly the requested services and make it active.
	prof, err := profilepkg.Scaffold(reg, services...)
	if err != nil {
		t.Fatalf("Scaffold profile: %v", err)
	}
	if err := st.CreateProfile(ws.ID, "default", prof.Services); err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}
	if err := st.SetWorkspaceActiveProfile(ws.ID, "default"); err != nil {
		t.Fatalf("SetWorkspaceActiveProfile: %v", err)
	}
	return st, reg
}

// activeWSID returns the id of the store's active workspace.
func activeWSID(t *testing.T, st *store.Store) int64 {
	t.Helper()
	ws, ok, err := st.ActiveWorkspace()
	if err != nil || !ok {
		t.Fatalf("ActiveWorkspace = (%+v, %v, %v)", ws, ok, err)
	}
	return ws.ID
}

// profileServices returns the active workspace's named profile services, for
// tests asserting what was persisted.
func profileServices(t *testing.T, st *store.Store, name string) map[string]profilepkg.ServiceEntry {
	t.Helper()
	svcs, err := st.ProfileServices(activeWSID(t, st), name)
	if err != nil {
		t.Fatalf("ProfileServices(%q): %v", name, err)
	}
	return svcs
}

// loadProfile reads the active workspace's named profile into a Profile value,
// so tests can assert against prof.Services as before.
func loadProfile(t *testing.T, st *store.Store, name string) (*profilepkg.Profile, error) {
	svcs, err := st.ProfileServices(activeWSID(t, st), name)
	if err != nil {
		return nil, err
	}
	return &profilepkg.Profile{Services: svcs}, nil
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
	srv := New(service.DefaultRegistry(), newStore(t), emptyUI)
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
	st, reg := initProject(t, "postgres", "redis")
	srv := New(reg, st, emptyUI)
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
	srv := New(service.DefaultRegistry(), newStore(t), emptyUI)
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
	st, reg := initProject(t, "postgres")
	srv := New(reg, st, emptyUI)

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
	srv := New(service.DefaultRegistry(), newStore(t), emptyUI)
	rec := doRequest(t, srv, http.MethodGet, "/api/profiles", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", rec.Code)
	}
	if got := decodeProfiles(t, rec); len(got.Profiles) != 0 {
		t.Errorf("Profiles = %+v, want empty", got.Profiles)
	}
}

func TestCreateProfile(t *testing.T) {
	st, reg := initProject(t, "postgres")
	srv := New(reg, st, emptyUI)

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
	st, reg := initProject(t, "postgres")
	srv := New(reg, st, emptyUI)

	rec := doRequest(t, srv, http.MethodPost, "/api/profiles", `{"name":"default"}`)
	if rec.Code != http.StatusConflict {
		t.Errorf("code = %d, want 409", rec.Code)
	}
}

func TestCreateProfileInvalidName(t *testing.T) {
	st, reg := initProject(t, "postgres")
	srv := New(reg, st, emptyUI)

	for _, name := range []string{"", "../escape", "has space", "with/slash"} {
		rec := doRequest(t, srv, http.MethodPost, "/api/profiles", `{"name":"`+name+`"}`)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("name %q: code = %d, want 400", name, rec.Code)
		}
	}
}

func TestCreateProfileNotInitialized(t *testing.T) {
	srv := New(service.DefaultRegistry(), newStore(t), emptyUI)
	rec := doRequest(t, srv, http.MethodPost, "/api/profiles", `{"name":"staging"}`)
	if rec.Code != http.StatusConflict {
		t.Errorf("code = %d, want 409", rec.Code)
	}
}

func TestActivateProfile(t *testing.T) {
	st, reg := initProject(t, "postgres")
	srv := New(reg, st, emptyUI)

	if rec := doRequest(t, srv, http.MethodPost, "/api/profiles", `{"name":"staging"}`); rec.Code != http.StatusCreated {
		t.Fatalf("create staging: code = %d", rec.Code)
	}
	// A new profile is empty; it needs at least one service to be activatable.
	if rec := doJSON(t, srv, http.MethodPost, "/api/profiles/staging/services", serviceNameRequest{Type: "postgres"}); rec.Code != http.StatusCreated {
		t.Fatalf("add service to staging: code = %d (body %s)", rec.Code, rec.Body)
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
	st, reg := initProject(t, "postgres")
	srv := New(reg, st, emptyUI)

	rec := doRequest(t, srv, http.MethodPost, "/api/profiles/ghost/activate", "")
	if rec.Code != http.StatusConflict {
		t.Errorf("code = %d, want 409", rec.Code)
	}
}

func TestDeleteProfile(t *testing.T) {
	st, reg := initProject(t, "postgres")
	srv := New(reg, st, emptyUI)

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
	st, reg := initProject(t, "postgres")
	srv := New(reg, st, emptyUI)

	rec := doRequest(t, srv, http.MethodDelete, "/api/profiles/default", "")
	if rec.Code != http.StatusConflict {
		t.Errorf("code = %d, want 409 (refuse active delete)", rec.Code)
	}
}

func TestDeleteProfileMissing(t *testing.T) {
	st, reg := initProject(t, "postgres")
	srv := New(reg, st, emptyUI)

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
	st, reg := initProject(t, "postgres", "redis")
	srv := New(reg, st, emptyUI)

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
	st, reg := initProject(t, "postgres")
	srv := New(reg, st, emptyUI)

	rec := doRequest(t, srv, http.MethodGet, "/api/profiles/ghost", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("code = %d, want 404", rec.Code)
	}
}

func TestUpdateProfileConfig(t *testing.T) {
	st, reg := initProject(t, "postgres")
	srv := New(reg, st, emptyUI)

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
	st, reg := initProject(t, "postgres")
	srv := New(reg, st, emptyUI)

	// Omitting the defined service leaves the profile invalid; the update is
	// refused rather than silently dropping config.
	rec := doRequest(t, srv, http.MethodPut, "/api/profiles/default", `{"services":[]}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("code = %d, want 400 (body %q)", rec.Code, rec.Body.String())
	}
}

func TestUpdateProfileConfigMissingProfile(t *testing.T) {
	st, reg := initProject(t, "postgres")
	srv := New(reg, st, emptyUI)

	rec := doRequest(t, srv, http.MethodPut, "/api/profiles/ghost", `{"services":[]}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("code = %d, want 400", rec.Code)
	}
}

func TestSPAUnbuilt(t *testing.T) {
	srv := New(service.DefaultRegistry(), newStore(t), emptyUI)
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
	srv := New(service.DefaultRegistry(), newStore(t), ui)

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
