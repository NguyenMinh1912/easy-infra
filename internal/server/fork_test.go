package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/minhnc/easy-infra/internal/service"
)

// TestForkUnsupportedService forks redis, which has no container provisioning
// yet, so the session finishes "unsupported" — exercising start, persistence,
// and polling of the fork endpoint without touching Docker.
func TestForkUnsupportedService(t *testing.T) {
	paths, reg := initProject(t, "postgres", "redis")
	srv := New(reg, regFrom(t, paths), emptyUI)

	rec := doRequest(t, srv, http.MethodPost, "/api/services/redis/fork", "")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("start code = %d, want 202 (body %q)", rec.Code, rec.Body.String())
	}
	sess := decodeSession(t, rec)
	if sess.ID == "" || sess.Kind != "fork" {
		t.Fatalf("session = %+v, want a fork session with an id", sess)
	}

	poll := pollUntilDone(t, srv, sess.ID)
	if poll.Session.Status != "unsupported" {
		t.Errorf("final status = %q, want unsupported", poll.Session.Status)
	}
	var joined strings.Builder
	for _, l := range poll.Logs {
		joined.WriteString(l.Line)
		joined.WriteByte('\n')
	}
	if want := `forking "redis" to a local container is not supported yet`; !strings.Contains(joined.String(), want) {
		t.Errorf("logs missing %q\ngot:\n%s", want, joined.String())
	}
}

func TestForkUnknownService(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, regFrom(t, paths), emptyUI)
	// redis is not defined in the active profile.
	rec := doRequest(t, srv, http.MethodPost, "/api/services/redis/fork", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("code = %d, want 404 (body %q)", rec.Code, rec.Body.String())
	}
}

func TestForkSnapshotNotFound(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, regFrom(t, paths), emptyUI)
	// A snapshot id that does not exist is rejected before any provisioning.
	rec := doRequest(t, srv, http.MethodPost, "/api/services/postgres/fork", `{"snapshot":"nope"}`)
	if rec.Code != http.StatusNotFound {
		t.Errorf("code = %d, want 404 (body %q)", rec.Code, rec.Body.String())
	}
}

func TestForkInvalidPort(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, regFrom(t, paths), emptyUI)
	// An out-of-range local port is rejected before any provisioning.
	rec := doRequest(t, srv, http.MethodPost, "/api/services/postgres/fork", `{"port":70000}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("code = %d, want 400 (body %q)", rec.Code, rec.Body.String())
	}
}

func TestForkNotInitialized(t *testing.T) {
	srv := New(service.DefaultRegistry(), regFrom(t, newPaths(t)), emptyUI)
	rec := doRequest(t, srv, http.MethodPost, "/api/services/postgres/fork", "")
	if rec.Code != http.StatusConflict {
		t.Errorf("code = %d, want 409", rec.Code)
	}
}
