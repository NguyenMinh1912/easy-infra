// Package backup persists backup sessions and their verbose logs in a small
// SQLite database, so a backup started from the UI survives the request that
// launched it: the browser can disconnect and reconnect (by polling) without
// losing progress, and finished runs remain inspectable.
//
// State elsewhere in easy-infra is tool-owned JSON; backup logs are append-heavy
// and queried incrementally (by line cursor), which is what a tiny embedded SQL
// store is good at. It lives at .easy-infra/backups.db alongside the JSON state.
package backup

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // pure-Go driver, registered as "sqlite"
)

// Status is the lifecycle state of a backup session.
type Status string

const (
	// StatusRunning is set while the backup goroutine is active.
	StatusRunning Status = "running"
	// StatusSuccess means the snapshot was written.
	StatusSuccess Status = "success"
	// StatusUnsupported means the service has no backup provider yet.
	StatusUnsupported Status = "unsupported"
	// StatusError means the backup failed; Session.Error has the reason.
	StatusError Status = "error"
	// StatusCancelled means the user cancelled the run before it finished.
	StatusCancelled Status = "cancelled"
)

// ErrNotFound is returned when a session id does not exist.
var ErrNotFound = errors.New("backup session not found")

// Session is one backup run: which service/profile, its current status, and —
// once finished — the snapshot folder or the failure reason.
type Session struct {
	ID        string
	Service   string
	Profile   string
	Status    Status
	Snapshot  string
	Error     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// LogLine is one verbose log line, identified by a per-session sequence number
// so clients can poll for "everything after seq N".
type LogLine struct {
	Seq  int64  `json:"seq"`
	Line string `json:"line"`
}

// Store is the SQLite-backed session/log store. Its zero value is not usable;
// construct it with Open.
type Store struct {
	db *sql.DB
}

// Open opens (creating if needed) the SQLite database at path, applies the
// schema, and reconciles any session left "running" by a previous process —
// those goroutines are gone, so they are marked as interrupted rather than
// lingering forever.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("creating backup db dir: %w", err)
	}
	// WAL + a busy timeout keep the single writer (a running backup) from
	// blocking pollers reading the same db.
	dsn := "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening backup db: %w", err)
	}
	// Serialize access: log writes are tiny and infrequent, so one connection
	// sidesteps SQLITE_BUSY without a meaningful throughput cost.
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the underlying database handle.
func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	const schema = `
CREATE TABLE IF NOT EXISTS backup_sessions (
	id         TEXT PRIMARY KEY,
	service    TEXT NOT NULL,
	profile    TEXT NOT NULL,
	status     TEXT NOT NULL,
	snapshot   TEXT NOT NULL DEFAULT '',
	error      TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_service ON backup_sessions(service, profile, created_at);
CREATE TABLE IF NOT EXISTS backup_logs (
	session_id TEXT NOT NULL,
	seq        INTEGER NOT NULL,
	line       TEXT NOT NULL,
	PRIMARY KEY (session_id, seq)
);`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("migrating backup db: %w", err)
	}
	// A session still "running" after a restart has no goroutine behind it.
	if _, err := s.db.Exec(
		`UPDATE backup_sessions SET status = ?, error = ?, updated_at = ? WHERE status = ?`,
		StatusError, "interrupted by server restart", nowMillis(), StatusRunning,
	); err != nil {
		return fmt.Errorf("reconciling running sessions: %w", err)
	}
	return nil
}

// CreateSession inserts a new running session for service/profile and returns
// it with a fresh id and timestamps.
func (s *Store) CreateSession(service, profile string) (Session, error) {
	id, err := newID()
	if err != nil {
		return Session{}, err
	}
	now := time.Now().UTC()
	sess := Session{
		ID:        id,
		Service:   service,
		Profile:   profile,
		Status:    StatusRunning,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if _, err := s.db.Exec(
		`INSERT INTO backup_sessions (id, service, profile, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		sess.ID, sess.Service, sess.Profile, sess.Status, sess.CreatedAt.UnixMilli(), sess.UpdatedAt.UnixMilli(),
	); err != nil {
		return Session{}, fmt.Errorf("creating backup session: %w", err)
	}
	return sess, nil
}

// RunningForService returns the most recent running session for service/profile,
// if any. It lets callers re-attach to an in-flight backup instead of starting a
// duplicate.
func (s *Store) RunningForService(service, profile string) (*Session, bool, error) {
	row := s.db.QueryRow(
		`SELECT id, service, profile, status, snapshot, error, created_at, updated_at
		   FROM backup_sessions
		  WHERE service = ? AND profile = ? AND status = ?
		  ORDER BY created_at DESC LIMIT 1`,
		service, profile, StatusRunning,
	)
	sess, err := scanSession(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &sess, true, nil
}

// Get returns the session with id, or ErrNotFound.
func (s *Store) Get(id string) (Session, error) {
	row := s.db.QueryRow(
		`SELECT id, service, profile, status, snapshot, error, created_at, updated_at
		   FROM backup_sessions WHERE id = ?`, id)
	sess, err := scanSession(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	return sess, err
}

// AppendLog stores one log line, assigning it the next per-session sequence
// number in a single atomic statement.
func (s *Store) AppendLog(id, line string) error {
	_, err := s.db.Exec(
		`INSERT INTO backup_logs (session_id, seq, line)
		 VALUES (?, (SELECT COALESCE(MAX(seq), 0) + 1 FROM backup_logs WHERE session_id = ?), ?)`,
		id, id, line,
	)
	if err != nil {
		return fmt.Errorf("appending backup log: %w", err)
	}
	return nil
}

// Logs returns the session's log lines with seq greater than afterSeq, in order.
// Poll with the highest seq seen so far to fetch only new lines.
func (s *Store) Logs(id string, afterSeq int64) ([]LogLine, error) {
	rows, err := s.db.Query(
		`SELECT seq, line FROM backup_logs WHERE session_id = ? AND seq > ? ORDER BY seq`,
		id, afterSeq,
	)
	if err != nil {
		return nil, fmt.Errorf("reading backup logs: %w", err)
	}
	defer rows.Close()
	var out []LogLine
	for rows.Next() {
		var l LogLine
		if err := rows.Scan(&l.Seq, &l.Line); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// Finish records a terminal status for a session, plus its snapshot folder (on
// success) or error reason (on failure).
func (s *Store) Finish(id string, status Status, snapshot, errMsg string) error {
	_, err := s.db.Exec(
		`UPDATE backup_sessions SET status = ?, snapshot = ?, error = ?, updated_at = ? WHERE id = ?`,
		status, snapshot, errMsg, nowMillis(), id,
	)
	if err != nil {
		return fmt.Errorf("finishing backup session: %w", err)
	}
	return nil
}

// scanSession reads one session row, converting the stored unix-millis columns
// back into time.Time.
func scanSession(row interface{ Scan(...any) error }) (Session, error) {
	var (
		sess             Session
		created, updated int64
		status           string
	)
	if err := row.Scan(&sess.ID, &sess.Service, &sess.Profile, &status, &sess.Snapshot, &sess.Error, &created, &updated); err != nil {
		return Session{}, err
	}
	sess.Status = Status(status)
	sess.CreatedAt = time.UnixMilli(created).UTC()
	sess.UpdatedAt = time.UnixMilli(updated).UTC()
	return sess, nil
}

func nowMillis() int64 { return time.Now().UTC().UnixMilli() }

// newID returns a random 128-bit hex id. Sessions are ordered by created_at, so
// the id only needs to be unique, not sortable.
func newID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generating session id: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}
