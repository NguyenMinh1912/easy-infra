package project

import (
	"path/filepath"
	"testing"

	"github.com/minhnc/easy-infra/internal/config"
	"github.com/minhnc/easy-infra/internal/service"
	"github.com/minhnc/easy-infra/internal/state"
)

// newTestProject builds an in-memory project rooted at a temp dir with the
// given services defined and no profiles yet.
func newTestProject(t *testing.T, services ...string) *Project {
	t.Helper()
	reg := service.DefaultRegistry()
	cfg, err := config.Scaffold(reg, services...)
	if err != nil {
		t.Fatalf("Scaffold config: %v", err)
	}
	dir := t.TempDir()
	return &Project{
		Config:   cfg,
		State:    &state.State{},
		Registry: reg,
		Paths: Paths{
			Config:      filepath.Join(dir, "easy-infra.yml"),
			State:       filepath.Join(dir, "state.json"),
			ProfilesDir: filepath.Join(dir, "profiles"),
		},
	}
}

func TestAddProfile(t *testing.T) {
	p := newTestProject(t, "postgres", "redis")

	prof, err := p.AddProfile("staging")
	if err != nil {
		t.Fatalf("AddProfile: %v", err)
	}
	// The new profile carries default env config for every defined service.
	for _, name := range []string{"postgres", "redis"} {
		if _, ok := prof.Services[name]; !ok {
			t.Errorf("scaffolded profile missing service %q", name)
		}
	}
	// It is persisted and listed.
	names, err := p.Profiles()
	if err != nil {
		t.Fatalf("Profiles: %v", err)
	}
	if len(names) != 1 || names[0] != "staging" {
		t.Errorf("Profiles() = %v, want [staging]", names)
	}
	// And it loads and validates.
	if _, err := p.LoadProfile("staging"); err != nil {
		t.Errorf("LoadProfile: %v", err)
	}
}

func TestAddProfileDuplicate(t *testing.T) {
	p := newTestProject(t, "postgres", "redis")
	if _, err := p.AddProfile("staging"); err != nil {
		t.Fatalf("AddProfile: %v", err)
	}
	if _, err := p.AddProfile("staging"); err == nil {
		t.Error("AddProfile(duplicate) error = nil, want error")
	}
}

func TestRemoveProfile(t *testing.T) {
	p := newTestProject(t, "postgres", "redis")
	if _, err := p.AddProfile("staging"); err != nil {
		t.Fatalf("AddProfile: %v", err)
	}
	if err := p.RemoveProfile("staging"); err != nil {
		t.Fatalf("RemoveProfile: %v", err)
	}
	names, err := p.Profiles()
	if err != nil {
		t.Fatalf("Profiles: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("Profiles() = %v, want empty", names)
	}
}

func TestRemoveProfileMissing(t *testing.T) {
	p := newTestProject(t, "postgres", "redis")
	if err := p.RemoveProfile("nope"); err == nil {
		t.Error("RemoveProfile(missing) error = nil, want error")
	}
}

func TestRemoveProfileActive(t *testing.T) {
	p := newTestProject(t, "postgres", "redis")
	if _, err := p.AddProfile("staging"); err != nil {
		t.Fatalf("AddProfile: %v", err)
	}
	p.State.ActiveProfile = "staging"
	if err := p.RemoveProfile("staging"); err == nil {
		t.Error("RemoveProfile(active) error = nil, want error")
	}
}
