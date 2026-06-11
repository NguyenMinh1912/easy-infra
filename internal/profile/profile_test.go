package profile

import (
	"path/filepath"
	"testing"

	"github.com/minhnc/easy-infra/internal/service"
)

func TestScaffoldSaveLoadRoundtrip(t *testing.T) {
	reg := service.DefaultRegistry()
	prof, err := Scaffold(reg, "postgres", "redis")
	if err != nil {
		t.Fatalf("Scaffold: %v", err)
	}

	dir := t.TempDir()
	path := Path(dir, "default")
	if err := prof.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := loaded.Validate(reg, []string{"postgres", "redis"}); err != nil {
		t.Fatalf("Validate roundtripped profile: %v", err)
	}
	if _, ok := loaded.Services["postgres"]; !ok {
		t.Error("expected postgres env after roundtrip")
	}
}

func TestList(t *testing.T) {
	reg := service.DefaultRegistry()
	dir := t.TempDir()
	for _, name := range []string{"default", "staging"} {
		prof, _ := Scaffold(reg, "redis")
		if err := prof.Save(Path(dir, name)); err != nil {
			t.Fatalf("Save %s: %v", name, err)
		}
	}
	names, err := List(dir)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(names) != 2 || names[0] != "default" || names[1] != "staging" {
		t.Errorf("List() = %v, want [default staging]", names)
	}
}

func TestListMissingDir(t *testing.T) {
	names, err := List(filepath.Join(t.TempDir(), "nope"))
	if err != nil {
		t.Fatalf("List(missing) error = %v, want nil", err)
	}
	if len(names) != 0 {
		t.Errorf("List(missing) = %v, want empty", names)
	}
}

func TestValidate(t *testing.T) {
	reg := service.DefaultRegistry()
	defined := []string{"postgres", "redis"}

	// env returns a service's default env config; fatal if the service is
	// somehow unregistered.
	env := func(name string) service.Config {
		svc, ok := reg.Get(name)
		if !ok {
			t.Fatalf("service %q not registered", name)
		}
		return svc.DefaultEnv()
	}

	tests := []struct {
		name    string
		profile *Profile
		wantErr bool
	}{
		{
			name: "valid covers all defined services",
			profile: &Profile{Services: map[string]service.Config{
				"postgres": env("postgres"),
				"redis":    env("redis"),
			}},
			wantErr: false,
		},
		{
			name: "missing a defined service",
			profile: &Profile{Services: map[string]service.Config{
				"postgres": env("postgres"),
			}},
			wantErr: true,
		},
		{
			name: "configures undefined service",
			profile: &Profile{Services: map[string]service.Config{
				"postgres": env("postgres"),
				"redis":    env("redis"),
				"minio":    env("minio"),
			}},
			wantErr: true,
		},
		{
			name: "invalid env",
			profile: &Profile{Services: map[string]service.Config{
				"postgres": {"port": 5432}, // missing required host/user/database
				"redis":    env("redis"),
			}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.profile.Validate(reg, defined)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
