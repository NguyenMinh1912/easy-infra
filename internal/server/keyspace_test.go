package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/minhnc/easy-infra/internal/service"
)

// stubKeyBrowser is a key-browse-capable stubService serving canned keyspace
// outcomes, recording the spec and arguments it was asked about.
type stubKeyBrowser struct {
	stubService
	count   int
	page    *service.KeyPage
	value   *service.KeyValue
	err     error
	gotSpec service.Spec
	gotDB   int
	gotKey  string
}

func (s *stubKeyBrowser) Databases(_ context.Context, spec service.Spec) (int, error) {
	s.gotSpec = spec
	return s.count, s.err
}

func (s *stubKeyBrowser) Keys(_ context.Context, spec service.Spec, db int, _ string, _ uint64) (*service.KeyPage, error) {
	s.gotSpec = spec
	s.gotDB = db
	return s.page, s.err
}

func (s *stubKeyBrowser) Value(_ context.Context, spec service.Spec, db int, key string) (*service.KeyValue, error) {
	s.gotSpec = spec
	s.gotDB = db
	s.gotKey = key
	return s.value, s.err
}

func TestKeyspaceDatabases(t *testing.T) {
	stub := &stubKeyBrowser{stubService: stubService{name: "stub"}, count: 16}
	srv := newConsoleServer(t, stub)
	rec := doJSON(t, srv, http.MethodGet, "/api/profiles/default/services/stub/databases", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	var got databasesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Count != 16 || got.Error != "" {
		t.Fatalf("response = %+v, want count 16", got)
	}
}

func TestKeyspaceKeys(t *testing.T) {
	stub := &stubKeyBrowser{
		stubService: stubService{name: "stub"},
		page: &service.KeyPage{
			Keys:   []service.KeyEntry{{Name: "k1", Type: "string", TTL: -1}},
			Cursor: 42,
		},
	}
	srv := newConsoleServer(t, stub)
	rec := doJSON(t, srv, http.MethodGet, "/api/profiles/default/services/stub/keys?db=2&pattern=k*", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	var got keysResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Keys) != 1 || got.Keys[0].Name != "k1" || got.Cursor != 42 {
		t.Fatalf("response = %+v", got)
	}
	if stub.gotDB != 2 {
		t.Errorf("db = %d, want 2", stub.gotDB)
	}
}

func TestKeyspaceValue(t *testing.T) {
	stub := &stubKeyBrowser{
		stubService: stubService{name: "stub"},
		value:       &service.KeyValue{Key: "greeting", Type: "string", String: "hello", Length: 5, TTL: -1},
	}
	srv := newConsoleServer(t, stub)
	rec := doJSON(t, srv, http.MethodGet, "/api/profiles/default/services/stub/key?key=greeting", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	var got valueResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.KeyValue == nil || got.Type != "string" || got.String != "hello" {
		t.Fatalf("response = %+v", got)
	}
	if stub.gotKey != "greeting" {
		t.Errorf("key = %q, want greeting", stub.gotKey)
	}
}

func TestKeyspaceValueRequiresKey(t *testing.T) {
	srv := newConsoleServer(t, &stubKeyBrowser{stubService: stubService{name: "stub"}})
	rec := doJSON(t, srv, http.MethodGet, "/api/profiles/default/services/stub/key", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body %q)", rec.Code, rec.Body.String())
	}
}

func TestKeyspaceConnectionError(t *testing.T) {
	stub := &stubKeyBrowser{stubService: stubService{name: "stub"}, err: errors.New("connection refused")}
	srv := newConsoleServer(t, stub)
	rec := doJSON(t, srv, http.MethodGet, "/api/profiles/default/services/stub/keys", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	var got keysResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Error == "" || len(got.Keys) != 0 {
		t.Fatalf("response = %+v, want error envelope with no keys", got)
	}
}

func TestKeyspaceUnsupportedService(t *testing.T) {
	// A plain stubService does not implement service.KeyBrowser.
	srv := newConsoleServer(t, stubService{name: "stub"})
	rec := doJSON(t, srv, http.MethodGet, "/api/profiles/default/services/stub/keys", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body %q)", rec.Code, rec.Body.String())
	}
}
