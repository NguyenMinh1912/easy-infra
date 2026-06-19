package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	profilepkg "github.com/minhnc/easy-infra/internal/profile"
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

// stubRowEditor is a console stub that also edits rows, recording the mutation
// it was handed and serving a canned command tag.
type stubRowEditor struct {
	stubQuerier
	cmd     string
	editErr error
	gotMut  service.RowMutation
	deleted bool
}

func (s *stubRowEditor) UpdateRow(_ context.Context, spec service.Spec, m service.RowMutation) (string, error) {
	s.gotSpec = spec
	s.gotMut = m
	return s.cmd, s.editErr
}

func (s *stubRowEditor) DeleteRow(_ context.Context, spec service.Spec, m service.RowMutation) (string, error) {
	s.gotSpec = spec
	s.gotMut = m
	s.deleted = true
	return s.cmd, s.editErr
}

// stubRelationBrowser is a console stub that also follows relations, recording
// the query it was handed and serving a canned result.
type stubRelationBrowser struct {
	stubQuerier
	related    *service.QueryResult
	relErr     error
	gotRelated service.RelatedQuery
}

func (s *stubRelationBrowser) RelatedRows(_ context.Context, spec service.Spec, q service.RelatedQuery) (*service.QueryResult, error) {
	s.gotSpec = spec
	s.gotRelated = q
	return s.related, s.relErr
}

// newConsoleServer scaffolds a project whose single service is the given stub,
// with a "default" profile, and returns a server over it.
func newConsoleServer(t *testing.T, svc service.Service) *Server {
	t.Helper()
	reg := service.NewRegistry()
	if err := reg.Register(svc); err != nil {
		t.Fatalf("Register: %v", err)
	}
	st := newStore(t)
	ws, err := st.CreateWorkspace("test")
	if err != nil {
		t.Fatalf("CreateWorkspace: %v", err)
	}
	prof, err := profilepkg.Scaffold(reg, svc.Name())
	if err != nil {
		t.Fatalf("Scaffold profile: %v", err)
	}
	if err := st.CreateProfile(ws.ID, "default", prof.Services); err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}
	if err := st.SetWorkspaceActiveProfile(ws.ID, "default"); err != nil {
		t.Fatalf("SetWorkspaceActiveProfile: %v", err)
	}
	if err := st.SetActiveWorkspace(ws.ID); err != nil {
		t.Fatalf("SetActiveWorkspace: %v", err)
	}
	return New(reg, st, emptyUI)
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

