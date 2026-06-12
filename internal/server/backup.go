package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/minhnc/easy-infra/internal/backup"
	"github.com/minhnc/easy-infra/internal/project"
	"github.com/minhnc/easy-infra/internal/service"
)

// backupManager runs backups in the background and tracks the ones in flight so
// they can be cancelled. A backup runs on a context derived from
// context.Background(), not the request context, so it survives the HTTP request
// that started it — the UI observes progress by polling, and may disconnect and
// reconnect freely. The SQLite store is opened lazily on first use, so an
// uninitialized project never creates a database.
type backupManager struct {
	dbPath string

	once    sync.Once
	store   *backup.Store
	openErr error

	mu      sync.Mutex
	cancels map[string]context.CancelFunc
}

func newBackupManager(dbPath string) *backupManager {
	return &backupManager{dbPath: dbPath, cancels: make(map[string]context.CancelFunc)}
}

// Store opens the backup database on first call and memoizes the result.
func (m *backupManager) Store() (*backup.Store, error) {
	m.once.Do(func() { m.store, m.openErr = backup.Open(m.dbPath) })
	return m.store, m.openErr
}

// Start launches a backup for service/profile, writing its artifact into dir and
// its verbose log into the store. If a backup for this service/profile is
// already running, that session is returned instead of starting a second one.
// The run closure performs the actual work; its returned error classifies the
// outcome (a service.ErrNotImplemented becomes "unsupported", a cancelled
// context becomes "cancelled").
func (m *backupManager) Start(store *backup.Store, svcName, profile, dir string, run func(ctx context.Context, w io.Writer) error) (backup.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok, err := store.RunningForService(svcName, profile); err != nil {
		return backup.Session{}, err
	} else if ok {
		return *existing, nil
	}

	sess, err := store.CreateSession(svcName, profile)
	if err != nil {
		return backup.Session{}, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancels[sess.ID] = cancel

	go func() {
		w := &storeLogWriter{store: store, id: sess.ID}
		err := run(ctx, w)
		w.flush()

		m.mu.Lock()
		delete(m.cancels, sess.ID)
		m.mu.Unlock()

		switch {
		case errors.Is(err, service.ErrNotImplemented):
			_ = store.Finish(sess.ID, backup.StatusUnsupported, "", "")
		case errors.Is(err, context.Canceled):
			// A cancelled backup may have written a partial artifact; drop the
			// whole snapshot folder so no truncated snapshot is left behind.
			_ = os.RemoveAll(dir)
			_ = store.Finish(sess.ID, backup.StatusCancelled, "", "backup cancelled")
		case err != nil:
			_ = os.RemoveAll(dir)
			_ = store.Finish(sess.ID, backup.StatusError, "", err.Error())
		default:
			_ = store.Finish(sess.ID, backup.StatusSuccess, filepath.Base(dir), "")
		}
	}()

	return sess, nil
}

// Cancel stops a running backup if its id is known; finished or unknown ids are
// a no-op.
func (m *backupManager) Cancel(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cancel, ok := m.cancels[id]; ok {
		cancel()
	}
}

// handleStartBackup snapshots a single service for the active profile in the
// background and returns the (new or already-running) session as JSON. The
// per-service action mirrors where backup is offered in the UI; the
// whole-profile snapshot lives in the CLI (`easy-infra backup snapshot`).
func (s *Server) handleStartBackup(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	proj, err := project.Load(s.paths, s.reg)
	if err != nil {
		s.writeProjectError(w, err)
		return
	}
	profileName, prof, err := proj.ActiveProfile()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	env, ok := prof.Services[name]
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("service %q is not defined in profile %q", name, profileName))
		return
	}
	svc, ok := s.reg.Get(name)
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("unknown service %q", name))
		return
	}
	store, err := s.backups.Store()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// One fresh snapshot folder for this service's artifact.
	dir := service.NewSnapshotDir(profileName)
	spec := service.Spec{
		Profile:    profileName,
		Definition: proj.Config.Services[name],
		Env:        env,
		BackupDir:  dir,
	}
	run := func(ctx context.Context, lw io.Writer) error {
		fmt.Fprintf(lw, "Backing up %q (profile %q) into %s\n", name, profileName, dir)
		spec.Log = lw
		err := svc.Backup(ctx, spec)
		if errors.Is(err, service.ErrNotImplemented) {
			fmt.Fprintf(lw, "backup is not supported for %q yet\n", name)
		}
		return err
	}

	sess, err := s.backups.Start(store, name, profileName, dir, run)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, toSessionJSON(sess))
}

// handleGetBackup returns a session's current status plus any log lines after
// the `after` query cursor, so the UI can poll for incremental progress.
func (s *Server) handleGetBackup(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	store, err := s.backups.Store()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sess, err := store.Get(id)
	if errors.Is(err, backup.ErrNotFound) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("backup %q not found", id))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	logs, err := store.Logs(id, after)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if logs == nil {
		logs = []backup.LogLine{}
	}
	writeJSON(w, http.StatusOK, backupPollResponse{Session: toSessionJSON(sess), Logs: logs})
}

// handleCancelBackup cancels a running backup and returns the session's current
// state. The goroutine flips the status to "cancelled" shortly after, which the
// client picks up on its next poll.
func (s *Server) handleCancelBackup(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	store, err := s.backups.Store()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sess, err := store.Get(id)
	if errors.Is(err, backup.ErrNotFound) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("backup %q not found", id))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.backups.Cancel(id)
	writeJSON(w, http.StatusAccepted, toSessionJSON(sess))
}

// sessionJSON is the JSON shape of a backup session.
type sessionJSON struct {
	ID        string `json:"id"`
	Service   string `json:"service"`
	Profile   string `json:"profile"`
	Status    string `json:"status"`
	Snapshot  string `json:"snapshot,omitempty"`
	Error     string `json:"error,omitempty"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// backupPollResponse is returned by GET /api/backups/{id}: the session plus the
// log lines after the requested cursor.
type backupPollResponse struct {
	Session sessionJSON      `json:"session"`
	Logs    []backup.LogLine `json:"logs"`
}

func toSessionJSON(s backup.Session) sessionJSON {
	return sessionJSON{
		ID:        s.ID,
		Service:   s.Service,
		Profile:   s.Profile,
		Status:    string(s.Status),
		Snapshot:  s.Snapshot,
		Error:     s.Error,
		CreatedAt: s.CreatedAt.Format(time.RFC3339),
		UpdatedAt: s.UpdatedAt.Format(time.RFC3339),
	}
}

// storeLogWriter is an io.Writer wired to Spec.Log: it buffers until a newline
// and appends each complete line to the session's log, so a service's verbose
// progress is persisted line by line. Lifecycle log lines are newline-
// terminated; flush persists any trailing partial line when the backup ends.
type storeLogWriter struct {
	store *backup.Store
	id    string
	buf   []byte
}

func (w *storeLogWriter) Write(b []byte) (int, error) {
	w.buf = append(w.buf, b...)
	for {
		i := bytes.IndexByte(w.buf, '\n')
		if i < 0 {
			break
		}
		_ = w.store.AppendLog(w.id, string(w.buf[:i]))
		w.buf = w.buf[i+1:]
	}
	return len(b), nil
}

func (w *storeLogWriter) flush() {
	if len(w.buf) > 0 {
		_ = w.store.AppendLog(w.id, string(w.buf))
		w.buf = nil
	}
}
