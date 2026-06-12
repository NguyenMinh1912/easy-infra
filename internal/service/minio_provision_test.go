package service

import (
	"context"
	"testing"
)

func TestMinIOLocalEnv(t *testing.T) {
	src := Config{
		"host":        "storage.staging.internal",
		"port":        9100,
		"consolePort": 9101,
		"user":        "minioadmin",
		"password":    "s3cret",
		"secure":      true,
	}
	got, err := (MinIO{}).LocalEnv(src)
	if err != nil {
		t.Fatalf("LocalEnv: %v", err)
	}
	if got["host"] != localHost {
		t.Errorf("host = %v, want %s", got["host"], localHost)
	}
	for key, want := range map[string]any{
		"port":        9100,
		"consolePort": 9101,
		"user":        "minioadmin",
		"password":    "s3cret",
		"secure":      true,
	} {
		if got[key] != want {
			t.Errorf("%s = %v, want %v", key, got[key], want)
		}
	}
}

func TestMinIOProvisionLaunchesContainer(t *testing.T) {
	fd := &fakeDocker{}
	// A healthy fake client so waitReady returns on the first probe.
	m := MinIO{
		open:   func(context.Context, minioParams) (s3Client, error) { return newFakeS3(), nil },
		docker: fd,
	}
	env, err := m.LocalEnv(Config{
		"host": "remote", "port": 9200, "consolePort": 9201, "user": "u", "password": "password123",
	})
	if err != nil {
		t.Fatalf("LocalEnv: %v", err)
	}
	spec := Spec{Profile: "local", Definition: Config{"version": "RELEASE.2024"}, Env: env}
	if err := m.Provision(context.Background(), spec); err != nil {
		t.Fatalf("Provision: %v", err)
	}

	if len(fd.calls) != 1 {
		t.Fatalf("EnsureContainer called %d times, want 1", len(fd.calls))
	}
	c := fd.calls[0]
	if c.Name != "easy-infra-local-minio" {
		t.Errorf("container name = %q, want easy-infra-local-minio", c.Name)
	}
	if c.Image != "minio/minio:RELEASE.2024" {
		t.Errorf("image = %q, want minio/minio:RELEASE.2024", c.Image)
	}
	if c.Env["MINIO_ROOT_USER"] != "u" || c.Env["MINIO_ROOT_PASSWORD"] != "password123" {
		t.Errorf("env = %v, want root user/password set", c.Env)
	}
	wantPorts := map[int]int{9200: 9000, 9201: 9001}
	if len(c.Ports) != 2 {
		t.Fatalf("ports = %v, want 2 mappings", c.Ports)
	}
	for _, p := range c.Ports {
		if wantPorts[p.Host] != p.Container {
			t.Errorf("port mapping %v not in want %v", p, wantPorts)
		}
	}
	wantCmd := []string{"server", "/data", "--console-address", ":9001"}
	if len(c.Cmd) != len(wantCmd) {
		t.Fatalf("cmd = %v, want %v", c.Cmd, wantCmd)
	}
	for i := range wantCmd {
		if c.Cmd[i] != wantCmd[i] {
			t.Errorf("cmd[%d] = %q, want %q", i, c.Cmd[i], wantCmd[i])
		}
	}
}
