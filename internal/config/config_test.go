package config

import (
	"path/filepath"
	"testing"

	"github.com/minhnc/easy-infra/internal/service"
)

func TestScaffoldSaveLoadRoundtrip(t *testing.T) {
	reg := service.DefaultRegistry()
	cfg, err := Scaffold(reg, "default", "postgres", "redis")
	if err != nil {
		t.Fatalf("Scaffold: %v", err)
	}

	path := filepath.Join(t.TempDir(), "easy-infra.yml")
	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := loaded.Validate(reg); err != nil {
		t.Fatalf("Validate roundtripped config: %v", err)
	}
	if _, ok := loaded.Profile("default"); !ok {
		t.Error("expected default profile after roundtrip")
	}
}

func TestScaffoldUnknownService(t *testing.T) {
	if _, err := Scaffold(service.DefaultRegistry(), "default", "mongodb"); err == nil {
		t.Error("expected error scaffolding unknown service")
	}
}

func TestValidate(t *testing.T) {
	reg := service.DefaultRegistry()
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "valid",
			cfg:     Config{Version: CurrentVersion, Profiles: map[string]Profile{"default": {Services: map[string]service.Config{"redis": {"port": 6379}}}}},
			wantErr: false,
		},
		{
			name:    "wrong version",
			cfg:     Config{Version: 999, Profiles: map[string]Profile{"default": {Services: map[string]service.Config{"redis": {}}}}},
			wantErr: true,
		},
		{
			name:    "no profiles",
			cfg:     Config{Version: CurrentVersion},
			wantErr: true,
		},
		{
			name:    "empty profile",
			cfg:     Config{Version: CurrentVersion, Profiles: map[string]Profile{"default": {}}},
			wantErr: true,
		},
		{
			name:    "unknown service",
			cfg:     Config{Version: CurrentVersion, Profiles: map[string]Profile{"default": {Services: map[string]service.Config{"mongodb": {}}}}},
			wantErr: true,
		},
		{
			name:    "invalid service config",
			cfg:     Config{Version: CurrentVersion, Profiles: map[string]Profile{"default": {Services: map[string]service.Config{"redis": {"port": 99999}}}}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate(reg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
