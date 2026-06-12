package service

import (
	"context"
	"os"
	"strings"
	"testing"
)

// TestPostgresIntegration exercises the full backup → clean → apply round-trip
// against a real PostgreSQL server. It is skipped unless EASY_INFRA_PG_IT=1.
// Connection comes from the standard PG* env vars (defaults match DefaultEnv).
//
// The user/role must be able to CREATE DATABASE and DROP/CREATE SCHEMA.
func TestPostgresIntegration(t *testing.T) {
	if os.Getenv("EASY_INFRA_PG_IT") == "" {
		t.Skip("set EASY_INFRA_PG_IT=1 (and PG* env) to run the real-DB integration test")
	}
	t.Chdir(t.TempDir()) // backups land under a throwaway cwd

	env := Config{
		"host":     getenv("PGHOST", "localhost"),
		"port":     atoiOr(getenv("PGPORT", "5432"), 5432),
		"user":     getenv("PGUSER", "app"),
		"password": getenv("PGPASSWORD", "app"),
		"database": getenv("PGDATABASE", "app"),
	}
	spec := Spec{Profile: "it", Env: env}
	p := Postgres{}
	ctx := context.Background()

	// Ensure the database exists and start from a clean schema.
	if err := p.Apply(ctx, spec); err != nil {
		t.Fatalf("Apply (ensure db): %v", err)
	}
	if err := p.Health(ctx, spec); err != nil {
		t.Fatalf("Health: %v", err)
	}
	if err := p.Clean(ctx, spec); err != nil {
		t.Fatalf("Clean (pre): %v", err)
	}

	// Seed a table with data via a real connection.
	cs, err := connString(env, "")
	if err != nil {
		t.Fatal(err)
	}
	conn, err := realOpener(ctx, cs)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if _, err := conn.Exec(ctx, `CREATE TABLE widget (id serial PRIMARY KEY, name text NOT NULL)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := conn.Exec(ctx, `INSERT INTO widget (name) VALUES ('a'), ('b'), ('c')`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	// A foreign key from an alphabetically-earlier table ("company") to a later
	// one ("region") guards constraint ordering: the FK must be restored after
	// region's primary key exists, even though company is dumped first.
	if _, err := conn.Exec(ctx, `CREATE TABLE region (id serial PRIMARY KEY, name text NOT NULL)`); err != nil {
		t.Fatalf("create region: %v", err)
	}
	if _, err := conn.Exec(ctx, `CREATE TABLE company (id serial PRIMARY KEY, region_id integer NOT NULL REFERENCES region (id))`); err != nil {
		t.Fatalf("create company: %v", err)
	}
	if _, err := conn.Exec(ctx, `INSERT INTO region (name) VALUES ('emea')`); err != nil {
		t.Fatalf("insert region: %v", err)
	}
	if _, err := conn.Exec(ctx, `INSERT INTO company (region_id) VALUES (1)`); err != nil {
		t.Fatalf("insert company: %v", err)
	}
	_ = conn.Close(ctx)

	// Back up (capturing verbose logs), wipe, then restore via Apply.
	var logs strings.Builder
	spec.Log = &logs
	if err := p.Backup(ctx, spec); err != nil {
		t.Fatalf("Backup: %v", err)
	}
	if !strings.Contains(logs.String(), `dumping table "widget"`) {
		t.Errorf("verbose backup logs missing table line:\n%s", logs.String())
	}
	spec.Log = nil
	if err := p.Clean(ctx, spec); err != nil {
		t.Fatalf("Clean: %v", err)
	}
	if err := p.Apply(ctx, spec); err != nil {
		t.Fatalf("Apply (restore): %v", err)
	}

	// Verify the data came back.
	conn, err = realOpener(ctx, cs)
	if err != nil {
		t.Fatalf("connect (verify): %v", err)
	}
	defer conn.Close(ctx)
	var count int
	if err := conn.QueryRow(ctx, `SELECT count(*) FROM widget`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 3 {
		t.Errorf("restored row count = %d, want 3", count)
	}
	// The foreign key must have survived the round-trip and still enforce
	// referential integrity.
	if _, err := conn.Exec(ctx, `INSERT INTO company (region_id) VALUES (999)`); err == nil {
		t.Error("expected foreign key violation inserting company with unknown region_id, got nil")
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func atoiOr(s string, def int) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return def
		}
		n = n*10 + int(r-'0')
	}
	if s == "" {
		return def
	}
	return n
}
