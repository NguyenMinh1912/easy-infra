package service

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestMinIOProvisionIntegration exercises real container provisioning: it
// launches a local minio via the docker CLI, waits for it to come up, then runs
// a backup → clean → restore round-trip against it — the same path "fork to
// local" drives. It is skipped unless EASY_INFRA_DOCKER_IT=1 and requires a
// working docker daemon with a pullable minio image.
func TestMinIOProvisionIntegration(t *testing.T) {
	if os.Getenv("EASY_INFRA_DOCKER_IT") == "" {
		t.Skip("set EASY_INFRA_DOCKER_IT=1 to run the real-docker provisioning test")
	}
	t.Chdir(t.TempDir()) // backups land under a throwaway cwd

	const profile = "fork-it"
	// High ports unlikely to collide with a local server.
	source := Config{
		"host": "example.invalid", "port": 9900, "consolePort": 9901,
		"user": "minioadmin", "password": "minioadmin",
	}
	m := MinIO{}
	localEnv, err := m.LocalEnv(source)
	if err != nil {
		t.Fatalf("LocalEnv: %v", err)
	}

	spec := Spec{Profile: profile, Definition: Config{"version": "latest"}, Env: localEnv}
	name := localContainerName(profile, "minio")
	t.Cleanup(func() {
		_ = exec.Command("docker", "rm", "-f", name).Run()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	if err := m.Provision(ctx, spec); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	// Provisioning is idempotent: a second call reuses the running container.
	if err := m.Provision(ctx, spec); err != nil {
		t.Fatalf("Provision (idempotent): %v", err)
	}
	if err := m.Health(ctx, spec); err != nil {
		t.Fatalf("Health: %v", err)
	}

	// Seed a bucket and object directly through the client seam.
	params, err := minioParamsFrom(localEnv)
	if err != nil {
		t.Fatal(err)
	}
	client, err := realS3Opener(ctx, params)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	if err := client.MakeBucket(ctx, "assets"); err != nil {
		t.Fatalf("make bucket: %v", err)
	}
	if err := client.PutObject(ctx, "assets", "logo.png", bytes.NewReader([]byte("PNGDATA")), 7, "image/png"); err != nil {
		t.Fatalf("put object: %v", err)
	}

	var logs strings.Builder
	spec.Log = &logs
	if err := m.Backup(ctx, spec); err != nil {
		t.Fatalf("Backup: %v", err)
	}
	spec.Log = nil
	if err := m.Clean(ctx, Spec{Profile: profile, Env: localEnv, Definition: Config{cleanableKey: true}}); err != nil {
		t.Fatalf("Clean: %v", err)
	}
	if err := m.Apply(ctx, spec); err != nil {
		t.Fatalf("Apply (restore): %v", err)
	}

	r, _, err := client.GetObject(ctx, "assets", "logo.png")
	if err != nil {
		t.Fatalf("get restored object: %v", err)
	}
	defer r.Close()
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read restored object: %v", err)
	}
	if string(got) != "PNGDATA" {
		t.Errorf("restored object = %q, want PNGDATA", got)
	}
}
