package service

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestPostgresProvisionIntegration exercises real container provisioning: it
// launches a local postgres via the docker CLI, waits for it to come up, then
// runs a backup → clean → restore round-trip against it — the same path "fork to
// local" drives. It is skipped unless EASY_INFRA_DOCKER_IT=1 and requires a
// working docker daemon with a pullable postgres image.
func TestPostgresProvisionIntegration(t *testing.T) {
	if os.Getenv("EASY_INFRA_DOCKER_IT") == "" {
		t.Skip("set EASY_INFRA_DOCKER_IT=1 to run the real-docker provisioning test")
	}
	t.Chdir(t.TempDir()) // backups land under a throwaway cwd

	const profile = "fork-it"
	// A high port unlikely to collide with a local server.
	source := Config{
		"host": "example.invalid", "port": 55433,
		"user": "app", "password": "app", "database": "app",
	}
	p := Postgres{}
	localEnv, err := p.LocalEnv(source)
	if err != nil {
		t.Fatalf("LocalEnv: %v", err)
	}

	spec := Spec{Profile: profile, Definition: Config{"version": "16"}, Env: localEnv}
	name := localContainerName(profile, "postgres")
	t.Cleanup(func() {
		_ = exec.Command("docker", "rm", "-f", name).Run()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	if err := p.Provision(ctx, spec); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	// Provisioning is idempotent: a second call reuses the running container.
	if err := p.Provision(ctx, spec); err != nil {
		t.Fatalf("Provision (idempotent): %v", err)
	}
	if err := p.Health(ctx, spec); err != nil {
		t.Fatalf("Health: %v", err)
	}

	// Seed, back up, wipe, and restore via Apply — the fork's restore step.
	cs, err := connString(localEnv, "")
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
	_ = conn.Close(ctx)

	var logs strings.Builder
	spec.Log = &logs
	if err := p.Backup(ctx, spec); err != nil {
		t.Fatalf("Backup: %v", err)
	}
	spec.Log = nil
	if err := p.Clean(ctx, spec); err != nil {
		t.Fatalf("Clean: %v", err)
	}
	if err := p.Apply(ctx, spec); err != nil {
		t.Fatalf("Apply (restore): %v", err)
	}

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
}
