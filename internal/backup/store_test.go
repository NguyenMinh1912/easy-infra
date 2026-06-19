package backup

import (
	"path/filepath"
	"testing"
)

func openTemp(t *testing.T) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "backups.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestSessionLifecycle(t *testing.T) {
	store := openTemp(t)

	sess, err := store.CreateSession(1, "postgres", "default", KindBackup)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if sess.Status != StatusRunning {
		t.Errorf("new session status = %q, want running", sess.Status)
	}

	// A running session is re-attachable; a second start should find it.
	if got, ok, err := store.RunningForService(1, "postgres", "default", KindBackup); err != nil || !ok {
		t.Fatalf("RunningForService = (%v, %v, %v), want a running session", got, ok, err)
	} else if got.ID != sess.ID {
		t.Errorf("RunningForService id = %q, want %q", got.ID, sess.ID)
	}

	for _, line := range []string{"first", "second", "third"} {
		if err := store.AppendLog(sess.ID, line); err != nil {
			t.Fatalf("AppendLog: %v", err)
		}
	}

	// Cursor semantics: after 0 returns all, after 1 returns the tail.
	all, err := store.Logs(sess.ID, 0)
	if err != nil {
		t.Fatalf("Logs: %v", err)
	}
	if len(all) != 3 || all[0].Seq != 1 || all[0].Line != "first" || all[2].Seq != 3 {
		t.Fatalf("Logs(0) = %+v, want seq 1..3", all)
	}
	tail, err := store.Logs(sess.ID, 1)
	if err != nil {
		t.Fatalf("Logs: %v", err)
	}
	if len(tail) != 2 || tail[0].Seq != 2 {
		t.Errorf("Logs(1) = %+v, want seq 2,3", tail)
	}

	if err := store.Finish(sess.ID, StatusSuccess, "20260101T000000Z", ""); err != nil {
		t.Fatalf("Finish: %v", err)
	}
	got, err := store.Get(sess.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != StatusSuccess || got.Snapshot != "20260101T000000Z" {
		t.Errorf("finished session = %+v, want success with snapshot", got)
	}
	// No longer running, so nothing to re-attach to.
	if _, ok, _ := store.RunningForService(1, "postgres", "default", KindBackup); ok {
		t.Error("RunningForService still reports a running session after Finish")
	}
}

// TestListAndDeleteSessions exercises paginated listing (newest first) and
// deletion of a session together with its logs.
func TestListAndDeleteSessions(t *testing.T) {
	store := openTemp(t)

	var ids []string
	for _, svc := range []string{"postgres", "redis", "minio"} {
		sess, err := store.CreateSession(1, svc, "default", KindBackup)
		if err != nil {
			t.Fatalf("CreateSession: %v", err)
		}
		ids = append(ids, sess.ID)
	}

	if n, err := store.CountSessions(1); err != nil || n != 3 {
		t.Fatalf("CountSessions = (%d, %v), want 3", n, err)
	}

	// Newest first: the last created session leads the list.
	page, err := store.ListSessions(1, 2, 0)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(page) != 2 || page[0].Service != "minio" {
		t.Fatalf("ListSessions(2,0) = %+v, want minio,redis", page)
	}
	rest, err := store.ListSessions(1, 2, 2)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(rest) != 1 || rest[0].Service != "postgres" {
		t.Fatalf("ListSessions(2,2) = %+v, want postgres", rest)
	}

	// Deleting drops the session and its logs.
	if err := store.AppendLog(ids[0], "line"); err != nil {
		t.Fatalf("AppendLog: %v", err)
	}
	if err := store.Delete(ids[0]); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Get(ids[0]); err != ErrNotFound {
		t.Errorf("Get after delete = %v, want ErrNotFound", err)
	}
	if logs, err := store.Logs(ids[0], 0); err != nil || len(logs) != 0 {
		t.Errorf("Logs after delete = (%v, %v), want none", logs, err)
	}
	if n, err := store.CountSessions(1); err != nil || n != 2 {
		t.Errorf("CountSessions after delete = (%d, %v), want 2", n, err)
	}
	if err := store.Delete("nope"); err != ErrNotFound {
		t.Errorf("Delete unknown = %v, want ErrNotFound", err)
	}
}

func TestGetUnknown(t *testing.T) {
	store := openTemp(t)
	if _, err := store.Get("nope"); err != ErrNotFound {
		t.Errorf("Get unknown err = %v, want ErrNotFound", err)
	}
}

// TestReconcileRunningOnOpen verifies that a session left "running" by a crashed
// process is marked errored when the store reopens — its goroutine is gone.
func TestReconcileRunningOnOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backups.db")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	sess, err := store.CreateSession(1, "postgres", "default", KindBackup)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	store.Close()

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer reopened.Close()
	got, err := reopened.Get(sess.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != StatusError {
		t.Errorf("status after reopen = %q, want error (interrupted)", got.Status)
	}
}
