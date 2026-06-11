package service

import (
	"context"
	"errors"
	"testing"
)

func TestDefaultRegistryHasBuiltins(t *testing.T) {
	reg := DefaultRegistry()
	want := []string{"localstack", "minio", "postgres", "redis"}
	got := reg.Names()
	if len(got) != len(want) {
		t.Fatalf("Names() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Names()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestRegistryRegisterDuplicate(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Register(Redis{}); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	if err := reg.Register(Redis{}); err == nil {
		t.Error("duplicate Register: expected error, got nil")
	}
}

func TestRegistryGet(t *testing.T) {
	reg := DefaultRegistry()
	if _, ok := reg.Get("postgres"); !ok {
		t.Error("Get(postgres): expected found")
	}
	if _, ok := reg.Get("nope"); ok {
		t.Error("Get(nope): expected not found")
	}
}

// TestDefaultsValidate asserts every service's own defaults satisfy its own
// validation, for both the definition and the environment halves.
func TestDefaultsValidate(t *testing.T) {
	reg := DefaultRegistry()
	for _, name := range reg.Names() {
		svc, _ := reg.Get(name)
		if err := svc.ValidateDefinition(svc.DefaultDefinition()); err != nil {
			t.Errorf("%s default definition failed validation: %v", name, err)
		}
		if err := svc.ValidateEnv(svc.DefaultEnv()); err != nil {
			t.Errorf("%s default env failed validation: %v", name, err)
		}
	}
}

func TestValidateEnvPortRange(t *testing.T) {
	env := Redis{}.DefaultEnv()
	env["port"] = 70000
	if err := (Redis{}).ValidateEnv(env); err == nil {
		t.Error("expected out-of-range port to fail validation")
	}
	env["port"] = "abc"
	if err := (Redis{}).ValidateEnv(env); err == nil {
		t.Error("expected non-numeric port to fail validation")
	}
}

// TestLifecycleSeam asserts every registered service exposes the four
// lifecycle operations and, until Docker providers land, reports
// ErrNotImplemented so callers can degrade gracefully.
func TestLifecycleSeam(t *testing.T) {
	reg := DefaultRegistry()
	ctx := context.Background()
	spec := Spec{}
	for _, name := range reg.Names() {
		svc, _ := reg.Get(name)
		ops := map[string]func(context.Context, Spec) error{
			"apply":  svc.Apply,
			"health": svc.Health,
			"backup": svc.Backup,
			"clean":  svc.Clean,
		}
		for op, fn := range ops {
			if err := fn(ctx, spec); !errors.Is(err, ErrNotImplemented) {
				t.Errorf("%s %s: got %v, want ErrNotImplemented", name, op, err)
			}
		}
	}
}

func TestValidateEnvRequiresHost(t *testing.T) {
	// Drop the required host field from postgres env.
	env := Postgres{}.DefaultEnv()
	delete(env, "host")
	if err := (Postgres{}).ValidateEnv(env); err == nil {
		t.Error("expected missing host to fail env validation")
	}
}
