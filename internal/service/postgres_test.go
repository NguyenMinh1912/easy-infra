package service

import (
	"context"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// fakeConn is an in-memory pgConn for unit tests: it records Exec SQL and
// CopyFrom payloads and serves canned QueryRow/Ping results.
type fakeConn struct {
	pingErr   error
	execErr   error
	rowExists bool
	execs     []string
	copyFroms []copyCall
}

type copyCall struct {
	sql  string
	data string
}

func (c *fakeConn) Ping(context.Context) error { return c.pingErr }

func (c *fakeConn) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	c.execs = append(c.execs, sql)
	return pgconn.CommandTag{}, c.execErr
}

func (c *fakeConn) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, io.EOF // not exercised by these tests
}

func (c *fakeConn) QueryRow(context.Context, string, ...any) pgx.Row {
	return fakeRow{exists: c.rowExists}
}

func (c *fakeConn) CopyTo(context.Context, io.Writer, string) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, io.EOF // not exercised by these tests
}

func (c *fakeConn) CopyFrom(_ context.Context, r io.Reader, sql string) (pgconn.CommandTag, error) {
	b, _ := io.ReadAll(r)
	c.copyFroms = append(c.copyFroms, copyCall{sql: sql, data: string(b)})
	return pgconn.CommandTag{}, nil
}

func (c *fakeConn) Close(context.Context) error { return nil }

// fakeRow scans a single bool (the SELECT EXISTS result).
type fakeRow struct{ exists bool }

func (r fakeRow) Scan(dest ...any) error {
	if len(dest) == 1 {
		if p, ok := dest[0].(*bool); ok {
			*p = r.exists
		}
	}
	return nil
}

// withConn returns a Postgres whose opener always yields fc.
func withConn(fc *fakeConn) Postgres {
	return Postgres{open: func(context.Context, string) (pgConn, error) { return fc, nil }}
}

func TestConnString(t *testing.T) {
	env := Postgres{}.DefaultEnv()
	cs, err := connString(env, "")
	if err != nil {
		t.Fatalf("connString: %v", err)
	}
	for _, want := range []string{"host='localhost'", "port=5432", "user='app'", "dbname='app'", "password='app'"} {
		if !strings.Contains(cs, want) {
			t.Errorf("connString = %q, missing %q", cs, want)
		}
	}
	// database override targets the maintenance DB.
	cs, err = connString(env, "postgres")
	if err != nil {
		t.Fatalf("connString override: %v", err)
	}
	if !strings.Contains(cs, "dbname='postgres'") {
		t.Errorf("override connString = %q, want dbname='postgres'", cs)
	}
}

func TestValidateEnvStringPort(t *testing.T) {
	// The web UI submits every field as a string, so a numeric string port
	// must validate; a non-numeric one must still be rejected.
	if err := (Postgres{}).ValidateEnv(Config{"host": "h", "port": "5432", "user": "u", "database": "d"}); err != nil {
		t.Errorf("string port should validate: %v", err)
	}
	if err := (Postgres{}).ValidateEnv(Config{"host": "h", "port": "abc", "user": "u", "database": "d"}); err == nil {
		t.Errorf("non-numeric port should be rejected")
	}
}

func TestConnStringSchema(t *testing.T) {
	env := Postgres{}.DefaultEnv()
	env["schema"] = "app_dynamic"
	cs, err := connString(env, "")
	if err != nil {
		t.Fatalf("connString: %v", err)
	}
	// schema is applied as a search_path runtime parameter, mirroring the URL
	// form's currentSchema handling.
	if !strings.Contains(cs, "search_path='app_dynamic'") {
		t.Errorf("connString = %q, missing search_path='app_dynamic'", cs)
	}
}

func TestConnStringURL(t *testing.T) {
	env := Config{"url": "postgres://app:secret@db.internal:5432/app?sslmode=require"}
	cs, err := connString(env, "")
	if err != nil {
		t.Fatalf("connString: %v", err)
	}
	if cs != "postgres://app:secret@db.internal:5432/app?sslmode=require" {
		t.Errorf("connString = %q, want the URL verbatim", cs)
	}
	// database override swaps only the path, preserving credentials and params.
	cs, err = connString(env, "postgres")
	if err != nil {
		t.Fatalf("connString override: %v", err)
	}
	if cs != "postgres://app:secret@db.internal:5432/postgres?sslmode=require" {
		t.Errorf("override connString = %q, want dbname swapped to postgres", cs)
	}
}

