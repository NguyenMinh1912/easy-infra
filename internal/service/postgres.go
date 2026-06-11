package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Postgres provisions a PostgreSQL database service.
//
// open is a seam for testing: when nil the lifecycle dials a real server via
// pgx (realOpener); tests set it to inject a fake connection.
type Postgres struct{ open opener }

// opener returns the connection opener to use, defaulting to a real pgx dial.
func (p Postgres) opener() opener {
	if p.open != nil {
		return p.open
	}
	return realOpener
}

// Name implements Service.
func (Postgres) Name() string { return "postgres" }

// DefaultDefinition implements Service.
func (Postgres) DefaultDefinition() Config {
	return Config{"version": "16"}
}

// ValidateDefinition implements Service.
func (Postgres) ValidateDefinition(cfg Config) error {
	_, err := optionalString(cfg, "version", "16")
	return err
}

// DefaultEnv implements Service.
func (Postgres) DefaultEnv() Config {
	return Config{
		"host":     "localhost",
		"port":     5432,
		"user":     "app",
		"password": "app",
		"database": "app",
	}
}

// ValidateEnv implements Service. A profile may describe the connection either
// as a single "url" DSN or as discrete host/port/user/database fields.
func (Postgres) ValidateEnv(cfg Config) error {
	if _, ok := cfg["url"]; ok {
		// databaseName validates the URL is a non-empty string, parses, and
		// carries a database name.
		_, err := databaseName(cfg)
		return err
	}
	if _, err := requireString(cfg, "host"); err != nil {
		return err
	}
	if _, err := optionalPort(cfg, "port", 5432); err != nil {
		return err
	}
	if _, err := requireString(cfg, "user"); err != nil {
		return err
	}
	if _, err := requireString(cfg, "database"); err != nil {
		return err
	}
	return nil
}

// Health implements Service: connect to the configured database and confirm it
// is reachable and accepting queries.
func (p Postgres) Health(ctx context.Context, spec Spec) error {
	conn, err := p.connect(ctx, spec.Env, "")
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	if err := conn.Ping(ctx); err != nil {
		return fmt.Errorf("postgres not ready: %w", err)
	}
	return nil
}

// Apply implements Service: ensure the target database exists (creating it if
// absent) and, if a backup exists for the active profile, restore the latest
// one into it. With no backup yet, Apply just leaves the freshly-created empty
// database in place.
func (p Postgres) Apply(ctx context.Context, spec Spec) error {
	if err := p.ensureDatabase(ctx, spec); err != nil {
		return err
	}
	path, err := latestBackup(spec.Profile)
	if err != nil {
		return err
	}
	if path == "" {
		return nil
	}

	conn, err := p.connect(ctx, spec.Env, "")
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening backup %s: %w", path, err)
	}
	defer f.Close()
	if err := restore(ctx, conn, f); err != nil {
		return fmt.Errorf("restoring %s: %w", path, err)
	}
	return nil
}

// Backup implements Service: write a logical SQL dump of the database to the
// active profile's backup directory.
func (p Postgres) Backup(ctx context.Context, spec Spec) error {
	conn, err := p.connect(ctx, spec.Env, "")
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	dir := backupDir(spec.Profile)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating backup dir %s: %w", dir, err)
	}
	path := filepath.Join(dir, backupStamp()+".sql")
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating backup file %s: %w", path, err)
	}
	defer f.Close()
	if err := dump(ctx, conn, f); err != nil {
		return fmt.Errorf("backing up postgres: %w", err)
	}
	return nil
}

// Clean implements Service: drop and recreate the public schema, returning the
// database to an empty state. Destructive — callers confirm before invoking.
func (p Postgres) Clean(ctx context.Context, spec Spec) error {
	conn, err := p.connect(ctx, spec.Env, "")
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"); err != nil {
		return fmt.Errorf("cleaning postgres: %w", err)
	}
	return nil
}

// connect opens a connection to the database in env, or to database when given
// (used to reach the maintenance DB).
func (p Postgres) connect(ctx context.Context, env Config, database string) (pgConn, error) {
	cs, err := connString(env, database)
	if err != nil {
		return nil, err
	}
	conn, err := p.opener()(ctx, cs)
	if err != nil {
		return nil, fmt.Errorf("connecting to postgres: %w", err)
	}
	return conn, nil
}

// ensureDatabase creates the target database if it does not already exist,
// connecting to the "postgres" maintenance database to do so.
func (p Postgres) ensureDatabase(ctx context.Context, spec Spec) error {
	dbName, err := databaseName(spec.Env)
	if err != nil {
		return err
	}
	conn, err := p.connect(ctx, spec.Env, "postgres")
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	var exists bool
	if err := conn.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", dbName).Scan(&exists); err != nil {
		return fmt.Errorf("checking database %q: %w", dbName, err)
	}
	if exists {
		return nil
	}
	// CREATE DATABASE cannot be parameterized, so the name is quoted as an
	// identifier.
	if _, err := conn.Exec(ctx, "CREATE DATABASE "+quoteIdent(dbName)); err != nil {
		return fmt.Errorf("creating database %q: %w", dbName, err)
	}
	return nil
}

// latestBackup returns the path of the most recent backup for profile, or an
// empty string when none exist. Backup filenames are sortable timestamps, so
// the lexically greatest name is the newest.
func latestBackup(profile string) (string, error) {
	dir := backupDir(profile)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("reading backups %s: %w", dir, err)
	}
	var latest string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		if e.Name() > latest {
			latest = e.Name()
		}
	}
	if latest == "" {
		return "", nil
	}
	return filepath.Join(dir, latest), nil
}

// backupStamp is a sortable, filesystem-safe UTC timestamp for backup files.
func backupStamp() string {
	return time.Now().UTC().Format("20060102T150405Z")
}
