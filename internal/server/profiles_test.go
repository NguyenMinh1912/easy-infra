package server

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/minhnc/easy-infra/internal/service"
)

// stubService is a minimal Service whose connection check outcome is
// controlled by healthErr, letting the handler test cover the reachable and
// unreachable paths without a live server.
type stubService struct {
	name      string
	healthErr error
}

func (s stubService) Name() string                               { return s.name }
func (stubService) DefaultDefinition() service.Config            { return service.Config{} }
func (stubService) ValidateDefinition(service.Config) error      { return nil }
func (stubService) DefaultEnv() service.Config                   { return service.Config{} }
func (stubService) ValidateEnv(service.Config) error             { return nil }
func (stubService) Apply(context.Context, service.Spec) error    { return nil }
func (s stubService) Health(context.Context, service.Spec) error { return s.healthErr }
func (stubService) Backup(context.Context, service.Spec) error   { return nil }
func (stubService) Clean(context.Context, service.Spec) error    { return nil }

func decodeCheck(t *testing.T, rec *http.Response, body []byte) checkConnectionResponse {
	t.Helper()
	var got checkConnectionResponse
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode check response: %v (body %q)", err, body)
	}
	return got
}

func TestCheckConnectionUnknownService(t *testing.T) {
	srv := New(service.DefaultRegistry(), newStore(t), emptyUI)
	rec := doJSON(t, srv, http.MethodPost, "/api/profiles/default/services/nope/check",
		checkConnectionRequest{Config: service.Config{}})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body %q)", rec.Code, rec.Body.String())
	}
}

func TestCheckConnectionInvalidConfig(t *testing.T) {
	srv := New(service.DefaultRegistry(), newStore(t), emptyUI)
	// An empty postgres env fails validation before any dial is attempted.
	rec := doJSON(t, srv, http.MethodPost, "/api/profiles/default/services/postgres/check",
		checkConnectionRequest{Config: service.Config{}})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	got := decodeCheck(t, rec.Result(), rec.Body.Bytes())
	if got.OK || got.Error == "" {
		t.Fatalf("response = %+v, want ok=false with an error", got)
	}
}

func TestCheckConnectionReachable(t *testing.T) {
	reg := service.NewRegistry()
	if err := reg.Register(stubService{name: "stub"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	srv := New(reg, newStore(t), emptyUI)
	rec := doJSON(t, srv, http.MethodPost, "/api/profiles/default/services/stub/check",
		checkConnectionRequest{Config: service.Config{"host": "x"}})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	got := decodeCheck(t, rec.Result(), rec.Body.Bytes())
	if !got.OK || got.Error != "" {
		t.Fatalf("response = %+v, want ok=true with no error", got)
	}
}
