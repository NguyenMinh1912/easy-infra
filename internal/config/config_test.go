package config

import (
	"path/filepath"
	"testing"
)

func TestScaffoldSaveLoadRoundtrip(t *testing.T) {
	cfg := Scaffold()

	path := filepath.Join(t.TempDir(), "easy-infra.yml")
	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := loaded.Validate(); err != nil {
		t.Fatalf("Validate roundtripped config: %v", err)
	}
	if loaded.Version != CurrentVersion {
		t.Errorf("Version = %d, want %d", loaded.Version, CurrentVersion)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "valid",
			cfg:     Config{Version: CurrentVersion},
			wantErr: false,
		},
		{
			name:    "wrong version",
			cfg:     Config{Version: 999},
			wantErr: true,
		},
		{
			name:    "zero version",
			cfg:     Config{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
