package server

import (
	"bytes"
	"context"
	"encoding/json"
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
	profilepkg "github.com/minhnc/easy-infra/internal/profile"
	"github.com/minhnc/easy-infra/internal/project"
	"github.com/minhnc/easy-infra/internal/service"
)

// targetProfile resolves the profile a per-service operation acts on. The UI
// scopes its service pages to a profile and passes it as the `profile` query
// parameter, so an action backs up the profile being viewed rather than whatever
// happens to be active. When the parameter is absent (e.g. CLI-driven callers)
// the active profile is used, preserving the previous behaviour. An explicit but
// unknown profile is a 404; a missing/invalid active profile stays a 409, each
// matching how it is reached. On failure it writes the response and returns
// ok=false.
func (s *Server) targetProfile(w http.ResponseWriter, r *http.Request, proj *project.Project) (string, *profilepkg.Profile, bool) {
	if name := r.URL.Query().Get("profile"); name != "" {
		prof, err := proj.LoadProfile(name)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return "", nil, false
		}
		return name, prof, true
	}
	name, prof, err := proj.ActiveProfile()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return "", nil, false
	}
	return name, prof, true
}

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

// Start launches a session of kind for service/profile, writing its verbose log
// into the store. If one of the same kind is already running for this
// service/profile, that session is returned instead of starting a second one.
// The run closure performs the actual work; its returned error classifies the
// outcome (a service.ErrNotImplemented becomes "unsupported", a cancelled
// context becomes "cancelled"). On success the session records snapshot. cleanup
// runs when the session is cancelled or fails: a backup passes its freshly
// written (and possibly partial) snapshot folder so it is dropped, while an
// apply restores from an existing snapshot and passes nil to leave it untouched.
func (m *backupManager) Start(store *backup.Store, svcName, profile string, kind backup.Kind, snapshot string, cleanup func(), run func(ctx context.Context, w io.Writer) error) (backup.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok, err := store.RunningForService(svcName, profile, kind); err != nil {
		return backup.Session{}, err
	} else if ok {
		return *existing, nil
	}

	sess, err := store.CreateSession(svcName, profile, kind)
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
			if cleanup != nil {
				cleanup()
			}
			_ = store.Finish(sess.ID, backup.StatusCancelled, "", string(kind)+" cancelled")
		case err != nil:
			if cleanup != nil {
				cleanup()
			}
			_ = store.Finish(sess.ID, backup.StatusError, "", err.Error())
		default:
			_ = store.Finish(sess.ID, backup.StatusSuccess, snapshot, "")
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

	// An optional body selects which buckets to back up (minio); an empty or
	// absent list backs up everything, preserving the default behaviour.
	var body struct {
		Buckets []string `json:"buckets"`
	}
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err != io.EOF {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
			return
		}
	}

	proj, err := project.Load(s.paths, s.reg)
	if err != nil {
		s.writeProjectError(w, err)
		return
	}
	profileName, prof, ok := s.targetProfile(w, r, proj)
	if !ok {
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
		Definition: env,
		Env:        env,
		BackupDir:  dir,
		Buckets:    body.Buckets,
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

	// A cancelled or failed backup may have written a partial artifact; drop the
	// whole snapshot folder so no truncated snapshot is left behind.
	cleanup := func() { _ = os.RemoveAll(dir) }
	sess, err := s.backups.Start(store, name, profileName, backup.KindBackup, filepath.Base(dir), cleanup, run)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, toSessionJSON(sess))
}

// handleListSnapshots returns the backup snapshot ids available to the active
// profile, newest first, so the UI can offer them as the versions an apply may
// restore. The {name} service must be defined in the profile; the snapshots
// themselves are profile-wide (one snapshot captures the whole profile).
func (s *Server) handleListSnapshots(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	proj, err := project.Load(s.paths, s.reg)
	if err != nil {
		s.writeProjectError(w, err)
		return
	}
	profileName, prof, ok := s.targetProfile(w, r, proj)
	if !ok {
		return
	}
	if _, ok := prof.Services[name]; !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("service %q is not defined in profile %q", name, profileName))
		return
	}

	ids, err := service.ListSnapshots(profileName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// ListSnapshots is oldest-first; present newest-first so the most recent
	// version leads the selection list.
	snapshots := make([]string, 0, len(ids))
	for i := len(ids) - 1; i >= 0; i-- {
		snapshots = append(snapshots, ids[i])
	}
	writeJSON(w, http.StatusOK, snapshotsResponse{Snapshots: snapshots})
}

