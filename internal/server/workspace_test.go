package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/minhnc/easy-infra/internal/service"
)

func decodeWorkspaces(t *testing.T, rec interface{ Bytes() []byte }) workspacesResponse {
	t.Helper()
	var got workspacesResponse
	if err := json.Unmarshal(rec.Bytes(), &got); err != nil {
		t.Fatalf("decode workspaces: %v", err)
	}
	return got
}

// newWorkspaceServer returns a server over an empty store (no workspaces yet),
// so workspace tests drive creation through the API.
func newWorkspaceServer(t *testing.T) *Server {
	t.Helper()
	return New(service.DefaultRegistry(), newStore(t), emptyUI)
}

func TestCreateAndListWorkspaces(t *testing.T) {
	srv := newWorkspaceServer(t)

	rec := doRequest(t, srv, http.MethodPost, "/api/workspaces", `{"name":"app"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("create: code %d (%s)", rec.Code, rec.Body.String())
	}
	got := decodeWorkspaces(t, rec.Body)
	if len(got.Workspaces) != 1 || got.Workspaces[0].Name != "app" {
		t.Fatalf("unexpected list: %+v", got)
	}
	// The new workspace is active but starts with no profiles; the user
	// creates their own.
	if got.Active != got.Workspaces[0].ID {
		t.Errorf("Active = %d, want %d", got.Active, got.Workspaces[0].ID)
	}
	_, status := getStatus(t, srv)
	if !status.Initialized || status.ActiveProfile != "" {
		t.Errorf("status = %+v, want initialized with no active profile", status)
	}
	if len(status.Profiles) != 0 {
		t.Errorf("Profiles = %+v, want none", status.Profiles)
	}
}

func TestCreateWorkspaceDuplicate(t *testing.T) {
	srv := newWorkspaceServer(t)
	if rec := doRequest(t, srv, http.MethodPost, "/api/workspaces", `{"name":"app"}`); rec.Code != http.StatusOK {
		t.Fatalf("create: code %d", rec.Code)
	}
	rec := doRequest(t, srv, http.MethodPost, "/api/workspaces", `{"name":"app"}`)
	if rec.Code != http.StatusConflict {
		t.Errorf("code = %d, want 409", rec.Code)
	}
}

func TestRenameWorkspace(t *testing.T) {
	srv := newWorkspaceServer(t)
	rec := doRequest(t, srv, http.MethodPost, "/api/workspaces", `{"name":"app"}`)
	id := decodeWorkspaces(t, rec.Body).Workspaces[0].ID

	rec = doRequest(t, srv, http.MethodPut, fmt.Sprintf("/api/workspaces/%d", id), `{"name":"renamed"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("rename: code %d (%s)", rec.Code, rec.Body.String())
	}
	got := decodeWorkspaces(t, rec.Body)
	if len(got.Workspaces) != 1 || got.Workspaces[0].Name != "renamed" {
		t.Errorf("after rename: %+v", got)
	}
	// Unknown id.
	if rec := doRequest(t, srv, http.MethodPut, "/api/workspaces/9999", `{"name":"x"}`); rec.Code != http.StatusNotFound {
		t.Errorf("rename unknown: code %d, want 404", rec.Code)
	}
}

func TestActivateWorkspaceSwitches(t *testing.T) {
	srv := newWorkspaceServer(t)
	a := decodeWorkspaces(t, doRequest(t, srv, http.MethodPost, "/api/workspaces", `{"name":"a"}`).Body)
	aID := a.Workspaces[0].ID
	b := decodeWorkspaces(t, doRequest(t, srv, http.MethodPost, "/api/workspaces", `{"name":"b"}`).Body)
	// Creating "b" made it active; switch back to "a".
	if b.Active == aID {
		t.Fatalf("expected b active after creating it, got %d", b.Active)
	}

	rec := doRequest(t, srv, http.MethodPost, "/api/workspaces/activate", fmt.Sprintf(`{"id":%d}`, aID))
	if rec.Code != http.StatusOK {
		t.Fatalf("activate: code %d (%s)", rec.Code, rec.Body.String())
	}
	if got := decodeWorkspaces(t, rec.Body); got.Active != aID {
		t.Errorf("Active = %d, want %d", got.Active, aID)
	}
}

func TestActivateUnknownWorkspace(t *testing.T) {
	srv := newWorkspaceServer(t)
	rec := doRequest(t, srv, http.MethodPost, "/api/workspaces/activate", `{"id":9999}`)
	if rec.Code != http.StatusNotFound {
		t.Errorf("code = %d, want 404", rec.Code)
	}
}

func TestRemoveWorkspace(t *testing.T) {
	srv := newWorkspaceServer(t)
	id := decodeWorkspaces(t, doRequest(t, srv, http.MethodPost, "/api/workspaces", `{"name":"doomed"}`).Body).Workspaces[0].ID

	rec := doRequest(t, srv, http.MethodDelete, fmt.Sprintf("/api/workspaces/%d", id), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("delete: code %d (%s)", rec.Code, rec.Body.String())
	}
	if got := decodeWorkspaces(t, rec.Body); len(got.Workspaces) != 0 {
		t.Errorf("workspace still listed after delete: %+v", got)
	}
	if rec := doRequest(t, srv, http.MethodDelete, fmt.Sprintf("/api/workspaces/%d", id), ""); rec.Code != http.StatusNotFound {
		t.Errorf("delete again: code %d, want 404", rec.Code)
	}
}
