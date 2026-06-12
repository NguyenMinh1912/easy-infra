package service

import (
	"context"
	"errors"
	"testing"
)

// TestCleanableDefaultsTrue confirms a definition with no `cleanable` flag is
// treated as cleanable, preserving the prior behaviour.
func TestCleanableDefaultsTrue(t *testing.T) {
	ok, err := cleanable(Config{})
	if err != nil {
		t.Fatalf("cleanable: %v", err)
	}
	if !ok {
		t.Error("absent cleanable flag should default to true")
	}
}

// TestCleanableAcceptsStringFalse covers the web UI path, which submits the
// flag as the string "false" rather than a real boolean.
func TestCleanableAcceptsStringFalse(t *testing.T) {
	ok, err := cleanable(Config{cleanableKey: "false"})
	if err != nil {
		t.Fatalf("cleanable: %v", err)
	}
	if ok {
		t.Error(`cleanable "false" should disable cleaning`)
	}
}

// TestValidateDefinitionRejectsBadCleanable confirms a non-boolean flag is an
// actionable validation error across every service.
func TestValidateDefinitionRejectsBadCleanable(t *testing.T) {
	reg := DefaultRegistry()
	for _, name := range reg.Names() {
		svc, _ := reg.Get(name)
		def := svc.DefaultDefinition()
		def[cleanableKey] = "maybe"
		if err := svc.ValidateDefinition(def); err == nil {
			t.Errorf("%s: expected a non-boolean cleanable to fail validation", name)
		}
	}
}

// TestCleanProtectedReturnsErrProtected confirms every service refuses to clean
// when its definition marks it not cleanable, leaving its data untouched.
func TestCleanProtectedReturnsErrProtected(t *testing.T) {
	reg := DefaultRegistry()
	ctx := context.Background()
	spec := Spec{Definition: Config{cleanableKey: false}}
	for _, name := range reg.Names() {
		svc, _ := reg.Get(name)
		if err := svc.Clean(ctx, spec); !errors.Is(err, ErrProtected) {
			t.Errorf("%s clean: got %v, want ErrProtected", name, err)
		}
	}
}

// TestPostgresCleanProtectedSkipsConnect confirms the guard short-circuits
// before any connection is opened, so a protected service is never touched.
func TestPostgresCleanProtectedSkipsConnect(t *testing.T) {
	fc := &fakeConn{}
	spec := Spec{Definition: Config{cleanableKey: false}, Env: Postgres{}.DefaultEnv()}
	if err := withConn(fc).Clean(context.Background(), spec); !errors.Is(err, ErrProtected) {
		t.Fatalf("Clean: got %v, want ErrProtected", err)
	}
	if len(fc.execs) != 0 {
		t.Errorf("protected Clean ran statements %v, want none", fc.execs)
	}
}