// backupOptionsTimeout bounds the live bucket listing so an unreachable store
// fails fast with a clear reason instead of hanging the dialog.
const backupOptionsTimeout = 15 * time.Second

// handleBackupOptions reports, for the active profile's service, which buckets a
// backup can capture and which to select by default. Only object-store services
// (those implementing service.Browser, i.e. minio) have buckets: the candidate
// list is the store's live buckets, and the default selection is the buckets
// declared in the profile config (or all of them when none are configured).
// Services without a bucket concept return empty lists, so the UI falls back to
// a plain "back up everything" confirmation. A store-unreachable error is
// reported inline (HTTP 200) like the browse endpoints, falling back to the
// configured buckets so the user is not blocked.
func (s *Server) handleBackupOptions(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	proj, err := project.Load(s.paths, s.reg)
	if err != nil {
		s.writeProjectError(w, err)
		return
	}
	profileName, prof, ok := s.targetProfile(w, r, proj)
	if !ok {
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

	browser, ok := svc.(service.Browser)
	if !ok {
		// No bucket concept: the UI offers a plain confirmation.
		writeJSON(w, http.StatusOK, backupOptionsResponse{Buckets: []string{}, Selected: []string{}})
		return
	}

	configured, err := service.ConfiguredBuckets(env)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), backupOptionsTimeout)
	defer cancel()
	live, err := browser.Buckets(ctx, service.Spec{Profile: profileName, Env: env})
	if err != nil {
		// Can't list the store: offer the configured buckets so the user can
		// still choose, and surface why the live list is missing.
		writeJSON(w, http.StatusOK, backupOptionsResponse{
			Buckets:  configured,
			Selected: configured,
			Error:    err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, backupOptionsResponse{
		Buckets:  nonNil(live),
		Selected: defaultBucketSelection(live, configured),
	})
}

// defaultBucketSelection picks which of the store's live buckets to select by
// default: the configured ones that actually exist, or — when none are
// configured — every live bucket (matching the back-up-everything default).
func defaultBucketSelection(live, configured []string) []string {
	if len(configured) == 0 {
		return nonNil(live)
	}
	liveSet := make(map[string]bool, len(live))
	for _, b := range live {
		liveSet[b] = true
	}
	selected := make([]string, 0, len(configured))
	for _, b := range configured {
		if liveSet[b] {
			selected = append(selected, b)
		}
	}
	return selected
}

// nonNil returns xs, or an empty slice when xs is nil, so it serialises as a
// JSON array rather than null.
func nonNil(xs []string) []string {
	if xs == nil {
		return []string{}
	}
	return xs
}

// handleStartApply restores a single service for the active profile from a
// chosen snapshot in the background, returning the (new or already-running)
// session as JSON. The body selects which snapshot version to apply; an empty
// snapshot applies the latest. The apply reads an existing snapshot, so a
// failure leaves it in place (no cleanup), unlike a backup.
func (s *Server) handleStartApply(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	var body struct {
		Snapshot string `json:"snapshot"`
	}
	if r.Body != nil {
		// An empty body is allowed and means "latest"; only a malformed one errs.
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err != io.EOF {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
			return
		}
	}

	proj, err := project.Load(s.paths, s.reg)
	if err != nil {
		s.writeProjectError(w, err)
		return
	}
	profileName, prof, ok := s.targetProfile(w, r, proj)
	if !ok {
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

	// Validate the requested snapshot against the known list rather than trusting
	// the client, so a crafted id cannot escape the backups directory.
	if body.Snapshot != "" {
		ids, err := service.ListSnapshots(profileName)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !contains(ids, body.Snapshot) {
			writeError(w, http.StatusNotFound, fmt.Sprintf("snapshot %q not found for profile %q", body.Snapshot, profileName))
			return
		}
	}

	store, err := s.backups.Store()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	spec := service.Spec{
		Profile:    profileName,
		Definition: env,
		Env:        env,
		Snapshot:   body.Snapshot,
	}
	run := func(ctx context.Context, lw io.Writer) error {
		version := body.Snapshot
		if version == "" {
			version = "latest snapshot"
		}
		fmt.Fprintf(lw, "Applying %q (profile %q) from %s\n", name, profileName, version)
		spec.Log = lw
		err := svc.Apply(ctx, spec)
		if errors.Is(err, service.ErrNotImplemented) {
			fmt.Fprintf(lw, "apply is not supported for %q yet\n", name)
		}
		return err
	}

	sess, err := s.backups.Start(store, name, profileName, backup.KindApply, body.Snapshot, nil, run)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, toSessionJSON(sess))
}

// contains reports whether s is in xs.
func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// defaultPageSize and maxPageSize bound the backup list pagination.
const (
	defaultPageSize = 10
	maxPageSize     = 100
)

// handleListBackups returns a page of backup sessions, newest first, for the
// Backups screen. The list spans every service and profile, so the whole backup
// history is browsable in one place. An uninitialized project has no sessions
// (and we avoid creating the database for a bare folder).
func (s *Server) handleListBackups(w http.ResponseWriter, r *http.Request) {
	page, pageSize := paginationParams(r)

	if _, err := project.Load(s.paths, s.reg); err != nil {
		if errors.Is(err, project.ErrNotInitialized) {
			writeJSON(w, http.StatusOK, backupListResponse{
				Sessions: []sessionJSON{}, Page: page, PageSize: pageSize,
			})
			return
		}
		s.writeProjectError(w, err)
		return
	}

	store, err := s.backups.Store()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	total, err := store.CountSessions()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sessions, err := store.ListSessions(pageSize, (page-1)*pageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	out := make([]sessionJSON, 0, len(sessions))
	for _, sess := range sessions {
		out = append(out, toSessionJSON(sess))
	}
	writeJSON(w, http.StatusOK, backupListResponse{
		Initialized: true,
		Sessions:    out,
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
	})
}

// handleDeleteBackup removes a finished backup session, its logs, and the
// snapshot folder it produced. A running session must be cancelled first — its
// goroutine is still writing — so deleting one is rejected with 409.
func (s *Server) handleDeleteBackup(w http.ResponseWriter, r *http.Request) {
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
	if sess.Status == backup.StatusRunning {
		writeError(w, http.StatusConflict, "cancel the running backup before deleting it")
		return
	}

	if err := store.Delete(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Drop the snapshot artifact on disk too, so deleting the record leaves
	// nothing behind. Best-effort: the record is already gone.
	if sess.Snapshot != "" {
		_ = os.RemoveAll(filepath.Join(service.BackupsDir(sess.Profile), sess.Snapshot))
	}
	w.WriteHeader(http.StatusNoContent)
}

// paginationParams reads the `page` (1-based) and `pageSize` query parameters,
// applying defaults and clamping the size so a client cannot request an
// unbounded page.
func paginationParams(r *http.Request) (page, pageSize int) {
	page, pageSize = 1, defaultPageSize
	if v, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && v > 0 {
		page = v
	}
	if v, err := strconv.Atoi(r.URL.Query().Get("pageSize")); err == nil && v > 0 {
		pageSize = v
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
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

// sessionJSON is the JSON shape of a backup or apply session.
type sessionJSON struct {
	ID        string `json:"id"`
	Service   string `json:"service"`
	Profile   string `json:"profile"`
	Kind      string `json:"kind"`
	Status    string `json:"status"`
	Snapshot  string `json:"snapshot,omitempty"`
	Error     string `json:"error,omitempty"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// snapshotsResponse is returned by GET /api/services/{name}/snapshots: the
// backup versions available to the active profile, newest first.
type snapshotsResponse struct {
	Snapshots []string `json:"snapshots"`
}

// backupOptionsResponse is returned by GET /api/services/{name}/backup-options:
// the buckets a backup can capture and the subset selected by default. Both are
// empty for services without a bucket concept. Error carries a store-unreachable
// reason without failing the request.
type backupOptionsResponse struct {
	Buckets  []string `json:"buckets"`
	Selected []string `json:"selected"`
	Error    string   `json:"error,omitempty"`
}

// backupPollResponse is returned by GET /api/backups/{id}: the session plus the
// log lines after the requested cursor.
type backupPollResponse struct {
	Session sessionJSON      `json:"session"`
	Logs    []backup.LogLine `json:"logs"`
}

// backupListResponse is returned by GET /api/backups: a page of sessions plus
// the total count and the page coordinates the client requested.
type backupListResponse struct {
	Initialized bool          `json:"initialized"`
	Sessions    []sessionJSON `json:"sessions"`
	Total       int           `json:"total"`
	Page        int           `json:"page"`
	PageSize    int           `json:"pageSize"`
}

func toSessionJSON(s backup.Session) sessionJSON {
	return sessionJSON{
		ID:        s.ID,
		Service:   s.Service,
		Profile:   s.Profile,
		Kind:      string(s.Kind),
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
