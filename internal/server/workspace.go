package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/minhnc/easy-infra/internal/project"
	"github.com/minhnc/easy-infra/internal/workspace"
)

// workspaceInfo is one entry in the workspaces list. Exists is false when the
// folder has been moved or deleted under the tool, so the UI can flag it.
type workspaceInfo struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
}

// workspacesResponse is the shape returned by the workspace endpoints. home and
// separator seed the create dialog's folder browser and keep the UI from
// hard-coding a path separator.
type workspacesResponse struct {
	Active     string          `json:"active"`
	Workspaces []workspaceInfo `json:"workspaces"`
	Home       string          `json:"home"`
	Separator  string          `json:"separator"`
}

// workspaces builds the current workspaces payload under the read lock.
func (s *Server) workspaces() workspacesResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()
	home, _ := os.UserHomeDir()
	list := make([]workspaceInfo, 0, len(s.ws.Workspaces))
	for _, w := range s.ws.Workspaces {
		list = append(list, workspaceInfo{Name: w.Name, Path: w.Path, Exists: isDir(w.Path)})
	}
	return workspacesResponse{
		Active:     s.ws.Active,
		Workspaces: list,
		Home:       home,
		Separator:  string(os.PathSeparator),
	}
}

func (s *Server) handleListWorkspaces(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.workspaces())
}

// handleCreateWorkspace registers a new workspace, scaffolding a project in the
// target folder when one is not already there, then makes it active. An empty
// folder is created if missing; an already-initialized folder is adopted as-is.
func (s *Server) handleCreateWorkspace(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	name := strings.TrimSpace(body.Name)
	dir := strings.TrimSpace(body.Path)
	if name == "" || dir == "" {
		writeError(w, http.StatusBadRequest, "name and path are required")
		return
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	abs = filepath.Clean(abs)
	if err := os.MkdirAll(abs, 0o755); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	paths := project.PathsFor(abs)
	if !project.IsInitialized(paths) {
		if err := project.Initialize(paths, s.reg); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	s.mu.Lock()
	if err := s.ws.Add(name, abs); err != nil {
		s.mu.Unlock()
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	_ = s.ws.SetActive(name)
	saveErr := workspace.Save(s.ws)
	s.mu.Unlock()
	if saveErr != nil {
		writeError(w, http.StatusInternalServerError, saveErr.Error())
		return
	}
	writeJSON(w, http.StatusOK, s.workspaces())
}

// handleActivateWorkspace switches the active workspace.
func (s *Server) handleActivateWorkspace(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	s.mu.Lock()
	if err := s.ws.SetActive(strings.TrimSpace(body.Name)); err != nil {
		s.mu.Unlock()
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	saveErr := workspace.Save(s.ws)
	s.mu.Unlock()
	if saveErr != nil {
		writeError(w, http.StatusInternalServerError, saveErr.Error())
		return
	}
	writeJSON(w, http.StatusOK, s.workspaces())
}

// handleRemoveWorkspace drops a workspace from the registry. Files on disk are
// left untouched.
func (s *Server) handleRemoveWorkspace(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	s.mu.Lock()
	if err := s.ws.Remove(name); err != nil {
		s.mu.Unlock()
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	saveErr := workspace.Save(s.ws)
	s.mu.Unlock()
	if saveErr != nil {
		writeError(w, http.StatusInternalServerError, saveErr.Error())
		return
	}
	writeJSON(w, http.StatusOK, s.workspaces())
}

// dirEntry is a single subdirectory in a browse listing. IsProject marks folders
// that already hold an easy-infra project.
type dirEntry struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	IsProject bool   `json:"isProject"`
}

// dirListing is the shape returned by the directory browser. parent is empty at
// the filesystem root.
type dirListing struct {
	Path    string     `json:"path"`
	Parent  string     `json:"parent"`
	Entries []dirEntry `json:"entries"`
}

// handleBrowseDirs lists the subdirectories of a folder so the UI can navigate
// the server's filesystem when choosing where a new workspace lives. It returns
// directories only — never file contents — and skips entries it cannot read.
func (s *Server) handleBrowseDirs(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSpace(r.URL.Query().Get("path"))
	if path == "" {
		path, _ = os.UserHomeDir()
	}
	path = filepath.Clean(path)

	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		writeError(w, http.StatusBadRequest, "not a directory: "+path)
		return
	}
	// os.ReadDir returns entries sorted by name.
	read, err := os.ReadDir(path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	entries := make([]dirEntry, 0, len(read))
	for _, e := range read {
		if !e.IsDir() {
			continue
		}
		child := filepath.Join(path, e.Name())
		entries = append(entries, dirEntry{
			Name:      e.Name(),
			Path:      child,
			IsProject: project.IsInitialized(project.PathsFor(child)),
		})
	}
	parent := filepath.Dir(path)
	if parent == path {
		parent = ""
	}
	writeJSON(w, http.StatusOK, dirListing{Path: path, Parent: parent, Entries: entries})
}

// isDir reports whether path is an existing directory.
func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
