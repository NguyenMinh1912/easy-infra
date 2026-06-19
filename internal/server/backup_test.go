package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	profilepkg "github.com/minhnc/easy-infra/internal/profile"
	"github.com/minhnc/easy-infra/internal/project"
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
	srv := New(reg, regFrom(t, paths), emptyUI)

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
	srv := New(reg, regFrom(t, paths), emptyUI)

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
	srv := New(reg, regFrom(t, paths), emptyUI)

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
	srv := New(reg, regFrom(t, paths), emptyUI)
	rec := doRequest(t, srv, http.MethodDelete, "/api/backups/nope", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("code = %d, want 404", rec.Code)
	}
}

func TestBackupListNotInitialized(t *testing.T) {
	srv := New(service.DefaultRegistry(), regFrom(t, newPaths(t)), emptyUI)
	list := decodeList(t, doRequest(t, srv, http.MethodGet, "/api/backups", ""))
	if list.Initialized || len(list.Sessions) != 0 {
		t.Errorf("list = %+v, want uninitialized with no sessions", list)
	}
}

func TestBackupNotInitialized(t *testing.T) {
	srv := New(service.DefaultRegistry(), regFrom(t, newPaths(t)), emptyUI)
	rec := doRequest(t, srv, http.MethodPost, "/api/services/redis/backup", "")
	if rec.Code != http.StatusConflict {
		t.Errorf("code = %d, want 409", rec.Code)
	}
}

func TestBackupUnknownService(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, regFrom(t, paths), emptyUI)
	rec := doRequest(t, srv, http.MethodPost, "/api/services/redis/backup", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("code = %d, want 404 (body %q)", rec.Code, rec.Body.String())
	}
}

func decodeBackupOptions(t *testing.T, rec *httptest.ResponseRecorder) backupOptionsResponse {
	t.Helper()
	var got backupOptionsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode backup options: %v (body %q)", err, rec.Body.String())
	}
	return got
}

// TestBackupOptionsNoBuckets confirms a service without a bucket concept
// (postgres) reports empty options, so the UI offers a plain confirmation.
func TestBackupOptionsNoBuckets(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, regFrom(t, paths), emptyUI)

	rec := doRequest(t, srv, http.MethodGet, "/api/services/postgres/backup-options", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	opts := decodeBackupOptions(t, rec)
	if len(opts.Buckets) != 0 || len(opts.Selected) != 0 {
		t.Errorf("options = %+v, want empty buckets/selected", opts)
	}
}

// addProfile scaffolds a non-active profile with the given services, so a test
// can have a profile the UI views that differs from the active one.
func addProfile(t *testing.T, paths project.Paths, reg *service.Registry, name string, services ...string) {
	t.Helper()
	prof, err := profilepkg.Scaffold(reg, services...)
	if err != nil {
		t.Fatalf("Scaffold profile %q: %v", name, err)
	}
	if err := prof.Save(paths.ProfilePath(name)); err != nil {
		t.Fatalf("Save profile %q: %v", name, err)
	}
}

// TestBackupOptionsTargetsRequestedProfile reproduces EFA-61: the active profile
// has no minio, but a separate "dev" profile does. Without a profile the lookup
// fails against the active profile; passing ?profile=dev scopes it to the viewed
// profile so the dialog can load the buckets the user actually wants to back up.
func TestBackupOptionsTargetsRequestedProfile(t *testing.T) {
	// Active profile "default" has only postgres (no minio).
	paths, reg := initProject(t, "postgres")
	// The profile the user is viewing ("dev") defines minio.
	addProfile(t, paths, reg, "dev", "minio")
	srv := New(reg, regFrom(t, paths), emptyUI)

	// Against the active profile, minio is not defined — the original failure.
	rec := doRequest(t, srv, http.MethodGet, "/api/services/minio/backup-options", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("active-profile code = %d, want 404 (body %q)", rec.Code, rec.Body.String())
	}

	// Scoping to the viewed profile finds minio and resolves options (the store
	// is unreachable in tests, so the failure is reported inline with 200).
	rec = doRequest(t, srv, http.MethodGet, "/api/services/minio/backup-options?profile=dev", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("scoped code = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
}

// TestBackupOptionsUnknownProfile confirms an explicit but nonexistent profile is
// reported as a 404 rather than silently falling back to the active profile.
func TestBackupOptionsUnknownProfile(t *testing.T) {
	paths, reg := initProject(t, "minio")
	srv := New(reg, regFrom(t, paths), emptyUI)

	rec := doRequest(t, srv, http.MethodGet, "/api/services/minio/backup-options?profile=ghost", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("code = %d, want 404 (body %q)", rec.Code, rec.Body.String())
	}
}

// TestBackupOptionsStoreUnreachable confirms that for an object store whose
// server cannot be reached, the endpoint still returns 200 with the failure
// reported inline rather than failing the request.
func TestBackupOptionsStoreUnreachable(t *testing.T) {
	paths, reg := initProject(t, "minio")
	srv := New(reg, regFrom(t, paths), emptyUI)

	rec := doRequest(t, srv, http.MethodGet, "/api/services/minio/backup-options", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	opts := decodeBackupOptions(t, rec)
	if opts.Error == "" {
		t.Errorf("options = %+v, want an inline error for the unreachable store", opts)
	}
}

// TestDefaultBucketSelection covers the default-selection rule: configured
// buckets that exist are selected, missing ones are dropped, and an empty
// configuration selects every live bucket.
func TestDefaultBucketSelection(t *testing.T) {
	live := []string{"assets", "logs", "uploads"}

	got := defaultBucketSelection(live, []string{"assets", "uploads", "ghost"})
	if len(got) != 2 || got[0] != "assets" || got[1] != "uploads" {
		t.Errorf("with config = %v, want [assets uploads]", got)
	}

	if got := defaultBucketSelection(live, nil); len(got) != 3 {
		t.Errorf("with no config = %v, want all 3 live buckets", got)
	}
}

func TestBackupGetUnknownID(t *testing.T) {
	paths, reg := initProject(t, "postgres")
	srv := New(reg, regFrom(t, paths), emptyUI)
	rec := doRequest(t, srv, http.MethodGet, "/api/backups/nope", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("code = %d, want 404", rec.Code)
	}
}
