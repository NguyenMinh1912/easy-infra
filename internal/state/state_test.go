package state

import (
	"errors"
	"io/fs"
	"path/filepath"
	"testing"
)

func TestSaveLoadRoundtrip(t *testing.T) {
	// Save into a nested dir to exercise directory creation.
	path := filepath.Join(t.TempDir(), Dir, "state.json")
	want := &State{ActiveProfile: "staging-like"}
	if err := want.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.ActiveProfile != want.ActiveProfile {
		t.Errorf("ActiveProfile = %q, want %q", got.ActiveProfile, want.ActiveProfile)
	}
}

func TestLoadMissingIsNotExist(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "absent.json"))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Load(missing) error = %v, want fs.ErrNotExist", err)
	}
}

func TestExists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if Exists(path) {
		t.Error("Exists() = true before save")
	}
	if err := (&State{}).Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !Exists(path) {
		t.Error("Exists() = false after save")
	}
}
