package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/minhnc/easy-infra/internal/project"
	"github.com/minhnc/easy-infra/internal/service"
	"github.com/minhnc/easy-infra/internal/workspace"
)

// newWorkspaceServer builds a server with a single active workspace at root and
// redirects the registry to a temp config dir so create/activate/remove persist
// without touching the real user config directory.
func newWorkspaceServer(t *testing.T, root string) *Server {
	t.Helper()
	t.Setenv("EASY_INFRA_CONFIG_DIR", t.TempDir())
	ws := &workspace.Registry{}
	if err := ws.Add("primary", root); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := ws.SetActive("primary"); err != nil {
		t.Fatalf("SetActive: %v", err)
	}
	return New(service.DefaultRegistry(), ws, emptyUI)
}

func decodeWorkspaces(t *testing.T, rec interface{ Bytes() []byte }) workspacesResponse {
	t.Helper()
	var got workspacesResponse
	if err := json.Unmarshal(rec.Bytes(), &got); err != nil {
		t.Fatalf("decode workspaces: %v", err)
	}
	return got
}

func TestListWorkspaces(t *testing.T) {
	root := t.TempDir()
	srv := newWorkspaceServer(t, root)

	rec := doRequest(t, srv, http.MethodGet, "/api/workspaces", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", rec.Code)
	}
	got := decodeWorkspaces(t, rec.Body)
	if got.Active != "primary" || len(got.Workspaces) != 1 {
		t.Fatalf("unexpected list: %+v", got)
	}
	if got.Workspaces[0].Path != root || !got.Workspaces[0].Exists {
		t.Errorf("workspace = %+v, want path %q exists", got.Workspaces[0], root)
	}
	if got.Separator == "" {
		t.Error("separator is empty")
	}
}

func TestCreateWorkspaceScaffoldsAndActivates(t *testing.T) {
	srv := newWorkspaceServer(t, t.TempDir())
	target := filepath.Join(t.TempDir(), "new-app")

	body := `{"name":"new-app","path":"` + target + `"}`
	rec := doRequest(t, srv, http.MethodPost, "/api/workspaces", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	got := decodeWorkspaces(t, rec.Body)
	if got.Active != "new-app" {
		t.Errorf("Active = %q, want \"new-app\"", got.Active)
	}
	// The folder was scaffolded into a real project.
	if !project.IsInitialized(project.PathsFor(target)) {
		t.Error("target folder was not initialized")
	}
	// And /api/status now reflects that folder.
	_, status := getStatus(t, srv)
	if !status.Initialized || status.ActiveProfile != "default" {
		t.Errorf("status = %+v, want initialized default", status)
	}
}

func TestActivateWorkspaceSwitchesFolder(t *testing.T) {
	srv := newWorkspaceServer(t, t.TempDir())

	// Create a second workspace (also makes it active).
	a := filepath.Join(t.TempDir(), "a")
	rec := doRequest(t, srv, http.MethodPost, "/api/workspaces", `{"name":"a","path":"`+a+`"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("create a: code %d (%s)", rec.Code, rec.Body.String())
	}

	// Switch back to primary, which has no project -> status not initialized.
	rec = doRequest(t, srv, http.MethodPost, "/api/workspaces/activate", `{"name":"primary"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("activate primary: code %d (%s)", rec.Code, rec.Body.String())
	}
	if _, status := getStatus(t, srv); status.Initialized {
		t.Error("status initialized after switching to an empty workspace")
	}

	// Switch to a -> initialized.
	rec = doRequest(t, srv, http.MethodPost, "/api/workspaces/activate", `{"name":"a"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("activate a: code %d", rec.Code)
	}
	if _, status := getStatus(t, srv); !status.Initialized {
		t.Error("status not initialized after switching to project workspace")
	}
}

func TestActivateUnknownWorkspace(t *testing.T) {
	srv := newWorkspaceServer(t, t.TempDir())
	rec := doRequest(t, srv, http.MethodPost, "/api/workspaces/activate", `{"name":"ghost"}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("code = %d, want 400", rec.Code)
	}
}

func TestRemoveWorkspaceLeavesFiles(t *testing.T) {
	srv := newWorkspaceServer(t, t.TempDir())
	target := filepath.Join(t.TempDir(), "doomed")
	if rec := doRequest(t, srv, http.MethodPost, "/api/workspaces", `{"name":"doomed","path":"`+target+`"}`); rec.Code != http.StatusOK {
		t.Fatalf("create: code %d (%s)", rec.Code, rec.Body.String())
	}

	rec := doRequest(t, srv, http.MethodDelete, "/api/workspaces/doomed", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("delete: code %d (%s)", rec.Code, rec.Body.String())
	}
	got := decodeWorkspaces(t, rec.Body)
	for _, w := range got.Workspaces {
		if w.Name == "doomed" {
			t.Error("workspace still listed after delete")
		}
	}
	// Files remain on disk.
	if !project.IsInitialized(project.PathsFor(target)) {
		t.Error("delete removed the project files, want them left in place")
	}
}

func TestBrowseDirsListsSubdirsAndFlagsProjects(t *testing.T) {
	srv := newWorkspaceServer(t, t.TempDir())

	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, "plain"), 0o755); err != nil {
		t.Fatal(err)
	}
	proj := filepath.Join(base, "proj")
	if err := project.Initialize(project.PathsFor(proj), service.DefaultRegistry()); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	// A file should not appear in the listing.
	if err := os.WriteFile(filepath.Join(base, "file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	rec := doRequest(t, srv, http.MethodGet, "/api/workspaces/browse?path="+base, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	var got dirListing
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Path != base || got.Parent != filepath.Dir(base) {
		t.Errorf("path/parent = %q/%q", got.Path, got.Parent)
	}
	names := map[string]bool{}
	projFlag := map[string]bool{}
	for _, e := range got.Entries {
		names[e.Name] = true
		projFlag[e.Name] = e.IsProject
	}
	if !names["plain"] || !names["proj"] {
		t.Errorf("entries missing dirs: %+v", got.Entries)
	}
	if names["file.txt"] {
		t.Error("listing included a file")
	}
	if projFlag["plain"] || !projFlag["proj"] {
		t.Errorf("isProject flags wrong: plain=%v proj=%v", projFlag["plain"], projFlag["proj"])
	}
}

func TestBrowseDirsRejectsMissingPath(t *testing.T) {
	srv := newWorkspaceServer(t, t.TempDir())
	rec := doRequest(t, srv, http.MethodGet, "/api/workspaces/browse?path="+filepath.Join(t.TempDir(), "nope"), "")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("code = %d, want 400", rec.Code)
	}
}
