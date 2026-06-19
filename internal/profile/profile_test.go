package profile

import (
	"testing"

	"github.com/minhnc/easy-infra/internal/service"
)

func TestScaffoldBuildsDefaultConfig(t *testing.T) {
	reg := service.DefaultRegistry()
	prof, err := Scaffold(reg, "postgres", "redis")
	if err != nil {
		t.Fatalf("Scaffold: %v", err)
	}
	if err := prof.Validate(reg); err != nil {
		t.Fatalf("Validate scaffolded profile: %v", err)
	}
	if _, ok := prof.Services["postgres"]; !ok {
		t.Error("expected postgres config")
	}
	if _, ok := prof.Services["redis"]; !ok {
		t.Error("expected redis config")
	}
}

func TestValidate(t *testing.T) {
	reg := service.DefaultRegistry()

	// cfg returns a service's default (merged) config; fatal if the service is
	// somehow unregistered.
	cfg := func(name string) service.Config {
		svc, ok := reg.Get(name)
		if !ok {
			t.Fatalf("service %q not registered", name)
		}
		return service.DefaultConfig(svc)
	}

	tests := []struct {
		name    string
		profile *Profile
		wantErr bool
	}{
		{
			name: "valid",
			profile: &Profile{Services: map[string]ServiceEntry{
				"postgres": {Config: cfg("postgres")},
				"redis":    {Config: cfg("redis")},
			}},
			wantErr: false,
		},
		{
			name: "valid with two of the same type",
			profile: &Profile{Services: map[string]ServiceEntry{
				"postgres":     {Config: cfg("postgres")},
				"db-analytics": {Type: "postgres", Name: "Analytics", Config: cfg("postgres")},
			}},
			wantErr: false,
		},
		{
			name:    "no services",
			profile: &Profile{Services: map[string]ServiceEntry{}},
			wantErr: true,
		},
		{
			name: "unknown service",
			profile: &Profile{Services: map[string]ServiceEntry{
				"mongodb": {},
			}},
			wantErr: true,
		},
		{
			name: "unknown explicit type",
			profile: &Profile{Services: map[string]ServiceEntry{
				"db-1": {Type: "mongodb"},
			}},
			wantErr: true,
		},
		{
			name: "invalid config",
			profile: &Profile{Services: map[string]ServiceEntry{
				"postgres": {Config: service.Config{"port": 5432}}, // missing required host/user/database
			}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.profile.Validate(reg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
