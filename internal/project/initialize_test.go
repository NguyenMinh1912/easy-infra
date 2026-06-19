package project

import (
	"testing"

	"github.com/minhnc/easy-infra/internal/service"
)

func TestInitializeScaffoldsProject(t *testing.T) {
	reg := service.DefaultRegistry()
	paths := PathsFor(t.TempDir())

	if err := Initialize(paths, reg); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if !IsInitialized(paths) {
		t.Fatal("IsInitialized = false after Initialize")
	}

	proj, err := Load(paths, reg)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if proj.State.ActiveProfile != "default" {
		t.Errorf("ActiveProfile = %q, want \"default\"", proj.State.ActiveProfile)
	}
	if _, err := proj.LoadProfile("default"); err != nil {
		t.Errorf("LoadProfile(default): %v", err)
	}
}

func TestInitializeRejectsExisting(t *testing.T) {
	reg := service.DefaultRegistry()
	paths := PathsFor(t.TempDir())

	if err := Initialize(paths, reg); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if err := Initialize(paths, reg); err == nil {
		t.Error("expected error initializing an already-initialized project")
	}
}

func TestIsInitializedFalseForEmptyDir(t *testing.T) {
	if IsInitialized(PathsFor(t.TempDir())) {
		t.Error("IsInitialized = true for an empty dir")
	}
}
