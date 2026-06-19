package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/minhnc/easy-infra/internal/project"
	"github.com/minhnc/easy-infra/internal/store"
)

// workspaceInfo is one entry in the workspaces list.
type workspaceInfo struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// workspacesResponse is the shape returned by the workspace endpoints. Active is
// the active workspace id, or 0 when none is set.
type workspacesResponse struct {
	Active     int64           `json:"active"`
	Workspaces []workspaceInfo `json:"workspaces"`
}

// workspaces builds the current workspaces payload from the store.
func (s *Server) workspaces() (workspacesResponse, error) {
	list, err := s.store.ListWorkspaces()
	if err != nil {
		return workspacesResponse{}, err
	}
	infos := make([]workspaceInfo, 0, len(list))
	for _, w := range list {
		infos = append(infos, workspaceInfo{ID: w.ID, Name: w.Name})
	}
	var active int64
	if w, ok, err := s.store.ActiveWorkspace(); err != nil {
		return workspacesResponse{}, err
	} else if ok {
		active = w.ID
	}
	return workspacesResponse{Active: active, Workspaces: infos}, nil
}

// writeWorkspaces writes the current workspaces payload, or a 500 on failure.
func (s *Server) writeWorkspaces(w http.ResponseWriter, status int) {
	resp, err := s.workspaces()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, status, resp)
}

func (s *Server) handleListWorkspaces(w http.ResponseWriter, _ *http.Request) {
	s.writeWorkspaces(w, http.StatusOK)
}

// handleCreateWorkspace creates a workspace (scaffolding its default profile)
// and makes it active.
func (s *Server) handleCreateWorkspace(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	ws, err := project.CreateWorkspace(s.store, s.reg, name)
	if err != nil {
		if errors.Is(err, store.ErrWorkspaceExists) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.store.SetActiveWorkspace(ws.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.writeWorkspaces(w, http.StatusOK)
}

// handleRenameWorkspace renames a workspace identified by its id.
func (s *Server) handleRenameWorkspace(w http.ResponseWriter, r *http.Request) {
	id, ok := workspaceID(w, r)
	if !ok {
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if err := s.store.RenameWorkspace(id, name); err != nil {
		s.writeWorkspaceError(w, err)
		return
	}
	s.writeWorkspaces(w, http.StatusOK)
}

// handleActivateWorkspace switches the active workspace.
func (s *Server) handleActivateWorkspace(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := s.store.SetActiveWorkspace(body.ID); err != nil {
		s.writeWorkspaceError(w, err)
		return
	}
	s.writeWorkspaces(w, http.StatusOK)
}

// handleRemoveWorkspace drops a workspace and its profiles/services.
func (s *Server) handleRemoveWorkspace(w http.ResponseWriter, r *http.Request) {
	id, ok := workspaceID(w, r)
	if !ok {
		return
	}
	if err := s.store.RemoveWorkspace(id); err != nil {
		s.writeWorkspaceError(w, err)
		return
	}
	s.writeWorkspaces(w, http.StatusOK)
}

// workspaceID parses the {id} path value, writing a 400 and returning ok=false
// when it is not a valid integer.
func workspaceID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid workspace id")
		return 0, false
	}
	return id, true
}

// writeWorkspaceError maps store errors onto HTTP responses.
func (s *Server) writeWorkspaceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrWorkspaceNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, store.ErrWorkspaceExists):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}
