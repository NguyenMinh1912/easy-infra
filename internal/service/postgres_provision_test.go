package service

import (
	"context"
	"testing"
)

// fakeDocker records the containers it was asked to ensure, standing in for the
// real docker CLI so provisioning can be tested without a daemon.
type fakeDocker struct {
	calls []containerSpec
	err   error
}

func (f *fakeDocker) EnsureContainer(_ context.Context, c containerSpec) error {
	f.calls = append(f.calls, c)
	return f.err
}

func TestPostgresLocalEnvFromFields(t *testing.T) {
	src := Config{
		"host":     "db.staging.internal",
		"port":     6543,
		"user":     "app",
		"password": "s3cret",
		"database": "appdb",
		"schema":   "app_dynamic",
	}
	got, err := (Postgres{}).LocalEnv(src)
	if err != nil {
		t.Fatalf("LocalEnv: %v", err)
	}
	if got["host"] != localHost {
		t.Errorf("host = %v, want %s", got["host"], localHost)
	}
	for key, want := range map[string]any{
		"port":     6543,
		"user":     "app",
		"password": "s3cret",
		"database": "appdb",
		"schema":   "app_dynamic",
	} {
		if got[key] != want {
			t.Errorf("%s = %v, want %v", key, got[key], want)
		}
	}
}

func TestPostgresLocalEnvFromURL(t *testing.T) {
	src := Config{
		"url": "postgres://app:s3cret@db.staging.internal:6543/appdb?search_path=app_dynamic",
	}
	got, err := (Postgres{}).LocalEnv(src)
	if err != nil {
		t.Fatalf("LocalEnv: %v", err)
	}
	// The url form is normalised into discrete fields pointed at loopback.
	if _, ok := got["url"]; ok {
		t.Error("LocalEnv kept the url form; want discrete fields")
	}
	for key, want := range map[string]any{
		"host":     localHost,
		"port":     6543,
		"user":     "app",
		"password": "s3cret",
		"database": "appdb",
		"schema":   "app_dynamic",
	} {
		if got[key] != want {
			t.Errorf("%s = %v, want %v", key, got[key], want)
		}
	}
}

func TestPostgresProvisionLaunchesContainer(t *testing.T) {
	fd := &fakeDocker{}
	// A healthy fake connection so waitReady returns on the first probe.
	pg := Postgres{
		open:   func(context.Context, string) (pgConn, error) { return &fakeConn{}, nil },
		docker: fd,
	}
	env, err := pg.LocalEnv(Config{
		"host": "remote", "port": 5544, "user": "u", "password": "p", "database": "d",
	})
	if err != nil {
		t.Fatalf("LocalEnv: %v", err)
	}
	spec := Spec{Profile: "local", Definition: Config{"version": "15"}, Env: env}
	if err := pg.Provision(context.Background(), spec); err != nil {
		t.Fatalf("Provision: %v", err)
	}

	if len(fd.calls) != 1 {
		t.Fatalf("EnsureContainer called %d times, want 1", len(fd.calls))
	}
	c := fd.calls[0]
	if c.Name != "easy-infra-local-postgres" {
		t.Errorf("container name = %q, want easy-infra-local-postgres", c.Name)
	}
	if c.Image != "postgres:15" {
		t.Errorf("image = %q, want postgres:15", c.Image)
	}
	if c.Env["POSTGRES_USER"] != "u" || c.Env["POSTGRES_DB"] != "d" || c.Env["POSTGRES_PASSWORD"] != "p" {
		t.Errorf("env = %v, want user/db/password set", c.Env)
	}
	if len(c.Ports) != 1 || c.Ports[0].Host != 5544 || c.Ports[0].Container != 5432 {
		t.Errorf("ports = %v, want host 5544 -> container 5432", c.Ports)
	}
}

func TestPostgresProvisionTrustAuthWhenNoPassword(t *testing.T) {
	fd := &fakeDocker{}
	pg := Postgres{
		open:   func(context.Context, string) (pgConn, error) { return &fakeConn{}, nil },
		docker: fd,
	}
	spec := Spec{
		Profile:    "local",
		Definition: Config{"version": "16"},
		Env:        Config{"host": localHost, "port": 5432, "user": "u", "database": "d"},
	}
	if err := pg.Provision(context.Background(), spec); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	c := fd.calls[0]
	if c.Env["POSTGRES_HOST_AUTH_METHOD"] != "trust" {
		t.Errorf("env = %v, want POSTGRES_HOST_AUTH_METHOD=trust when no password", c.Env)
	}
	if _, ok := c.Env["POSTGRES_PASSWORD"]; ok {
		t.Error("POSTGRES_PASSWORD set despite empty password")
	}
}
