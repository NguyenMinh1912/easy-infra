package project

import (
	"testing"

	"github.com/minhnc/easy-infra/internal/service"
	"github.com/minhnc/easy-infra/internal/store"
)

// newTestProject builds a Project backed by a fresh store in a temp config dir,
// with one (active) workspace and no profiles yet.
func newTestProject(t *testing.T) *Project {
	t.Helper()
	t.Setenv("EASY_INFRA_CONFIG_DIR", t.TempDir())
	st, err := store.Open()
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	reg := service.DefaultRegistry()
	ws, err := st.CreateWorkspace("test")
	if err != nil {
		t.Fatalf("CreateWorkspace: %v", err)
	}
	p, err := Open(st, reg, ws.ID)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return p
}

// seedStarterServices populates a profile with the conventional starter
// services (postgres, redis). AddProfile creates an empty profile, so tests
// that need a populated one call this rather than relying on a default scaffold.
func seedStarterServices(t *testing.T, p *Project, profileName string) {
	t.Helper()
	for _, svc := range []string{"postgres", "redis"} {
		if _, err := p.AddProfileService(profileName, svc, "", nil); err != nil {
			t.Fatalf("AddProfileService(%s): %v", svc, err)
		}
	}
}

func TestCreateWorkspaceHasNoProfiles(t *testing.T) {
	t.Setenv("EASY_INFRA_CONFIG_DIR", t.TempDir())
	st, err := store.Open()
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	defer st.Close()
	reg := service.DefaultRegistry()

	ws, err := CreateWorkspace(st, reg, "app")
	if err != nil {
		t.Fatalf("CreateWorkspace: %v", err)
	}
	if ws.ActiveProfile != "" {
		t.Errorf("ActiveProfile = %q, want empty", ws.ActiveProfile)
	}
	p, err := Open(st, reg, ws.ID)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	names, err := p.Profiles()
	if err != nil {
		t.Fatalf("Profiles: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("Profiles = %v, want none", names)
	}
}

func TestAddProfile(t *testing.T) {
	p := newTestProject(t)

	prof, err := p.AddProfile("staging")
	if err != nil {
		t.Fatalf("AddProfile: %v", err)
	}
	// A new profile starts empty; the user adds services afterwards.
	if len(prof.Services) != 0 {
		t.Errorf("new profile services = %v, want empty", prof.Services)
	}
	names, err := p.Profiles()
	if err != nil {
		t.Fatalf("Profiles: %v", err)
	}
	if len(names) != 1 || names[0] != "staging" {
		t.Errorf("Profiles() = %v, want [staging]", names)
	}
	stored, err := p.ProfileConfig("staging")
	if err != nil {
		t.Fatalf("ProfileConfig: %v", err)
	}
	if len(stored) != 0 {
		t.Errorf("stored services = %v, want empty", stored)
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
	seedStarterServices(t, p, "staging")

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

	local, err := p.LoadProfile(LocalProfile)
	if err != nil {
		t.Fatalf("LoadProfile(local): %v", err)
	}
	if local.Services["postgres"].Config["host"] != "127.0.0.1" {
		t.Errorf("postgres host = %v, want 127.0.0.1", local.Services["postgres"].Config["host"])
	}
	// The local profile holds only what was forked. redis was never forked, so it
	// must not leak in from the source profile.
	if _, ok := local.Services["redis"]; ok {
		t.Error("local profile contains redis, but only postgres was forked")
	}
	if len(local.Services) != 1 {
		t.Errorf("local profile has %d services, want 1 (only the forked postgres)", len(local.Services))
	}
}

func TestForkLocalProfilePreservesPriorForks(t *testing.T) {
	p := newTestProject(t)
	if _, err := p.AddProfile("staging"); err != nil {
		t.Fatalf("AddProfile: %v", err)
	}
	seedStarterServices(t, p, "staging")

	pgEnv := service.Config{
		"host": "127.0.0.1", "port": 5432, "user": "app", "password": "app", "database": "app",
	}
	if _, err := p.ForkLocalProfile("staging", "postgres", pgEnv); err != nil {
		t.Fatalf("ForkLocalProfile(postgres): %v", err)
	}

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
	// Config round-trips through JSON, so a numeric port comes back as float64.
	if port, _ := local.Services["redis"].Config["port"].(float64); port != 6379 {
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
	seedStarterServices(t, p, "staging")
	if err := p.SetActiveProfile("staging"); err != nil {
		t.Fatalf("SetActiveProfile: %v", err)
	}
	if err := p.RemoveProfile("staging"); err == nil {
		t.Error("RemoveProfile(active) error = nil, want error")
	}
}
