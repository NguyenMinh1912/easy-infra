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

func TestCreateWorkspaceScaffoldsDefault(t *testing.T) {
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
	if ws.ActiveProfile != DefaultProfile {
		t.Errorf("ActiveProfile = %q, want %q", ws.ActiveProfile, DefaultProfile)
	}
	p, err := Open(st, reg, ws.ID)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := p.LoadProfile(DefaultProfile); err != nil {
		t.Errorf("default profile invalid: %v", err)
	}
}

func TestAddProfile(t *testing.T) {
	p := newTestProject(t)

	prof, err := p.AddProfile("staging")
	if err != nil {
		t.Fatalf("AddProfile: %v", err)
	}
	for _, name := range []string{"postgres", "redis"} {
		if _, ok := prof.Services[name]; !ok {
			t.Errorf("scaffolded profile missing service %q", name)
		}
	}
	names, err := p.Profiles()
	if err != nil {
		t.Fatalf("Profiles: %v", err)
	}
	if len(names) != 1 || names[0] != "staging" {
		t.Errorf("Profiles() = %v, want [staging]", names)
	}
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
	if err := p.SetActiveProfile("staging"); err != nil {
		t.Fatalf("SetActiveProfile: %v", err)
	}
	if err := p.RemoveProfile("staging"); err == nil {
		t.Error("RemoveProfile(active) error = nil, want error")
	}
}
