// Package store is the single source of truth for easy-infra's data. Everything
// the tool manages — the known workspaces, which one is active, each workspace's
// profiles, and every profile's service instances — lives in one SQLite database
// in the user config directory.
//
// This replaces the previous file-based model (a per-folder easy-infra.yml,
// .easy-infra/state.json, and .easy-infra/profiles/*.yml plus a global
// workspaces.json). There are no project folders any more: a workspace is a row,
// not a directory. The database is tool-owned and not meant to be hand-edited.
//
// The store is deliberately low-level: it persists and returns domain types
// (profile.ServiceEntry) but holds no policy. Validation (a profile must define
// at least one valid service, the active profile cannot be removed, …) lives in
// the project facade that sits on top of it.
package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/minhnc/easy-infra/internal/profile"
	"github.com/minhnc/easy-infra/internal/service"
	"github.com/minhnc/easy-infra/internal/sqltemplate"
	_ "modernc.org/sqlite" // pure-Go driver, registered as "sqlite"
)

// configDirEnv lets the user (and tests) relocate the database away from the OS
// user config directory. It mirrors the seam the old file-based store used, so
// tests redirect persistence with a single t.Setenv.
const configDirEnv = "EASY_INFRA_CONFIG_DIR"

// dbFile is the database filename within the config directory.
const dbFile = "easy-infra.db"

// Sentinel errors so callers (the project facade, the HTTP server) can map each
// condition onto an appropriate response without string matching.
var (
	// ErrWorkspaceNotFound means no workspace has the given id (or name).
	ErrWorkspaceNotFound = errors.New("workspace not found")
	// ErrWorkspaceExists means a workspace with that name already exists.
	ErrWorkspaceExists = errors.New("workspace already exists")
	// ErrProfileNotFound means the workspace has no profile with that name.
	ErrProfileNotFound = errors.New("profile not found")
	// ErrProfileExists means the workspace already has a profile with that name.
	ErrProfileExists = errors.New("profile already exists")
	// ErrTemplateNotFound means the workspace has no SQL template with that name.
	ErrTemplateNotFound = errors.New("template not found")
	// ErrTemplateExists means the workspace already has a SQL template with that name.
	ErrTemplateExists = errors.New("template already exists")
)

// Workspace is one known workspace: a named bundle of profiles. ActiveProfile is
// the profile commands operate on by default (the empty string when none).
type Workspace struct {
	ID            int64
	Name          string
	ActiveProfile string
	CreatedAt     time.Time
}

// Store owns the SQLite database. Its zero value is not usable; construct it
// with Open.
type Store struct {
	db *sql.DB
}

// DBPath returns the on-disk location of the central database, honouring
// configDirEnv. The backup store shares this same file, so all data lives in one
// place.
func DBPath() (string, error) { return dbPath() }

// dbPath returns the on-disk location of the database, honouring configDirEnv.
func dbPath() (string, error) {
	if dir := os.Getenv(configDirEnv); dir != "" {
		return filepath.Join(dir, dbFile), nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locating user config dir: %w", err)
	}
	return filepath.Join(dir, "easy-infra", dbFile), nil
}

