package service

import (
	"context"
	"errors"
	"testing"
)

func TestJenkinsValidateEnv(t *testing.T) {
	j := Jenkins{}
	if err := j.ValidateEnv(j.DefaultEnv()); err != nil {
		t.Fatalf("default env failed validation: %v", err)
	}

	env := j.DefaultEnv()
	delete(env, "host")
	if err := j.ValidateEnv(env); err == nil {
		t.Error("expected missing host to fail validation")
	}

	env = j.DefaultEnv()
	env["port"] = 70000
	if err := j.ValidateEnv(env); err == nil {
		t.Error("expected out-of-range port to fail validation")
	}

	// Optional credentials are accepted when present and well-typed.
	env = j.DefaultEnv()
	env["user"] = "admin"
	env["token"] = "11aa22bb"
	if err := j.ValidateEnv(env); err != nil {
		t.Errorf("expected credentials to validate: %v", err)
	}
}

func TestJenkinsHealth(t *testing.T) {
	ctx := context.Background()
	spec := Spec{Env: Jenkins{}.DefaultEnv()}

	// A reachable server makes Health succeed.
	ok := Jenkins{ping: func(context.Context, string) error { return nil }}
	if err := ok.Health(ctx, spec); err != nil {
		t.Errorf("Health with reachable server: %v", err)
	}

	// An unreachable server surfaces an actionable, wrapped error.
	boom := errors.New("connection refused")
	down := Jenkins{ping: func(context.Context, string) error { return boom }}
	if err := down.Health(ctx, spec); err == nil || !errors.Is(err, boom) {
		t.Errorf("Health with unreachable server: got %v, want wrapped %v", err, boom)
	}
}

func TestJenkinsHealthBaseURL(t *testing.T) {
	var got string
	j := Jenkins{ping: func(_ context.Context, baseURL string) error {
		got = baseURL
		return nil
	}}
	spec := Spec{Env: Config{"host": "ci.example", "port": 9090}}
	if err := j.Health(context.Background(), spec); err != nil {
		t.Fatalf("Health: %v", err)
	}
	if want := "http://ci.example:9090"; got != want {
		t.Errorf("base URL = %q, want %q", got, want)
	}
}
