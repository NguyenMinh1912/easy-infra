package service

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// pgConn is the subset of a pgx connection the postgres lifecycle relies on.
// Depending on an interface (rather than *pgx.Conn directly) lets tests inject
// a fake connection and assert the exact SQL and COPY traffic without a live
// server.
type pgConn interface {
	Ping(ctx context.Context) error
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	CopyTo(ctx context.Context, w io.Writer, sql string) (pgconn.CommandTag, error)
	CopyFrom(ctx context.Context, r io.Reader, sql string) (pgconn.CommandTag, error)
	Close(ctx context.Context) error
}

// opener establishes a connection to the database named by connString. It is a
// seam: the zero-value Postgres uses realOpener, while tests supply a fake.
type opener func(ctx context.Context, connString string) (pgConn, error)

// realOpener dials a real PostgreSQL server with pgx.
func realOpener(ctx context.Context, connString string) (pgConn, error) {
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return nil, err
	}
	return &pgxConn{conn}, nil
}

// pgxConn adapts *pgx.Conn to pgConn, routing the COPY operations through the
// lower-level PgConn which exposes the streaming COPY protocol.
type pgxConn struct{ c *pgx.Conn }

func (p *pgxConn) Ping(ctx context.Context) error { return p.c.Ping(ctx) }

func (p *pgxConn) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return p.c.Exec(ctx, sql, args...)
}

func (p *pgxConn) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return p.c.Query(ctx, sql, args...)
}

func (p *pgxConn) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return p.c.QueryRow(ctx, sql, args...)
}

func (p *pgxConn) CopyTo(ctx context.Context, w io.Writer, sql string) (pgconn.CommandTag, error) {
	return p.c.PgConn().CopyTo(ctx, w, sql)
}

func (p *pgxConn) CopyFrom(ctx context.Context, r io.Reader, sql string) (pgconn.CommandTag, error) {
	return p.c.PgConn().CopyFrom(ctx, r, sql)
}

func (p *pgxConn) Close(ctx context.Context) error { return p.c.Close(ctx) }

// connString builds a libpq connection string from a profile's environment
// config. When database is non-empty it overrides the configured dbname — used
// to reach the "postgres" maintenance database when creating the target
// database.
//
// A profile may instead supply the whole DSN as a single "url" field
// (e.g. postgres://user:pass@host:port/db?sslmode=require), in which case the
// URL is used, with only the database overridden when requested. JDBC-style
// URLs (jdbc:postgresql://...) and their schema query parameter are normalized
// by postgresURL.
func connString(env Config, database string) (string, error) {
	if raw, ok := env["url"]; ok {
		s, err := urlString(raw)
		if err != nil {
			return "", err
		}
		u, err := postgresURL(s)
		if err != nil {
			return "", err
		}
		if database != "" {
			u.Path = "/" + database
		}
		return u.String(), nil
	}

	host, err := requireString(env, "host")
	if err != nil {
		return "", err
	}
	port, err := optionalPort(env, "port", 5432)
	if err != nil {
		return "", err
	}
	user, err := requireString(env, "user")
	if err != nil {
		return "", err
	}
	password, err := optionalString(env, "password", "")
	if err != nil {
		return "", err
	}
	if database == "" {
		if database, err = requireString(env, "database"); err != nil {
			return "", err
		}
	}

	parts := []string{
		"host=" + quoteDSN(host),
		fmt.Sprintf("port=%d", port),
		"user=" + quoteDSN(user),
		"dbname=" + quoteDSN(database),
	}
	if password != "" {
		parts = append(parts, "password="+quoteDSN(password))
	}
	return strings.Join(parts, " "), nil
}

// urlString returns the validated, non-empty "url" DSN value.
func urlString(raw any) (string, error) {
	s, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("%q must be a string, got %v", "url", raw)
	}
	if s == "" {
		return "", fmt.Errorf("%q must not be empty", "url")
	}
	return s, nil
}

// databaseName returns the target database for env: parsed from the "url" DSN
// when one is given, and otherwise read from the "database" field.
func databaseName(env Config) (string, error) {
	raw, ok := env["url"]
	if !ok {
		return requireString(env, "database")
	}
	s, err := urlString(raw)
	if err != nil {
		return "", err
	}
	u, err := postgresURL(s)
	if err != nil {
		return "", err
	}
	name := strings.TrimPrefix(u.Path, "/")
	if name == "" {
		return "", fmt.Errorf("%q must include a database name", "url")
	}
	return name, nil
}

// postgresURL parses a profile "url" DSN into a pgx-acceptable URL. It accepts
// JDBC-style URLs by stripping the leading "jdbc:" prefix, and translates
// JDBC's "currentSchema" query parameter into PostgreSQL's "search_path"
// runtime parameter so the configured schema is applied on connect (pgx rejects
// the unknown "currentSchema" parameter otherwise).
func postgresURL(s string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimPrefix(s, "jdbc:"))
	if err != nil {
		return nil, fmt.Errorf("%q is not a valid connection URL: %w", "url", err)
	}
	q := u.Query()
	if schema := q.Get("currentSchema"); schema != "" {
		q.Del("currentSchema")
		if q.Get("search_path") == "" {
			q.Set("search_path", schema)
		}
		u.RawQuery = q.Encode()
	}
	return u, nil
}

// quoteDSN wraps a connection-string value in single quotes, escaping the
// characters libpq treats specially, so credentials with spaces or quotes are
// passed safely.
func quoteDSN(v string) string {
	return "'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(v) + "'"
}

// quoteIdent renders s as a quoted SQL identifier, doubling embedded quotes.
func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// backupDir is the directory holding a profile's postgres backups, relative to
// the project root (matching how config/state paths are resolved).
func backupDir(profile string) string {
	return filepath.Join(".easy-infra", "backups", profile, "postgres")
}