func TestConsoleQueryDBOverride(t *testing.T) {
	stub := &stubQuerier{
		stubService: stubService{name: "stub"},
		result:      &service.QueryResult{Command: "GET"},
	}
	srv := newConsoleServer(t, stub)
	db := 3
	rec := doJSON(t, srv, http.MethodPost, "/api/profiles/default/services/stub/query",
		queryRequest{SQL: "GET foo", DB: &db})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	if got := stub.gotSpec.Env["db"]; got != 3 {
		t.Errorf("spec env db = %v, want 3", got)
	}

	negative := -1
	rec = doJSON(t, srv, http.MethodPost, "/api/profiles/default/services/stub/query",
		queryRequest{SQL: "GET foo", DB: &negative})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("negative db: status = %d, want 400 (body %q)", rec.Code, rec.Body.String())
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

func TestRowUpdateHappyPath(t *testing.T) {
	stub := &stubRowEditor{stubQuerier: stubQuerier{stubService: stubService{name: "stub"}}, cmd: "UPDATE 1"}
	srv := newConsoleServer(t, stub)
	email := "ada@example.com"
	rec := doJSON(t, srv, http.MethodPatch, "/api/profiles/default/services/stub/row",
		rowMutationRequest{Schema: "public", Table: "users", Key: map[string]string{"id": "7"}, Column: "email", Value: &email})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	var got commandResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Command != "UPDATE 1" {
		t.Errorf("command = %q, want %q", got.Command, "UPDATE 1")
	}
	if stub.gotMut.Table != "users" || stub.gotMut.Column != "email" || stub.gotMut.Key["id"] != "7" {
		t.Errorf("mutation = %+v", stub.gotMut)
	}
	if stub.gotMut.Value == nil || *stub.gotMut.Value != email {
		t.Errorf("value = %v, want %q", stub.gotMut.Value, email)
	}
}

func TestRowDeleteHappyPath(t *testing.T) {
	stub := &stubRowEditor{stubQuerier: stubQuerier{stubService: stubService{name: "stub"}}, cmd: "DELETE 1"}
	srv := newConsoleServer(t, stub)
	rec := doJSON(t, srv, http.MethodDelete, "/api/profiles/default/services/stub/row",
		rowMutationRequest{Schema: "public", Table: "users", Key: map[string]string{"id": "7"}})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	if !stub.deleted || stub.gotMut.Key["id"] != "7" {
		t.Errorf("delete not applied: deleted=%v mutation=%+v", stub.deleted, stub.gotMut)
	}
}

func TestRowMutationValidation(t *testing.T) {
	stub := &stubRowEditor{stubQuerier: stubQuerier{stubService: stubService{name: "stub"}}, cmd: "UPDATE 1"}
	srv := newConsoleServer(t, stub)
	cases := []struct {
		name string
		req  rowMutationRequest
	}{
		{"missing table", rowMutationRequest{Schema: "public", Key: map[string]string{"id": "7"}, Column: "email"}},
		{"missing key", rowMutationRequest{Schema: "public", Table: "users", Column: "email"}},
		{"missing column", rowMutationRequest{Schema: "public", Table: "users", Key: map[string]string{"id": "7"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doJSON(t, srv, http.MethodPatch, "/api/profiles/default/services/stub/row", tc.req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body %q)", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestRowEditUnsupportedService(t *testing.T) {
	// A plain stubQuerier supports a console but not row editing.
	srv := newConsoleServer(t, &stubQuerier{stubService: stubService{name: "stub"}})
	rec := doJSON(t, srv, http.MethodDelete, "/api/profiles/default/services/stub/row",
		rowMutationRequest{Schema: "public", Table: "users", Key: map[string]string{"id": "7"}})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body %q)", rec.Code, rec.Body.String())
	}
}

func TestRelatedRowsHappyPath(t *testing.T) {
	stub := &stubRelationBrowser{
		stubQuerier: stubQuerier{stubService: stubService{name: "stub"}},
		related: &service.QueryResult{
			Columns: []string{"id", "order_id"}, Rows: [][]any{{int64(5), int64(9)}},
			RowCount: 1, Command: "SELECT 1",
		},
	}
	srv := newConsoleServer(t, stub)
	order := "9"
	rec := doJSON(t, srv, http.MethodPost, "/api/profiles/default/services/stub/related",
		relatedRequest{Schema: "public", Table: "order_items", Filters: []relatedFilterIn{{Column: "order_id", Value: &order}}})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	got := decodeQuery(t, rec.Body.Bytes())
	if got.Command != "SELECT 1" || got.RowCount != 1 {
		t.Errorf("response = %+v", got)
	}
	if stub.gotRelated.Table != "order_items" || len(stub.gotRelated.Filters) != 1 {
		t.Fatalf("query = %+v", stub.gotRelated)
	}
	f := stub.gotRelated.Filters[0]
	if f.Column != "order_id" || f.Value == nil || *f.Value != "9" {
		t.Errorf("filter = %+v", f)
	}
}

func TestRelatedRowsExecutionError(t *testing.T) {
	stub := &stubRelationBrowser{
		stubQuerier: stubQuerier{stubService: stubService{name: "stub"}},
		relErr:      errors.New("relation does not exist"),
	}
	srv := newConsoleServer(t, stub)
	// A failing lookup is an expected outcome: 200 with the reason in error.
	rec := doJSON(t, srv, http.MethodPost, "/api/profiles/default/services/stub/related",
		relatedRequest{Schema: "public", Table: "order_items", Filters: []relatedFilterIn{{Column: "order_id", Value: nil}}})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	if got := decodeQuery(t, rec.Body.Bytes()); got.Error == "" {
		t.Errorf("expected error in response, got %+v", got)
	}
}

func TestRelatedRowsValidation(t *testing.T) {
	stub := &stubRelationBrowser{stubQuerier: stubQuerier{stubService: stubService{name: "stub"}}}
	srv := newConsoleServer(t, stub)
	rec := doJSON(t, srv, http.MethodPost, "/api/profiles/default/services/stub/related",
		relatedRequest{Schema: "public"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body %q)", rec.Code, rec.Body.String())
	}
}

func TestRelatedRowsUnsupportedService(t *testing.T) {
	// A plain stubQuerier supports a console but not following relations.
	srv := newConsoleServer(t, &stubQuerier{stubService: stubService{name: "stub"}})
	rec := doJSON(t, srv, http.MethodPost, "/api/profiles/default/services/stub/related",
		relatedRequest{Schema: "public", Table: "order_items"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body %q)", rec.Code, rec.Body.String())
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