// Open opens (creating if needed) the database, creating its directory and
// applying the schema. WAL + a busy timeout keep the single writer from blocking
// concurrent readers; foreign keys are enabled so deleting a workspace cascades
// to its profiles and services.
func Open() (*Store, error) {
	path, err := dbPath()
	if err != nil {
		return nil, err
	}
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("creating config dir %s: %w", dir, err)
		}
	}
	dsn := "file:" + path +
		"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	// Serialize access: writes are small and infrequent, so one connection
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
CREATE TABLE IF NOT EXISTS workspaces (
	id             INTEGER PRIMARY KEY AUTOINCREMENT,
	name           TEXT NOT NULL UNIQUE,
	active_profile TEXT NOT NULL DEFAULT '',
	created_at     INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS app_state (
	id                  INTEGER PRIMARY KEY CHECK (id = 1),
	active_workspace_id INTEGER REFERENCES workspaces(id) ON DELETE SET NULL
);
CREATE TABLE IF NOT EXISTS profiles (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	workspace_id INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
	name         TEXT NOT NULL,
	UNIQUE (workspace_id, name)
);
CREATE TABLE IF NOT EXISTS services (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	profile_id INTEGER NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
	svc_id     TEXT NOT NULL,
	type       TEXT NOT NULL,
	name       TEXT NOT NULL,
	config     TEXT NOT NULL,
	UNIQUE (profile_id, svc_id)
);
CREATE TABLE IF NOT EXISTS sql_templates (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	workspace_id INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
	name         TEXT NOT NULL,
	description  TEXT NOT NULL DEFAULT '',
	sql          TEXT NOT NULL,
	created_at   INTEGER NOT NULL,
	updated_at   INTEGER NOT NULL,
	UNIQUE (workspace_id, name)
);`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("migrating database: %w", err)
	}
	return nil
}

// --- Workspaces -------------------------------------------------------------

// CreateWorkspace inserts a new workspace and returns it with its assigned id.
// It does not create any profile or change the active workspace; the caller
// orchestrates scaffolding and activation. A duplicate name yields
// ErrWorkspaceExists.
func (s *Store) CreateWorkspace(name string) (Workspace, error) {
	if strings.TrimSpace(name) == "" {
		return Workspace{}, fmt.Errorf("workspace name is required")
	}
	now := time.Now().UTC()
	res, err := s.db.Exec(
		`INSERT INTO workspaces (name, active_profile, created_at) VALUES (?, '', ?)`,
		name, now.UnixMilli(),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return Workspace{}, fmt.Errorf("%q: %w", name, ErrWorkspaceExists)
		}
		return Workspace{}, fmt.Errorf("creating workspace: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Workspace{}, fmt.Errorf("creating workspace: %w", err)
	}
	return Workspace{ID: id, Name: name, CreatedAt: now}, nil
}

// RenameWorkspace changes a workspace's name. It yields ErrWorkspaceNotFound for
// an unknown id and ErrWorkspaceExists if the new name is taken.
func (s *Store) RenameWorkspace(id int64, newName string) error {
	if strings.TrimSpace(newName) == "" {
		return fmt.Errorf("workspace name is required")
	}
	res, err := s.db.Exec(`UPDATE workspaces SET name = ? WHERE id = ?`, newName, id)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("%q: %w", newName, ErrWorkspaceExists)
		}
		return fmt.Errorf("renaming workspace: %w", err)
	}
	return mustAffectOne(res, ErrWorkspaceNotFound)
}

// RemoveWorkspace deletes a workspace and (via cascade) its profiles and
// services. If it was the active workspace the active pointer is cleared. An
// unknown id yields ErrWorkspaceNotFound.
func (s *Store) RemoveWorkspace(id int64) error {
	res, err := s.db.Exec(`DELETE FROM workspaces WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("removing workspace: %w", err)
	}
	return mustAffectOne(res, ErrWorkspaceNotFound)
}

