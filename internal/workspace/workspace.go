// Package workspace manages the tool-owned registry of project folders the web
// UI knows about. A workspace is a named easy-infra project folder; the registry
// records the known workspaces and which one is active, so the user can create
// and switch between them from the web app rather than being tied to the folder
// `easy-infra ui` was launched from.
//
// The registry is user-global (not per-project): it lives under the OS user
// config directory because it must exist before any workspace is chosen. Like
// state.json it is tool-owned JSON and not meant to be hand-edited.
package workspace

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Workspace is one known project folder.
type Workspace struct {
	Name string `json:"name"`
	Path string `json:"path"` // absolute
}

// Registry is the persisted set of known workspaces plus the active one.
type Registry struct {
	Active     string      `json:"active"` // name of the active workspace
	Workspaces []Workspace `json:"workspaces"`
}

// userConfigDir is a seam so tests can redirect the registry location away from
// the real user config directory.
var userConfigDir = os.UserConfigDir

// configDirEnv lets the user (and tests) relocate the registry away from the OS
// user config directory.
const configDirEnv = "EASY_INFRA_CONFIG_DIR"

// registryPath returns the on-disk location of the workspace registry.
func registryPath() (string, error) {
	if dir := os.Getenv(configDirEnv); dir != "" {
		return filepath.Join(dir, "workspaces.json"), nil
	}
	dir, err := userConfigDir()
	if err != nil {
		return "", fmt.Errorf("locating user config dir: %w", err)
	}
	return filepath.Join(dir, "easy-infra", "workspaces.json"), nil
}

// Load reads the registry. A missing file yields an empty registry rather than
// an error, so a first run starts clean.
func Load() (*Registry, error) {
	path, err := registryPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &Registry{}, nil
		}
		return nil, fmt.Errorf("reading workspaces %s: %w", path, err)
	}
	var r Registry
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parsing workspaces %s: %w", path, err)
	}
	return &r, nil
}

// Save writes the registry, creating the config directory if needed. JSON is
// indented so the file stays readable even though it is tool-owned.
func Save(r *Registry) error {
	path, err := registryPath()
	if err != nil {
		return err
	}
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating config dir %s: %w", dir, err)
		}
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding workspaces: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing workspaces %s: %w", path, err)
	}
	return nil
}

// Add registers a new workspace. The path is resolved to an absolute path. It
// rejects a duplicate name or a path already registered under another name.
func (r *Registry) Add(name, path string) error {
	if name == "" {
		return fmt.Errorf("workspace name is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolving path %s: %w", path, err)
	}
	for _, w := range r.Workspaces {
		if w.Name == name {
			return fmt.Errorf("workspace %q already exists", name)
		}
		if w.Path == abs {
			return fmt.Errorf("folder %s is already registered as workspace %q", abs, w.Name)
		}
	}
	r.Workspaces = append(r.Workspaces, Workspace{Name: name, Path: abs})
	return nil
}

// SetActive marks the named workspace active. It errors if the name is unknown.
func (r *Registry) SetActive(name string) error {
	for _, w := range r.Workspaces {
		if w.Name == name {
			r.Active = name
			return nil
		}
	}
	return fmt.Errorf("workspace %q does not exist", name)
}

// Remove drops the named workspace from the registry. It does not touch any
// files on disk. If the removed workspace was active, the first remaining
// workspace becomes active (or none, if the registry is now empty).
func (r *Registry) Remove(name string) error {
	idx := -1
	for i, w := range r.Workspaces {
		if w.Name == name {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("workspace %q does not exist", name)
	}
	r.Workspaces = append(r.Workspaces[:idx], r.Workspaces[idx+1:]...)
	if r.Active == name {
		r.Active = ""
		if len(r.Workspaces) > 0 {
			r.Active = r.Workspaces[0].Name
		}
	}
	return nil
}

// ActivePath returns the active workspace's folder path. ok is false when there
// is no active workspace (empty registry or a dangling active name).
func (r *Registry) ActivePath() (string, bool) {
	if r.Active == "" {
		return "", false
	}
	for _, w := range r.Workspaces {
		if w.Name == r.Active {
			return w.Path, true
		}
	}
	return "", false
}
