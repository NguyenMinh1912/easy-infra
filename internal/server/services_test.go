package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	profilepkg "github.com/minhnc/easy-infra/internal/profile"
	"github.com/minhnc/easy-infra/internal/project"
	"github.com/minhnc/easy-infra/internal/service"
)

// doJSON sends a request with an optional JSON body and returns the recorder.
func doJSON(t *testing.T, srv *Server, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

func TestServiceCatalog(t *testing.T) {
	// The catalog comes from the registry, so it is available without a project.
	srv := New(service.DefaultRegistry(), newPaths(t), emptyUI)
	rec := doJSON(t, srv, http.MethodGet, "/api/services/catalog", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", rec.Code)
	}
	var got catalogResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	names := map[string]bool{}
	for _, e := range got.Services {
		names[e.Name] = true
	}
	for _, want := range []string{"postgres", "redis", "minio", "localstack"} {
		if !names[want] {
			t.Errorf("catalog missing %q (got %v)", want, names)
		}
	}
	// Default definitions ride along so the UI can preview/seed them.
	for _, e := range got.Services {
		if e.Name == "postgres" && e.DefaultDefinition["version"] == nil {
			t.Errorf("postgres catalog entry missing default version: %+v", e)
		}
	}
}

func TestListServicesNotInitialized(t *testing.T) {
	srv := New(service.DefaultRegistry(), newPaths(t), emptyUI)
	rec := doJSON(t, srv, http.MethodGet, "/api/services", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", rec.Code)
	}
	var got servicesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Initialized || len(got.Services) != 0 {
		t.Errorf("got %+v, want uninitialized and empty", got)
	}
}

func TestListServices(t *testing.T) {
	paths, reg := initProject(t, "postgres", "redis")
	srv := New(reg, paths, emptyUI)
	rec := doJSON(t, srv, http.MethodGet, "/api/services", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", rec.Code)
	}
	var got servicesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !got.Initialized || len(got.Services) != 2 {
		t.Fatalf("got %+v, want 2 services", got)
	}
	if got.Services[0].Name != "postgres" || got.Services[1].Name != "redis" {
		t.Errorf("services not sorted: %+v", got.Services)
	}
	if got.Services[0].Definition["version"] != "16" {
		t.Errorf("postgres definition = %+v, want version 16", got.Services[0].Definition)
	}
}

func TestCreateService(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, paths, emptyUI)

	rec := doJSON(t, srv, http.MethodPost, "/api/services", serviceRequest{Name: "redis"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("code = %d, want 201 (body %s)", rec.Code, rec.Body)
	}

	// Config now defines redis...
	proj, err := project.Load(paths, reg)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !proj.Config.HasService("redis") {
		t.Error("config does not define redis after create")
	}
	// ...and the active profile was scaffolded with redis env, so it stays valid.
	prof, err := profilepkg.Load(paths.ProfilePath("default"))
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	if _, ok := prof.Services["redis"]; !ok {
		t.Error("profile not scaffolded with redis env")
	}
	if _, err := proj.LoadProfile("default"); err != nil {
		t.Errorf("profile invalid after create: %v", err)
	}
}

func TestCreateServiceErrors(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, paths, emptyUI)

	tests := []struct {
		name     string
		body     serviceRequest
		wantCode int
	}{
		{"already defined", serviceRequest{Name: "postgres"}, http.StatusConflict},
		{"unknown service", serviceRequest{Name: "nope"}, http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := doJSON(t, srv, http.MethodPost, "/api/services", tc.body)
			if rec.Code != tc.wantCode {
				t.Errorf("code = %d, want %d (body %s)", rec.Code, tc.wantCode, rec.Body)
			}
		})
	}
}

func TestCreateServiceNotInitialized(t *testing.T) {
	srv := New(service.DefaultRegistry(), newPaths(t), emptyUI)
	rec := doJSON(t, srv, http.MethodPost, "/api/services", serviceRequest{Name: "redis"})
	if rec.Code != http.StatusConflict {
		t.Errorf("code = %d, want 409", rec.Code)
	}
}

func TestUpdateService(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, paths, emptyUI)

	rec := doJSON(t, srv, http.MethodPut, "/api/services/postgres",
		serviceRequest{Definition: service.Config{"version": "17"}})
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200 (body %s)", rec.Code, rec.Body)
	}

	proj, err := project.Load(paths, reg)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if proj.Config.Services["postgres"]["version"] != "17" {
		t.Errorf("version = %v, want 17", proj.Config.Services["postgres"]["version"])
	}
}

func TestUpdateServiceNotDefined(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, paths, emptyUI)
	rec := doJSON(t, srv, http.MethodPut, "/api/services/redis",
		serviceRequest{Definition: service.Config{"version": "7"}})
	if rec.Code != http.StatusNotFound {
		t.Errorf("code = %d, want 404", rec.Code)
	}
}

func TestDeleteService(t *testing.T) {
	paths, reg := initProject(t, "postgres", "redis")
	srv := New(reg, paths, emptyUI)

	rec := doJSON(t, srv, http.MethodDelete, "/api/services/redis", nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("code = %d, want 204 (body %s)", rec.Code, rec.Body)
	}

	proj, err := project.Load(paths, reg)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if proj.Config.HasService("redis") {
		t.Error("config still defines redis after delete")
	}
	// The profile's redis env was removed too, so it stays valid.
	prof, err := profilepkg.Load(paths.ProfilePath("default"))
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	if _, ok := prof.Services["redis"]; ok {
		t.Error("profile still has redis env after delete")
	}
	if _, err := proj.LoadProfile("default"); err != nil {
		t.Errorf("profile invalid after delete: %v", err)
	}
}

func TestDeleteLastService(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, paths, emptyUI)
	rec := doJSON(t, srv, http.MethodDelete, "/api/services/postgres", nil)
	if rec.Code != http.StatusConflict {
		t.Errorf("code = %d, want 409 (cannot remove last service)", rec.Code)
	}
}