// ListWorkspaces returns all workspaces, oldest first.
func (s *Store) ListWorkspaces() ([]Workspace, error) {
	rows, err := s.db.Query(
		`SELECT id, name, active_profile, created_at FROM workspaces ORDER BY created_at, id`)
	if err != nil {
		return nil, fmt.Errorf("listing workspaces: %w", err)
	}
	defer rows.Close()
	var out []Workspace
	for rows.Next() {
		w, err := scanWorkspace(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// GetWorkspace returns the workspace with id, or ErrWorkspaceNotFound.
func (s *Store) GetWorkspace(id int64) (Workspace, error) {
	row := s.db.QueryRow(
		`SELECT id, name, active_profile, created_at FROM workspaces WHERE id = ?`, id)
	w, err := scanWorkspace(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Workspace{}, ErrWorkspaceNotFound
	}
	return w, err
}

// SetActiveWorkspace marks the workspace active. An unknown id yields
// ErrWorkspaceNotFound.
func (s *Store) SetActiveWorkspace(id int64) error {
	if _, err := s.GetWorkspace(id); err != nil {
		return err
	}
	_, err := s.db.Exec(
		`INSERT INTO app_state (id, active_workspace_id) VALUES (1, ?)
		 ON CONFLICT(id) DO UPDATE SET active_workspace_id = excluded.active_workspace_id`,
		id,
	)
	if err != nil {
		return fmt.Errorf("setting active workspace: %w", err)
	}
	return nil
}

// ActiveWorkspace returns the active workspace. ok is false when none is set
// (no workspaces, or the active one was removed).
func (s *Store) ActiveWorkspace() (Workspace, bool, error) {
	row := s.db.QueryRow(
		`SELECT w.id, w.name, w.active_profile, w.created_at
		   FROM app_state a JOIN workspaces w ON w.id = a.active_workspace_id
		  WHERE a.id = 1`)
	w, err := scanWorkspace(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Workspace{}, false, nil
	}
	if err != nil {
		return Workspace{}, false, err
	}
	return w, true, nil
}

// SetWorkspaceActiveProfile records name as the workspace's active profile. It
// does not check that the profile exists — that is the facade's job. An unknown
// workspace id yields ErrWorkspaceNotFound.
func (s *Store) SetWorkspaceActiveProfile(id int64, name string) error {
	res, err := s.db.Exec(`UPDATE workspaces SET active_profile = ? WHERE id = ?`, name, id)
	if err != nil {
		return fmt.Errorf("setting active profile: %w", err)
	}
	return mustAffectOne(res, ErrWorkspaceNotFound)
}

// --- Profiles ---------------------------------------------------------------

// ListProfiles returns the names of the workspace's profiles, sorted.
func (s *Store) ListProfiles(wsID int64) ([]string, error) {
	rows, err := s.db.Query(`SELECT name FROM profiles WHERE workspace_id = ? ORDER BY name`, wsID)
	if err != nil {
		return nil, fmt.Errorf("listing profiles: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

// ProfileExists reports whether the workspace has a profile with that name.
func (s *Store) ProfileExists(wsID int64, name string) (bool, error) {
	_, err := s.profileID(wsID, name)
	if errors.Is(err, ErrProfileNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// CreateProfile creates a profile owning the given service instances. A
// duplicate name yields ErrProfileExists; an unknown workspace yields
// ErrWorkspaceNotFound.
func (s *Store) CreateProfile(wsID int64, name string, services map[string]profile.ServiceEntry) error {
	if _, err := s.GetWorkspace(wsID); err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("creating profile: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.Exec(`INSERT INTO profiles (workspace_id, name) VALUES (?, ?)`, wsID, name)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("%q: %w", name, ErrProfileExists)
		}
		return fmt.Errorf("creating profile: %w", err)
	}
	profID, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("creating profile: %w", err)
	}
	if err := insertServices(tx, profID, services); err != nil {
		return err
	}
	return tx.Commit()
}

// RemoveProfile deletes a profile and (via cascade) its services. An unknown
// profile yields ErrProfileNotFound.
func (s *Store) RemoveProfile(wsID int64, name string) error {
	res, err := s.db.Exec(`DELETE FROM profiles WHERE workspace_id = ? AND name = ?`, wsID, name)
	if err != nil {
		return fmt.Errorf("removing profile: %w", err)
	}
	return mustAffectOne(res, ErrProfileNotFound)
}

// ProfileServices returns the profile's service instances keyed by id. An
// unknown profile yields ErrProfileNotFound.
func (s *Store) ProfileServices(wsID int64, name string) (map[string]profile.ServiceEntry, error) {
	profID, err := s.profileID(wsID, name)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(
		`SELECT svc_id, type, name, config FROM services WHERE profile_id = ?`, profID)
	if err != nil {
		return nil, fmt.Errorf("reading services: %w", err)
	}
	defer rows.Close()
	out := map[string]profile.ServiceEntry{}
	for rows.Next() {
		var id, svcType, svcName, cfgJSON string
		if err := rows.Scan(&id, &svcType, &svcName, &cfgJSON); err != nil {
			return nil, err
		}
		cfg := service.Config{}
		if cfgJSON != "" {
			if err := json.Unmarshal([]byte(cfgJSON), &cfg); err != nil {
				return nil, fmt.Errorf("decoding config for service %q: %w", id, err)
			}
		}
		out[id] = profile.ServiceEntry{Type: svcType, Name: svcName, Config: cfg}
	}
	return out, rows.Err()
}

// ReplaceProfileServices swaps the profile's service instances for the given
// set. An unknown profile yields ErrProfileNotFound.
func (s *Store) ReplaceProfileServices(wsID int64, name string, services map[string]profile.ServiceEntry) error {
	profID, err := s.profileID(wsID, name)
	if err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("replacing services: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM services WHERE profile_id = ?`, profID); err != nil {
		return fmt.Errorf("replacing services: %w", err)
	}
	if err := insertServices(tx, profID, services); err != nil {
		return err
	}
	return tx.Commit()
}

// profileID resolves a profile's row id, or ErrProfileNotFound.
func (s *Store) profileID(wsID int64, name string) (int64, error) {
	var id int64
	err := s.db.QueryRow(
		`SELECT id FROM profiles WHERE workspace_id = ? AND name = ?`, wsID, name).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("%q: %w", name, ErrProfileNotFound)
	}
	if err != nil {
		return 0, fmt.Errorf("looking up profile: %w", err)
	}
	return id, nil
}

// --- SQL templates ----------------------------------------------------------

// ListTemplates returns the workspace's SQL templates, sorted by name.
func (s *Store) ListTemplates(wsID int64) ([]sqltemplate.Template, error) {
	rows, err := s.db.Query(
		`SELECT name, description, sql, created_at, updated_at
		   FROM sql_templates WHERE workspace_id = ? ORDER BY name`, wsID)
	if err != nil {
		return nil, fmt.Errorf("listing templates: %w", err)
	}
	defer rows.Close()
	var out []sqltemplate.Template
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// GetTemplate returns the named template in the workspace, or
// ErrTemplateNotFound.
func (s *Store) GetTemplate(wsID int64, name string) (sqltemplate.Template, error) {
	row := s.db.QueryRow(
		`SELECT name, description, sql, created_at, updated_at
		   FROM sql_templates WHERE workspace_id = ? AND name = ?`, wsID, name)
	t, err := scanTemplate(row)
	if errors.Is(err, sql.ErrNoRows) {
		return sqltemplate.Template{}, fmt.Errorf("%q: %w", name, ErrTemplateNotFound)
	}
	return t, err
}

// CreateTemplate inserts a new template. A duplicate name yields
// ErrTemplateExists; an unknown workspace yields ErrWorkspaceNotFound.
func (s *Store) CreateTemplate(wsID int64, t sqltemplate.Template) error {
	if _, err := s.GetWorkspace(wsID); err != nil {
		return err
	}
	now := time.Now().UTC().UnixMilli()
	_, err := s.db.Exec(
		`INSERT INTO sql_templates (workspace_id, name, description, sql, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		wsID, t.Name, t.Description, t.SQL, now, now,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("%q: %w", t.Name, ErrTemplateExists)
		}
		return fmt.Errorf("creating template: %w", err)
	}
	return nil
}

// UpdateTemplate replaces the description and SQL of the named template,
// refreshing its updated_at. An unknown template yields ErrTemplateNotFound.
func (s *Store) UpdateTemplate(wsID int64, name string, t sqltemplate.Template) error {
	now := time.Now().UTC().UnixMilli()
	res, err := s.db.Exec(
		`UPDATE sql_templates SET description = ?, sql = ?, updated_at = ?
		  WHERE workspace_id = ? AND name = ?`,
		t.Description, t.SQL, now, wsID, name,
	)
	if err != nil {
		return fmt.Errorf("updating template: %w", err)
	}
	return mustAffectOne(res, ErrTemplateNotFound)
}

// RemoveTemplate deletes the named template. An unknown template yields
// ErrTemplateNotFound.
func (s *Store) RemoveTemplate(wsID int64, name string) error {
	res, err := s.db.Exec(
		`DELETE FROM sql_templates WHERE workspace_id = ? AND name = ?`, wsID, name)
	if err != nil {
		return fmt.Errorf("removing template: %w", err)
	}
	return mustAffectOne(res, ErrTemplateNotFound)
}

// scanTemplate reads one template row, converting the stored millisecond
// timestamps back to time.Time.
func scanTemplate(row interface{ Scan(...any) error }) (sqltemplate.Template, error) {
	var (
		t                sqltemplate.Template
		created, updated int64
	)
	if err := row.Scan(&t.Name, &t.Description, &t.SQL, &created, &updated); err != nil {
		return sqltemplate.Template{}, err
	}
	t.CreatedAt = time.UnixMilli(created).UTC()
	t.UpdatedAt = time.UnixMilli(updated).UTC()
	return t, nil
}

// --- helpers ----------------------------------------------------------------

// insertServices writes a profile's service instances within tx. Config is
// stored as JSON; a nil config serialises as an empty object.
func insertServices(tx *sql.Tx, profID int64, services map[string]profile.ServiceEntry) error {
	// Insert in id order so a profile's rows are written deterministically.
	ids := make([]string, 0, len(services))
	for id := range services {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		entry := services[id]
		cfg := entry.Config
		if cfg == nil {
			cfg = service.Config{}
		}
		data, err := json.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("encoding config for service %q: %w", id, err)
		}
		if _, err := tx.Exec(
			`INSERT INTO services (profile_id, svc_id, type, name, config) VALUES (?, ?, ?, ?, ?)`,
			profID, id, entry.Type, entry.Name, string(data),
		); err != nil {
			return fmt.Errorf("writing service %q: %w", id, err)
		}
	}
	return nil
}

func scanWorkspace(row interface{ Scan(...any) error }) (Workspace, error) {
	var (
		w       Workspace
		created int64
	)
	if err := row.Scan(&w.ID, &w.Name, &w.ActiveProfile, &created); err != nil {
		return Workspace{}, err
	}
	w.CreatedAt = time.UnixMilli(created).UTC()
	return w, nil
}

// mustAffectOne maps a zero-rows-affected result onto notFound, so an UPDATE or
// DELETE against a missing row reports the right sentinel.
func mustAffectOne(res sql.Result, notFound error) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return notFound
	}
	return nil
}

// isUniqueViolation reports whether err is a SQLite UNIQUE constraint failure,
// so a duplicate name maps to a friendly sentinel rather than a raw driver error.
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}
