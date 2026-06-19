package store

import (
	"errors"
	"testing"

	"github.com/minhnc/easy-infra/internal/profile"
	"github.com/minhnc/easy-infra/internal/service"
)

// open returns a Store backed by a fresh database in a temp config dir, so tests
// never touch the real user config directory.
func open(t *testing.T) *Store {
	t.Helper()
	t.Setenv(configDirEnv, t.TempDir())
	s, err := Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestOpenEmpty(t *testing.T) {
	s := open(t)
	ws, err := s.ListWorkspaces()
	if err != nil {
		t.Fatalf("ListWorkspaces: %v", err)
	}
	if len(ws) != 0 {
		t.Errorf("expected no workspaces, got %d", len(ws))
	}
	if _, ok, err := s.ActiveWorkspace(); err != nil || ok {
		t.Errorf("ActiveWorkspace ok=%v err=%v, want false/nil", ok, err)
	}
}

func TestCreateAndListWorkspaces(t *testing.T) {
	s := open(t)
	a, err := s.CreateWorkspace("app")
	if err != nil {
		t.Fatalf("CreateWorkspace: %v", err)
	}
	if a.ID == 0 || a.Name != "app" {
		t.Fatalf("unexpected workspace %+v", a)
	}
	if _, err := s.CreateWorkspace("other"); err != nil {
		t.Fatalf("CreateWorkspace other: %v", err)
	}
	list, err := s.ListWorkspaces()
	if err != nil {
		t.Fatalf("ListWorkspaces: %v", err)
	}
	if len(list) != 2 || list[0].Name != "app" || list[1].Name != "other" {
		t.Errorf("unexpected list %+v", list)
	}

	got, err := s.GetWorkspace(a.ID)
	if err != nil || got.Name != "app" {
		t.Errorf("GetWorkspace = %+v, %v", got, err)
	}
}

func TestCreateWorkspaceRejectsDuplicate(t *testing.T) {
	s := open(t)
	if _, err := s.CreateWorkspace("app"); err != nil {
		t.Fatalf("CreateWorkspace: %v", err)
	}
	if _, err := s.CreateWorkspace("app"); !errors.Is(err, ErrWorkspaceExists) {
		t.Errorf("err = %v, want ErrWorkspaceExists", err)
	}
}

func TestRenameWorkspace(t *testing.T) {
	s := open(t)
	a, _ := s.CreateWorkspace("app")
	b, _ := s.CreateWorkspace("other")

	if err := s.RenameWorkspace(a.ID, "renamed"); err != nil {
		t.Fatalf("RenameWorkspace: %v", err)
	}
	got, _ := s.GetWorkspace(a.ID)
	if got.Name != "renamed" {
		t.Errorf("name = %q, want renamed", got.Name)
	}
	// Collision with an existing name is rejected.
	if err := s.RenameWorkspace(a.ID, "other"); !errors.Is(err, ErrWorkspaceExists) {
		t.Errorf("err = %v, want ErrWorkspaceExists", err)
	}
	_ = b
	// Unknown id.
	if err := s.RenameWorkspace(9999, "x"); !errors.Is(err, ErrWorkspaceNotFound) {
		t.Errorf("err = %v, want ErrWorkspaceNotFound", err)
	}
}

func TestActiveWorkspace(t *testing.T) {
	s := open(t)
	a, _ := s.CreateWorkspace("app")

	if err := s.SetActiveWorkspace(a.ID); err != nil {
		t.Fatalf("SetActiveWorkspace: %v", err)
	}
	got, ok, err := s.ActiveWorkspace()
	if err != nil || !ok || got.ID != a.ID {
		t.Fatalf("ActiveWorkspace = %+v, %v, %v", got, ok, err)
	}
	if err := s.SetActiveWorkspace(9999); !errors.Is(err, ErrWorkspaceNotFound) {
		t.Errorf("err = %v, want ErrWorkspaceNotFound", err)
	}
}

func TestRemoveWorkspaceClearsActiveAndCascades(t *testing.T) {
	s := open(t)
	a, _ := s.CreateWorkspace("app")
	if err := s.CreateProfile(a.ID, "default", map[string]profile.ServiceEntry{
		"postgres": {Config: service.Config{"host": "localhost"}},
	}); err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}
	if err := s.SetActiveWorkspace(a.ID); err != nil {
		t.Fatalf("SetActiveWorkspace: %v", err)
	}

	if err := s.RemoveWorkspace(a.ID); err != nil {
		t.Fatalf("RemoveWorkspace: %v", err)
	}
	if _, ok, _ := s.ActiveWorkspace(); ok {
		t.Error("active workspace still set after removing it")
	}
	// Profiles cascaded away with the workspace.
	if names, _ := s.ListProfiles(a.ID); len(names) != 0 {
		t.Errorf("profiles survived workspace deletion: %+v", names)
	}
	if err := s.RemoveWorkspace(a.ID); !errors.Is(err, ErrWorkspaceNotFound) {
		t.Errorf("err = %v, want ErrWorkspaceNotFound", err)
	}
}

