package project

import (
	"path/filepath"
	"testing"

	"github.com/minhnc/easy-infra/internal/config"
	"github.com/minhnc/easy-infra/internal/service"
	"github.com/minhnc/easy-infra/internal/state"
)

// newTestProject builds an in-memory project rooted at a temp dir with no
// profiles yet. New profiles scaffold the conventional default services
// (project.DefaultServices).
func newTestProject(t *testing.T) *Project {
	t.Helper()
	reg := service.DefaultRegistry()
	cfg := config.Scaffold()
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
	p := newTestProject(t)

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
	p := newTestProject(t)
	if _, err := p.AddProfile("staging"); err != nil {
		t.Fatalf("AddProfile: %v", err)
	}
	if _, err := p.AddProfile("staging"); err == nil {
		t.Error("AddProfile(duplicate) error = nil, want error")
	}
}

func TestRemoveProfile(t *testing.T) {
	p := newTestProject(t)
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

func TestForkLocalProfile(t *testing.T) {
	p := newTestProject(t)
	if _, err := p.AddProfile("staging"); err != nil {
		t.Fatalf("AddProfile: %v", err)
	}

	localEnv := service.Config{
		"host": "127.0.0.1", "port": 5432, "user": "app", "password": "app", "database": "app",
	}
	name, err := p.ForkLocalProfile("staging", "postgres", localEnv)
	if err != nil {
		t.Fatalf("ForkLocalProfile: %v", err)
	}
	if name != LocalProfile {
		t.Errorf("name = %q, want %q", name, LocalProfile)
	}

	// The local profile carries the localised postgres env plus a copy of the
	// source's other services, so it stays valid.
	local, err := p.LoadProfile(LocalProfile)
	if err != nil {
		t.Fatalf("LoadProfile(local): %v", err)
	}
	if local.Services["postgres"].Config["host"] != "127.0.0.1" {
		t.Errorf("postgres host = %v, want 127.0.0.1", local.Services["postgres"].Config["host"])
	}
	if _, ok := local.Services["redis"]; !ok {
		t.Error("local profile missing copied service redis")
	}
}

func TestForkLocalProfilePreservesPriorForks(t *testing.T) {
	p := newTestProject(t)
	if _, err := p.AddProfile("staging"); err != nil {
		t.Fatalf("AddProfile: %v", err)
	}

	// First fork localises postgres.
	pgEnv := service.Config{
		"host": "127.0.0.1", "port": 5432, "user": "app", "password": "app", "database": "app",
	}
	if _, err := p.ForkLocalProfile("staging", "postgres", pgEnv); err != nil {
		t.Fatalf("ForkLocalProfile(postgres): %v", err)
	}

	// A second fork of redis must keep postgres's prior localisation intact.
	redisEnv := service.Config{"host": "127.0.0.1", "port": 6379}
	if _, err := p.ForkLocalProfile("staging", "redis", redisEnv); err != nil {
		t.Fatalf("ForkLocalProfile(redis): %v", err)
	}
	local, err := p.LoadProfile(LocalProfile)
	if err != nil {
		t.Fatalf("LoadProfile(local): %v", err)
	}
	if local.Services["postgres"].Config["host"] != "127.0.0.1" {
		t.Errorf("postgres host = %v, want preserved 127.0.0.1", local.Services["postgres"].Config["host"])
	}
	if local.Services["redis"].Config["port"] != 6379 {
		t.Errorf("redis port = %v, want 6379", local.Services["redis"].Config["port"])
	}
}

func TestRemoveProfileMissing(t *testing.T) {
	p := newTestProject(t)
	if err := p.RemoveProfile("nope"); err == nil {
		t.Error("RemoveProfile(missing) error = nil, want error")
	}
}

func TestRemoveProfileActive(t *testing.T) {
	p := newTestProject(t)
	if _, err := p.AddProfile("staging"); err != nil {
		t.Fatalf("AddProfile: %v", err)
	}
	p.State.ActiveProfile = "staging"
	if err := p.RemoveProfile("staging"); err == nil {
		t.Error("RemoveProfile(active) error = nil, want error")
	}
}
