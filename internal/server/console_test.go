package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/minhnc/easy-infra/internal/config"
	profilepkg "github.com/minhnc/easy-infra/internal/profile"
	"github.com/minhnc/easy-infra/internal/project"
	"github.com/minhnc/easy-infra/internal/service"
)

// stubQuerier is a console-capable stubService serving canned query and schema
// outcomes, recording the SQL it was asked to run.
type stubQuerier struct {
	stubService
	result    *service.QueryResult
	queryErr  error
	schema    *service.SchemaInfo
	schemaErr error
	gotSQL    string
	gotSpec   service.Spec
}

func (s *stubQuerier) Query(_ context.Context, spec service.Spec, sql string) (*service.QueryResult, error) {
	s.gotSQL = sql
	s.gotSpec = spec
	return s.result, s.queryErr
}

func (s *stubQuerier) Schema(_ context.Context, spec service.Spec) (*service.SchemaInfo, error) {
	s.gotSpec = spec
	return s.schema, s.schemaErr
}

// newConsoleServer scaffolds a project whose single service is the given stub,
// with a "default" profile, and returns a server over it.
func newConsoleServer(t *testing.T, svc service.Service) *Server {
	t.Helper()
	reg := service.NewRegistry()
	if err := reg.Register(svc); err != nil {
		t.Fatalf("Register: %v", err)
	}
	paths := newPaths(t)
	if err := config.Scaffold().Save(paths.Config); err != nil {
		t.Fatalf("Save config: %v", err)
	}
	prof, err := profilepkg.Scaffold(reg, svc.Name())
	if err != nil {
		t.Fatalf("Scaffold profile: %v", err)
	}
	if err := prof.Save(paths.ProfilePath("default")); err != nil {
		t.Fatalf("Save profile: %v", err)
	}
	if _, err := project.Load(paths, reg); err != nil {
		t.Fatalf("Load: %v", err)
	}
	return New(reg, paths, emptyUI)
}

func decodeQuery(t *testing.T, body []byte) queryResponse {
	t.Helper()
	var got queryResponse
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode query response: %v (body %q)", err, body)
	}
	return got
}

func TestConsoleQueryHappyPath(t *testing.T) {
	stub := &stubQuerier{
		stubService: stubService{name: "stub"},
		result: &service.QueryResult{
			Columns:  []string{"id"},
			Rows:     [][]any{{float64(1)}},
			RowCount: 1,
			Command:  "SELECT 1",
		},
	}
	srv := newConsoleServer(t, stub)
	rec := doJSON(t, srv, http.MethodPost, "/api/profiles/default/services/stub/query",
		queryRequest{SQL: "SELECT id FROM t"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	got := decodeQuery(t, rec.Body.Bytes())
	if got.Error != "" || got.RowCount != 1 || got.Command != "SELECT 1" || len(got.Rows) != 1 {
		t.Fatalf("response = %+v", got)
	}
	if stub.gotSQL != "SELECT id FROM t" {
		t.Errorf("executed sql = %q", stub.gotSQL)
	}
	if stub.gotSpec.Profile != "default" || stub.gotSpec.Env == nil {
		t.Errorf("spec = %+v, want profile default with the saved env", stub.gotSpec)
	}
}

func TestConsoleQueryExecutionError(t *testing.T) {
	stub := &stubQuerier{
		stubService: stubService{name: "stub"},
		queryErr:    errors.New(`relation "usrs" does not exist`),
	}
	srv := newConsoleServer(t, stub)
	rec := doJSON(t, srv, http.MethodPost, "/api/profiles/default/services/stub/query",
		queryRequest{SQL: "SELECT * FROM usrs"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	got := decodeQuery(t, rec.Body.Bytes())
	if got.Error == "" || len(got.Rows) != 0 {
		t.Fatalf("response = %+v, want error envelope with no rows", got)
	}
}

func TestConsoleQueryEmptySQL(t *testing.T) {
	srv := newConsoleServer(t, &stubQuerier{stubService: stubService{name: "stub"}})
	rec := doJSON(t, srv, http.MethodPost, "/api/profiles/default/services/stub/query",
		queryRequest{SQL: "   \n"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body %q)", rec.Code, rec.Body.String())
	}
}

func TestConsoleQueryUnknownProfileAndService(t *testing.T) {
	srv := newConsoleServer(t, &stubQuerier{stubService: stubService{name: "stub"}})
	rec := doJSON(t, srv, http.MethodPost, "/api/profiles/nope/services/stub/query",
		queryRequest{SQL: "SELECT 1"})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown profile: status = %d, want 404 (body %q)", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, srv, http.MethodPost, "/api/profiles/default/services/nope/query",
		queryRequest{SQL: "SELECT 1"})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown service: status = %d, want 404 (body %q)", rec.Code, rec.Body.String())
	}
}

func TestConsoleQueryUnsupportedService(t *testing.T) {
	// A plain stubService does not implement service.Querier.
	srv := newConsoleServer(t, stubService{name: "stub"})
	rec := doJSON(t, srv, http.MethodPost, "/api/profiles/default/services/stub/query",
		queryRequest{SQL: "SELECT 1"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body %q)", rec.Code, rec.Body.String())
	}
}

func TestConsoleSchema(t *testing.T) {
	stub := &stubQuerier{
		stubService: stubService{name: "stub"},
		schema: &service.SchemaInfo{Tables: []service.TableInfo{
			{Schema: "public", Name: "users", Columns: []string{"id", "email"}},
		}},
	}
	srv := newConsoleServer(t, stub)
	rec := doJSON(t, srv, http.MethodGet, "/api/profiles/default/services/stub/schema", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	var got schemaResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode schema response: %v", err)
	}
	if got.Error != "" || len(got.Tables) != 1 || got.Tables[0].Name != "users" {
		t.Fatalf("response = %+v", got)
	}
}

func TestConsoleSchemaIntrospectionError(t *testing.T) {
	stub := &stubQuerier{
		stubService: stubService{name: "stub"},
		schemaErr:   errors.New("connection refused"),
	}
	srv := newConsoleServer(t, stub)
	rec := doJSON(t, srv, http.MethodGet, "/api/profiles/default/services/stub/schema", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	var got schemaResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode schema response: %v", err)
	}
	if got.Error == "" || len(got.Tables) != 0 {
		t.Fatalf("response = %+v, want error envelope with no tables", got)
	}
}