func TestConnStringJDBCURL(t *testing.T) {
	// JDBC URL with a currentSchema: the jdbc: prefix is stripped and
	// currentSchema is translated to search_path so pgx accepts it.
	env := Config{"url": "jdbc:postgresql://kot:secret@172.16.10.228:5432/kot?currentSchema=kot_dynamic"}
	cs, err := connString(env, "")
	if err != nil {
		t.Fatalf("connString: %v", err)
	}
	u, err := url.Parse(cs)
	if err != nil {
		t.Fatalf("result is not a valid URL %q: %v", cs, err)
	}
	if u.Scheme != "postgresql" {
		t.Errorf("scheme = %q, want postgresql (jdbc: stripped)", u.Scheme)
	}
	if got := u.Query().Get("search_path"); got != "kot_dynamic" {
		t.Errorf("search_path = %q, want kot_dynamic", got)
	}
	if u.Query().Has("currentSchema") {
		t.Errorf("currentSchema should be removed, got %q", cs)
	}
	// database override keeps the schema query parameter intact.
	cs, err = connString(env, "postgres")
	if err != nil {
		t.Fatalf("connString override: %v", err)
	}
	u, _ = url.Parse(cs)
	if u.Path != "/postgres" {
		t.Errorf("path = %q, want /postgres", u.Path)
	}
	if got := u.Query().Get("search_path"); got != "kot_dynamic" {
		t.Errorf("override search_path = %q, want kot_dynamic", got)
	}
	// databaseName parses the db out of a JDBC URL.
	if name, err := databaseName(env); err != nil || name != "kot" {
		t.Errorf("databaseName = %q, %v; want kot", name, err)
	}
}

func TestValidateEnvURL(t *testing.T) {
	if err := (Postgres{}).ValidateEnv(Config{"url": "postgres://app@db.internal/app"}); err != nil {
		t.Errorf("ValidateEnv (valid url): %v", err)
	}
	if err := (Postgres{}).ValidateEnv(Config{"url": "postgres://app@db.internal/"}); err == nil {
		t.Error("ValidateEnv (url without database): expected error")
	}
	if err := (Postgres{}).ValidateEnv(Config{"url": ""}); err == nil {
		t.Error("ValidateEnv (empty url): expected error")
	}
}

