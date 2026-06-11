package service

import "testing"

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

func TestDefaultConfigsValidate(t *testing.T) {
	reg := DefaultRegistry()
	for _, name := range reg.Names() {
		svc, _ := reg.Get(name)
		if err := svc.Validate(svc.DefaultConfig()); err != nil {
			t.Errorf("%s default config failed validation: %v", name, err)
		}
	}
}

func TestValidatePortRange(t *testing.T) {
	if err := (Redis{}).Validate(Config{"port": 70000}); err == nil {
		t.Error("expected out-of-range port to fail validation")
	}
	if err := (Redis{}).Validate(Config{"port": "abc"}); err == nil {
		t.Error("expected non-numeric port to fail validation")
	}
}

func TestValidateRequiredString(t *testing.T) {
	// postgres requires "database"; an empty map should fail.
	if err := (Postgres{}).Validate(Config{"port": 5432}); err == nil {
		t.Error("expected missing database to fail validation")
	}
}
