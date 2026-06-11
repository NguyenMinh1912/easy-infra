package service

import (
	"context"
	"io"
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

func TestLatestBackup(t *testing.T) {
	dir := backupDir("default")
	t.Chdir(t.TempDir())
	if path, err := latestBackup("default"); err != nil || path != "" {
		t.Fatalf("latestBackup (none) = %q, %v; want empty", path, err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"20240101T000000Z.sql", "20250101T000000Z.sql", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	path, err := latestBackup("default")
	if err != nil {
		t.Fatalf("latestBackup: %v", err)
	}
	if filepath.Base(path) != "20250101T000000Z.sql" {
		t.Errorf("latestBackup = %q, want the newest .sql", path)
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
