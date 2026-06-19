package workspace

import (
	"path/filepath"
	"testing"
)

func TestLoadMissingIsEmpty(t *testing.T) {
	t.Setenv(configDirEnv, t.TempDir())
	r, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(r.Workspaces) != 0 || r.Active != "" {
		t.Errorf("expected empty registry, got %+v", r)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Setenv(configDirEnv, t.TempDir())
	r := &Registry{}
	if err := r.Add("app", t.TempDir()); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := r.SetActive("app"); err != nil {
		t.Fatalf("SetActive: %v", err)
	}
	if err := Save(r); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Active != "app" || len(got.Workspaces) != 1 || got.Workspaces[0].Name != "app" {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}

func TestAddRejectsDuplicates(t *testing.T) {
	dir := t.TempDir()
	r := &Registry{}
	if err := r.Add("app", dir); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := r.Add("app", t.TempDir()); err == nil {
		t.Error("expected duplicate-name rejection")
	}
	if err := r.Add("other", dir); err == nil {
		t.Error("expected duplicate-path rejection")
	}
}

func TestAddStoresAbsolutePath(t *testing.T) {
	r := &Registry{}
	if err := r.Add("app", "."); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if !filepath.IsAbs(r.Workspaces[0].Path) {
		t.Errorf("path %q is not absolute", r.Workspaces[0].Path)
	}
}

func TestSetActiveUnknown(t *testing.T) {
	r := &Registry{}
	if err := r.SetActive("nope"); err == nil {
		t.Error("expected error for unknown workspace")
	}
}

func TestRemoveFallsBackActive(t *testing.T) {
	r := &Registry{}
	_ = r.Add("a", t.TempDir())
	_ = r.Add("b", t.TempDir())
	_ = r.SetActive("a")

	if err := r.Remove("a"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if r.Active != "b" {
		t.Errorf("Active = %q, want fallback to \"b\"", r.Active)
	}
	if err := r.Remove("b"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if r.Active != "" {
		t.Errorf("Active = %q, want empty after removing all", r.Active)
	}
	if _, ok := r.ActivePath(); ok {
		t.Error("ActivePath ok = true, want false for empty registry")
	}
}

func TestRemoveUnknown(t *testing.T) {
	r := &Registry{}
	if err := r.Remove("nope"); err == nil {
		t.Error("expected error removing unknown workspace")
	}
}
