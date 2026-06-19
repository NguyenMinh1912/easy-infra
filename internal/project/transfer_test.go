package project

import (
	"testing"

	"github.com/minhnc/easy-infra/internal/service"
)

// TestExportImportRoundTrip seeds a workspace, exports it, imports it back into
// the same store, and checks the imported copy mirrors the original (under a
// fresh, unique name).
func TestExportImportRoundTrip(t *testing.T) {
	p := newTestProject(t)
	if _, err := p.AddProfile("default"); err != nil {
		t.Fatalf("AddProfile: %v", err)
	}
	seedStarterServices(t, p, "default")
	if err := p.SetActiveProfile("default"); err != nil {
		t.Fatalf("SetActiveProfile: %v", err)
	}

	exp, err := ExportWorkspace(p.Store, p.Workspace.ID)
	if err != nil {
		t.Fatalf("ExportWorkspace: %v", err)
	}
	if exp.Name != "test" || exp.ActiveProfile != "default" {
		t.Fatalf("export = {%q, %q}, want {test, default}", exp.Name, exp.ActiveProfile)
	}
	if len(exp.Profiles) != 1 || len(exp.Profiles[0].Services) != 2 {
		t.Fatalf("export profiles = %+v, want 1 profile with 2 services", exp.Profiles)
	}

	ws, err := ImportWorkspace(p.Store, p.Registry, exp)
	if err != nil {
		t.Fatalf("ImportWorkspace: %v", err)
	}
	// The name "test" is taken by the original, so the import is suffixed.
	if ws.Name != "test (2)" {
		t.Errorf("imported name = %q, want %q", ws.Name, "test (2)")
	}
	if ws.ActiveProfile != "default" {
		t.Errorf("imported active profile = %q, want %q", ws.ActiveProfile, "default")
	}

	imported, err := Open(p.Store, p.Registry, ws.ID)
	if err != nil {
		t.Fatalf("Open imported: %v", err)
	}
	services, err := imported.ProfileConfig("default")
	if err != nil {
		t.Fatalf("ProfileConfig: %v", err)
	}
	if len(services) != 2 {
		t.Errorf("imported services = %d, want 2", len(services))
	}
}

// TestImportRejectsUnknownService ensures a payload referencing a service the
// registry does not know is rejected before any rows are written.
func TestImportRejectsUnknownService(t *testing.T) {
	p := newTestProject(t)
	exp := WorkspaceExport{
		Version: exportVersion,
		Name:    "incoming",
		Profiles: []ProfileExport{{
			Name:     "default",
			Services: []ServiceExport{{ID: "mystery", Type: "mystery", Config: service.Config{}}},
		}},
	}
	if _, err := ImportWorkspace(p.Store, p.Registry, exp); err == nil {
		t.Fatal("ImportWorkspace accepted an unknown service, want error")
	}
}

// TestImportRejectsUnsupportedVersion guards the format check.
func TestImportRejectsUnsupportedVersion(t *testing.T) {
	p := newTestProject(t)
	exp := WorkspaceExport{Version: exportVersion + 1, Name: "incoming"}
	if _, err := ImportWorkspace(p.Store, p.Registry, exp); err == nil {
		t.Fatal("ImportWorkspace accepted an unsupported version, want error")
	}
}