func TestPostgresApplyURLCreatesDatabase(t *testing.T) {
	t.Chdir(t.TempDir())
	fc := &fakeConn{rowExists: false}
	env := Config{"url": "postgres://app:secret@db.internal:5432/app"}
	if err := withConn(fc).Apply(context.Background(), Spec{Profile: "default", Env: env}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	var created bool
	for _, e := range fc.execs {
		if strings.Contains(e, `CREATE DATABASE "app"`) {
			created = true
		}
	}
	if !created {
		t.Errorf("Apply execs = %v, want CREATE DATABASE \"app\" (db name parsed from url)", fc.execs)
	}
}

func TestPostgresHealth(t *testing.T) {
	if err := withConn(&fakeConn{}).Health(context.Background(), Spec{Env: Postgres{}.DefaultEnv()}); err != nil {
		t.Errorf("Health (healthy): %v", err)
	}
	if err := withConn(&fakeConn{pingErr: io.EOF}).Health(context.Background(), Spec{Env: Postgres{}.DefaultEnv()}); err == nil {
		t.Error("Health (ping fails): expected error")
	}
	// opener failure should surface as a connection error.
	p := Postgres{open: func(context.Context, string) (pgConn, error) { return nil, io.EOF }}
	if err := p.Health(context.Background(), Spec{Env: Postgres{}.DefaultEnv()}); err == nil {
		t.Error("Health (connect fails): expected error")
	}
}

func TestPostgresClean(t *testing.T) {
	fc := &fakeConn{}
	if err := withConn(fc).Clean(context.Background(), Spec{Env: Postgres{}.DefaultEnv()}); err != nil {
		t.Fatalf("Clean: %v", err)
	}
	if len(fc.execs) != 1 || !strings.Contains(fc.execs[0], "DROP SCHEMA public CASCADE") {
		t.Errorf("Clean execs = %v, want a DROP/CREATE SCHEMA statement", fc.execs)
	}
}

func TestPostgresApplyCreatesDatabaseNoBackup(t *testing.T) {
	t.Chdir(t.TempDir()) // isolate: no backups present
	fc := &fakeConn{rowExists: false}
	if err := withConn(fc).Apply(context.Background(), Spec{Profile: "default", Env: Postgres{}.DefaultEnv()}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	var created bool
	for _, e := range fc.execs {
		if strings.Contains(e, `CREATE DATABASE "app"`) {
			created = true
		}
	}
	if !created {
		t.Errorf("Apply execs = %v, want CREATE DATABASE \"app\"", fc.execs)
	}
}

func TestPostgresApplySkipsCreateWhenExists(t *testing.T) {
	t.Chdir(t.TempDir())
	fc := &fakeConn{rowExists: true}
	if err := withConn(fc).Apply(context.Background(), Spec{Profile: "default", Env: Postgres{}.DefaultEnv()}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	for _, e := range fc.execs {
		if strings.Contains(e, "CREATE DATABASE") {
			t.Errorf("Apply created a database that already exists: %v", fc.execs)
		}
	}
}

func TestLatestSnapshotDir(t *testing.T) {
	t.Chdir(t.TempDir())
	if dir, err := latestSnapshotDir("default"); err != nil || dir != "" {
		t.Fatalf("latestSnapshotDir (none) = %q, %v; want empty", dir, err)
	}
	for _, id := range []string{"20240101T000000Z", "20250101T000000Z"} {
		if err := os.MkdirAll(filepath.Join(BackupsDir("default"), id), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// A stray file in the backups dir must be ignored — only snapshot folders count.
	if err := os.WriteFile(filepath.Join(BackupsDir("default"), "notes.txt"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	ids, err := ListSnapshots("default")
	if err != nil {
		t.Fatalf("ListSnapshots: %v", err)
	}
	if len(ids) != 2 || ids[0] != "20240101T000000Z" || ids[1] != "20250101T000000Z" {
		t.Errorf("ListSnapshots = %v, want the two snapshot ids sorted", ids)
	}

	dir, err := latestSnapshotDir("default")
	if err != nil {
		t.Fatalf("latestSnapshotDir: %v", err)
	}
	if filepath.Base(dir) != "20250101T000000Z" {
		t.Errorf("latestSnapshotDir = %q, want the newest snapshot", dir)
	}
}

// schemaRow scans a single *string (the current_schema() result, which may be
// NULL when search_path resolves to no existing schema).
type schemaRow struct{ schema *string }

func (r schemaRow) Scan(dest ...any) error {
	if len(dest) == 1 {
		if p, ok := dest[0].(**string); ok {
			*p = r.schema
		}
	}
	return nil
}

type schemaConn struct {
	fakeConn
	schema *string
}

func (c *schemaConn) QueryRow(context.Context, string, ...any) pgx.Row {
	return schemaRow{schema: c.schema}
}

func TestCurrentSchema(t *testing.T) {
	dyn := "kot_asean_dynamic"
	empty := ""
	cases := []struct {
		name   string
		schema *string
		want   string
	}{
		{"configured schema", &dyn, "kot_asean_dynamic"},
		{"null falls back to public", nil, "public"},
		{"empty falls back to public", &empty, "public"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := currentSchema(context.Background(), &schemaConn{schema: tc.schema})
			if err != nil {
				t.Fatalf("currentSchema: %v", err)
			}
			if got != tc.want {
				t.Errorf("currentSchema = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRestoreReplay(t *testing.T) {
	dumpText := `-- easy-infra postgres backup
SET client_encoding = 'UTF8';

CREATE TABLE public."t" (
    "id" integer NOT NULL
);

COPY public."t" FROM stdin;
1
2
\.

ALTER TABLE public."t" ADD CONSTRAINT "t_pkey" PRIMARY KEY ("id");
`
	fc := &fakeConn{}
	if err := restore(context.Background(), fc, strings.NewReader(dumpText)); err != nil {
		t.Fatalf("restore: %v", err)
	}

	if len(fc.execs) < 2 || fc.execs[0] != "BEGIN" || fc.execs[len(fc.execs)-1] != "COMMIT" {
		t.Fatalf("restore execs not wrapped in BEGIN/COMMIT: %v", fc.execs)
	}
	joined := strings.Join(fc.execs, "\n")
	if !strings.Contains(joined, "CREATE TABLE public.\"t\"") {
		t.Errorf("missing CREATE TABLE in execs: %v", fc.execs)
	}
	if !strings.Contains(joined, "ADD CONSTRAINT \"t_pkey\"") {
		t.Errorf("missing ALTER TABLE in execs: %v", fc.execs)
	}
	if len(fc.copyFroms) != 1 {
		t.Fatalf("copyFroms = %d, want 1", len(fc.copyFroms))
	}
	if fc.copyFroms[0].sql != `COPY public."t" FROM stdin` {
		t.Errorf("copy sql = %q", fc.copyFroms[0].sql)
	}
	if fc.copyFroms[0].data != "1\n2\n" {
		t.Errorf("copy data = %q, want \"1\\n2\\n\"", fc.copyFroms[0].data)
	}
}
