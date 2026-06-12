package server

import (
	"net/http"
	"strings"
	"testing"

	"github.com/minhnc/easy-infra/internal/service"
)

// TestBackupServiceStreams exercises the streaming path end to end. Redis has no
// backup provider yet, so it streams a couple of log events and finishes with a
// terminal "unsupported" done event — enough to assert the SSE framing without
// needing a live service.
func TestBackupServiceStreams(t *testing.T) {
	paths, reg := initProject(t, "postgres", "redis")
	srv := New(reg, paths, emptyUI)

	rec := doRequest(t, srv, http.MethodPost, "/api/services/redis/backup", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}

	body := rec.Body.String()
	for _, want := range []string{
		"event: log\ndata: Backing up \"redis\"",
		"event: log\ndata: backup is not supported for \"redis\" yet",
		"event: done\ndata: {\"status\":\"unsupported\"}",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("stream missing %q\nfull body:\n%s", want, body)
		}
	}
}

// TestBackupServiceNotInitialized backs up in a folder with no project, which is
// reported as a JSON error before any streaming starts.
func TestBackupServiceNotInitialized(t *testing.T) {
	srv := New(service.DefaultRegistry(), newPaths(t), emptyUI)
	rec := doRequest(t, srv, http.MethodPost, "/api/services/redis/backup", "")
	if rec.Code != http.StatusConflict {
		t.Errorf("code = %d, want 409", rec.Code)
	}
}

// TestBackupServiceUnknown asks to back up a service the active profile does not
// define, which is a 404 before streaming.
func TestBackupServiceUnknown(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, paths, emptyUI)

	rec := doRequest(t, srv, http.MethodPost, "/api/services/redis/backup", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("code = %d, want 404 (body %q)", rec.Code, rec.Body.String())
	}
}
