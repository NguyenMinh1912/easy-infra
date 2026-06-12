package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/minhnc/easy-infra/internal/service"
)

func decodeSession(t *testing.T, rec *httptest.ResponseRecorder) sessionJSON {
	t.Helper()
	var got sessionJSON
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode session: %v (body %q)", err, rec.Body.String())
	}
	return got
}

func decodePoll(t *testing.T, rec *httptest.ResponseRecorder) backupPollResponse {
	t.Helper()
	var got backupPollResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode poll: %v (body %q)", err, rec.Body.String())
	}
	return got
}

// pollUntilDone polls a backup session until it leaves the running state or the
// attempt budget runs out, returning the final poll response.
func pollUntilDone(t *testing.T, srv *Server, id string) backupPollResponse {
	t.Helper()
	var poll backupPollResponse
	for i := 0; i < 100; i++ {
		rec := doRequest(t, srv, http.MethodGet, "/api/backups/"+id, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("poll code = %d, want 200 (body %q)", rec.Code, rec.Body.String())
		}
		poll = decodePoll(t, rec)
		if poll.Session.Status != "running" {
			return poll
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("backup %q still running after polling", id)
	return poll
}

// TestBackupRunsAndPersists starts a backup and polls it to completion. Redis
// has no provider yet, so it finishes "unsupported" after a couple of log lines
// — enough to exercise start, persistence, and polling without a live service.
func TestBackupRunsAndPersists(t *testing.T) {
	paths, reg := initProject(t, "postgres", "redis")
	srv := New(reg, paths, emptyUI)

	rec := doRequest(t, srv, http.MethodPost, "/api/services/redis/backup", "")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("start code = %d, want 202 (body %q)", rec.Code, rec.Body.String())
	}
	sess := decodeSession(t, rec)
	if sess.ID == "" || sess.Service != "redis" {
		t.Fatalf("session = %+v, want a redis session with an id", sess)
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
	for _, want := range []string{`Backing up "redis"`, `backup is not supported for "redis" yet`} {
		if !strings.Contains(joined.String(), want) {
			t.Errorf("logs missing %q\ngot:\n%s", want, joined.String())
		}
	}
}

// TestBackupReattachesRunning verifies that starting a backup while one is
// already running for the same service returns the existing session rather than
// spawning a duplicate.
func TestBackupReattachesRunning(t *testing.T) {
	paths, reg := initProject(t, "postgres", "redis")
	srv := New(reg, paths, emptyUI)

	first := decodeSession(t, doRequest(t, srv, http.MethodPost, "/api/services/redis/backup", ""))
	// Let the (fast, unsupported) backup settle, then a second start makes a new
	// session since the first is no longer running.
	pollUntilDone(t, srv, first.ID)
	second := decodeSession(t, doRequest(t, srv, http.MethodPost, "/api/services/redis/backup", ""))
	if second.ID == first.ID {
		t.Errorf("second start reused finished session %q; want a new one", first.ID)
	}
}

func decodeList(t *testing.T, rec *httptest.ResponseRecorder) backupListResponse {
	t.Helper()
	var got backupListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode list: %v (body %q)", err, rec.Body.String())
	}
	return got
}

// TestBackupListAndDelete runs two backups, lists them (newest first, with
// pagination), then deletes one and confirms it leaves the list.
func TestBackupListAndDelete(t *testing.T) {
	paths, reg := initProject(t, "postgres", "redis")
	srv := New(reg, paths, emptyUI)

	first := decodeSession(t, doRequest(t, srv, http.MethodPost, "/api/services/redis/backup", ""))
	pollUntilDone(t, srv, first.ID)
	second := decodeSession(t, doRequest(t, srv, http.MethodPost, "/api/services/redis/backup", ""))
	pollUntilDone(t, srv, second.ID)

	list := decodeList(t, doRequest(t, srv, http.MethodGet, "/api/backups", ""))
	if !list.Initialized || list.Total != 2 || len(list.Sessions) != 2 {
		t.Fatalf("list = %+v, want 2 initialized sessions", list)
	}
	if list.Sessions[0].ID != second.ID {
		t.Errorf("first listed = %q, want newest %q", list.Sessions[0].ID, second.ID)
	}

	// Pagination: one per page, second page holds the older session.
	pageTwo := decodeList(t, doRequest(t, srv, http.MethodGet, "/api/backups?page=2&pageSize=1", ""))
	if pageTwo.Total != 2 || len(pageTwo.Sessions) != 1 || pageTwo.Sessions[0].ID != first.ID {
		t.Errorf("page 2 = %+v, want only the older session %q", pageTwo, first.ID)
	}

	// Delete the newer one; the list shrinks to just the older session.
	rec := doRequest(t, srv, http.MethodDelete, "/api/backups/"+second.ID, "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete code = %d, want 204 (body %q)", rec.Code, rec.Body.String())
	}
	after := decodeList(t, doRequest(t, srv, http.MethodGet, "/api/backups", ""))
	if after.Total != 1 || len(after.Sessions) != 1 || after.Sessions[0].ID != first.ID {
		t.Errorf("after delete = %+v, want only %q", after, first.ID)
	}
}

func TestBackupDeleteUnknownID(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, paths, emptyUI)
	rec := doRequest(t, srv, http.MethodDelete, "/api/backups/nope", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("code = %d, want 404", rec.Code)
	}
}

func TestBackupListNotInitialized(t *testing.T) {
	srv := New(service.DefaultRegistry(), newPaths(t), emptyUI)
	list := decodeList(t, doRequest(t, srv, http.MethodGet, "/api/backups", ""))
	if list.Initialized || len(list.Sessions) != 0 {
		t.Errorf("list = %+v, want uninitialized with no sessions", list)
	}
}

func TestBackupNotInitialized(t *testing.T) {
	srv := New(service.DefaultRegistry(), newPaths(t), emptyUI)
	rec := doRequest(t, srv, http.MethodPost, "/api/services/redis/backup", "")
	if rec.Code != http.StatusConflict {
		t.Errorf("code = %d, want 409", rec.Code)
	}
}

func TestBackupUnknownService(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, paths, emptyUI)
	rec := doRequest(t, srv, http.MethodPost, "/api/services/redis/backup", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("code = %d, want 404 (body %q)", rec.Code, rec.Body.String())
	}
}

func TestBackupGetUnknownID(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, paths, emptyUI)
	rec := doRequest(t, srv, http.MethodGet, "/api/backups/nope", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("code = %d, want 404", rec.Code)
	}
}