func TestSetWorkspaceActiveProfile(t *testing.T) {
	s := open(t)
	a, _ := s.CreateWorkspace("app")
	if err := s.SetWorkspaceActiveProfile(a.ID, "default"); err != nil {
		t.Fatalf("SetWorkspaceActiveProfile: %v", err)
	}
	got, _ := s.GetWorkspace(a.ID)
	if got.ActiveProfile != "default" {
		t.Errorf("ActiveProfile = %q, want default", got.ActiveProfile)
	}
	if err := s.SetWorkspaceActiveProfile(9999, "x"); !errors.Is(err, ErrWorkspaceNotFound) {
		t.Errorf("err = %v, want ErrWorkspaceNotFound", err)
	}
}

func TestProfileServicesRoundTrip(t *testing.T) {
	s := open(t)
	a, _ := s.CreateWorkspace("app")

	want := map[string]profile.ServiceEntry{
		"postgres": {Type: "postgres", Name: "primary", Config: service.Config{"host": "db.local", "user": "admin"}},
		"redis":    {Config: service.Config{"host": "cache.local"}},
	}
	if err := s.CreateProfile(a.ID, "default", want); err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}

	got, err := s.ProfileServices(a.ID, "default")
	if err != nil {
		t.Fatalf("ProfileServices: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d services, want 2: %+v", len(got), got)
	}
	pg := got["postgres"]
	if pg.Type != "postgres" || pg.Name != "primary" || pg.Config["host"] != "db.local" || pg.Config["user"] != "admin" {
		t.Errorf("postgres entry = %+v", pg)
	}
	rd := got["redis"]
	if rd.Type != "" || rd.Name != "" || rd.Config["host"] != "cache.local" {
		t.Errorf("redis entry = %+v", rd)
	}
}

func TestCreateProfileRejectsDuplicateAndUnknownWorkspace(t *testing.T) {
	s := open(t)
	a, _ := s.CreateWorkspace("app")
	svcs := map[string]profile.ServiceEntry{"redis": {Config: service.Config{}}}

	if err := s.CreateProfile(a.ID, "default", svcs); err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}
	if err := s.CreateProfile(a.ID, "default", svcs); !errors.Is(err, ErrProfileExists) {
		t.Errorf("err = %v, want ErrProfileExists", err)
	}
	if err := s.CreateProfile(9999, "p", svcs); !errors.Is(err, ErrWorkspaceNotFound) {
		t.Errorf("err = %v, want ErrWorkspaceNotFound", err)
	}
}

func TestReplaceProfileServices(t *testing.T) {
	s := open(t)
	a, _ := s.CreateWorkspace("app")
	if err := s.CreateProfile(a.ID, "default", map[string]profile.ServiceEntry{
		"postgres": {Config: service.Config{"host": "old"}},
	}); err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}

	next := map[string]profile.ServiceEntry{
		"redis": {Config: service.Config{"host": "new"}},
		"minio": {Config: service.Config{}},
	}
	if err := s.ReplaceProfileServices(a.ID, "default", next); err != nil {
		t.Fatalf("ReplaceProfileServices: %v", err)
	}
	got, _ := s.ProfileServices(a.ID, "default")
	if _, ok := got["postgres"]; ok {
		t.Error("postgres survived replace")
	}
	if len(got) != 2 || got["redis"].Config["host"] != "new" {
		t.Errorf("after replace = %+v", got)
	}
	if err := s.ReplaceProfileServices(a.ID, "ghost", next); !errors.Is(err, ErrProfileNotFound) {
		t.Errorf("err = %v, want ErrProfileNotFound", err)
	}
}

func TestProfileNotFound(t *testing.T) {
	s := open(t)
	a, _ := s.CreateWorkspace("app")
	if _, err := s.ProfileServices(a.ID, "ghost"); !errors.Is(err, ErrProfileNotFound) {
		t.Errorf("err = %v, want ErrProfileNotFound", err)
	}
	if err := s.RemoveProfile(a.ID, "ghost"); !errors.Is(err, ErrProfileNotFound) {
		t.Errorf("err = %v, want ErrProfileNotFound", err)
	}
	ok, err := s.ProfileExists(a.ID, "ghost")
	if err != nil || ok {
		t.Errorf("ProfileExists = %v, %v", ok, err)
	}
}

func TestListProfilesSorted(t *testing.T) {
	s := open(t)
	a, _ := s.CreateWorkspace("app")
	svcs := map[string]profile.ServiceEntry{"redis": {Config: service.Config{}}}
	for _, name := range []string{"staging", "default", "ci"} {
		if err := s.CreateProfile(a.ID, name, svcs); err != nil {
			t.Fatalf("CreateProfile %q: %v", name, err)
		}
	}
	got, err := s.ListProfiles(a.ID)
	if err != nil {
		t.Fatalf("ListProfiles: %v", err)
	}
	want := []string{"ci", "default", "staging"}
	if len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Errorf("ListProfiles = %v, want %v", got, want)
	}
}
