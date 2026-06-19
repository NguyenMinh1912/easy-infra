package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
	srv := New(service.DefaultRegistry(), newStore(t), emptyUI)
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
	// Default merged config rides along so the UI can preview/seed it: postgres
	// carries both its definition (version) and environment (host) defaults.
	for _, e := range got.Services {
		if e.Name == "postgres" {
			if e.DefaultConfig["version"] == nil {
				t.Errorf("postgres catalog entry missing default version: %+v", e)
			}
			if e.DefaultConfig["host"] == nil {
				t.Errorf("postgres catalog entry missing default host: %+v", e)
			}
		}
	}
}

func TestCreateProfileService(t *testing.T) {
	st, reg := initProject(t, "postgres")
	srv := New(reg, st, emptyUI)

	rec := doJSON(t, srv, http.MethodPost, "/api/profiles/default/services", serviceNameRequest{Type: "redis"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("code = %d, want 201 (body %s)", rec.Code, rec.Body)
	}

	prof, err := loadProfile(t, st, "default")
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	if _, ok := prof.Services["redis"]; !ok {
		t.Error("profile not scaffolded with redis config")
	}
	if _, ok := prof.Services["postgres"]; !ok {
		t.Error("existing postgres dropped after adding redis")
	}
}

func TestCreateProfileServiceSameTypeTwice(t *testing.T) {
	st, reg := initProject(t, "postgres")
	srv := New(reg, st, emptyUI)

	// A profile may now hold several instances of the same type; the second is
	// assigned a distinct id rather than being rejected.
	rec := doJSON(t, srv, http.MethodPost, "/api/profiles/default/services",
		serviceNameRequest{Type: "postgres", Name: "Analytics"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("code = %d, want 201 (body %s)", rec.Code, rec.Body)
	}

	prof, err := loadProfile(t, st, "default")
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	entry, ok := prof.Services["postgres-2"]
	if !ok {
		t.Fatalf("second postgres instance not created; services = %v", prof.Services)
	}
	if entry.ResolveType("postgres-2") != "postgres" {
		t.Errorf("postgres-2 type = %q, want postgres", entry.ResolveType("postgres-2"))
	}
	if entry.Name != "Analytics" {
		t.Errorf("postgres-2 name = %q, want Analytics", entry.Name)
	}
}

func TestCreateProfileServiceErrors(t *testing.T) {
	st, reg := initProject(t, "postgres")
	srv := New(reg, st, emptyUI)

	tests := []struct {
		name     string
		body     serviceNameRequest
		wantCode int
	}{
		{"unknown service", serviceNameRequest{Type: "nope"}, http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := doJSON(t, srv, http.MethodPost, "/api/profiles/default/services", tc.body)
			if rec.Code != tc.wantCode {
				t.Errorf("code = %d, want %d (body %s)", rec.Code, tc.wantCode, rec.Body)
			}
		})
	}
}

func TestCreateProfileServiceNotInitialized(t *testing.T) {
	srv := New(service.DefaultRegistry(), newStore(t), emptyUI)
	rec := doJSON(t, srv, http.MethodPost, "/api/profiles/default/services", serviceNameRequest{Type: "redis"})
	if rec.Code != http.StatusConflict {
		t.Errorf("code = %d, want 409", rec.Code)
	}
}

func TestUpdateProfileService(t *testing.T) {
	st, reg := initProject(t, "postgres")
	srv := New(reg, st, emptyUI)

	cfg := service.Config{
		"version": "17", "host": "db.example", "port": 5433, "user": "u", "database": "d",
	}
	rec := doJSON(t, srv, http.MethodPut, "/api/profiles/default/services/postgres",
		serviceConfigRequest{Config: cfg})
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200 (body %s)", rec.Code, rec.Body)
	}

	prof, err := loadProfile(t, st, "default")
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	if prof.Services["postgres"].Config["version"] != "17" {
		t.Errorf("version = %v, want 17", prof.Services["postgres"].Config["version"])
	}
	if prof.Services["postgres"].Config["host"] != "db.example" {
		t.Errorf("host = %v, want db.example", prof.Services["postgres"].Config["host"])
	}
}

func TestUpdateProfileServiceNotDefined(t *testing.T) {
	st, reg := initProject(t, "postgres")
	srv := New(reg, st, emptyUI)
	rec := doJSON(t, srv, http.MethodPut, "/api/profiles/default/services/redis",
		serviceConfigRequest{Config: service.Config{"host": "x"}})
	if rec.Code != http.StatusNotFound {
		t.Errorf("code = %d, want 404", rec.Code)
	}
}

func TestDeleteProfileService(t *testing.T) {
	st, reg := initProject(t, "postgres", "redis")
	srv := New(reg, st, emptyUI)

	rec := doJSON(t, srv, http.MethodDelete, "/api/profiles/default/services/redis", nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("code = %d, want 204 (body %s)", rec.Code, rec.Body)
	}

	prof, err := loadProfile(t, st, "default")
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	if _, ok := prof.Services["redis"]; ok {
		t.Error("profile still has redis after delete")
	}
}

func TestDeleteLastProfileService(t *testing.T) {
	st, reg := initProject(t, "postgres")
	srv := New(reg, st, emptyUI)
	rec := doJSON(t, srv, http.MethodDelete, "/api/profiles/default/services/postgres", nil)
	if rec.Code != http.StatusConflict {
		t.Errorf("code = %d, want 409 (cannot remove last service)", rec.Code)
	}
}
