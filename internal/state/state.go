// Package state manages the tool-owned JSON state (.easy-infra/state.json).
// State records runtime/derived facts — most importantly the active profile —
// and is never meant to be hand-edited by users.
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Dir is the conventional directory holding tool-owned state.
const Dir = ".easy-infra"

// DefaultPath is the conventional state file path within a project folder.
var DefaultPath = filepath.Join(Dir, "state.json")

// State is the tool-owned runtime state for a project.
type State struct {
	// ActiveProfile is the profile that commands operate on by default.
	ActiveProfile string `json:"activeProfile"`
}

// Load reads the state at path. A missing file is reported via os.IsNotExist on
// the returned error so callers can distinguish "not initialized" from a read
// failure.
func Load(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing state %s: %w", path, err)
	}
	return &s, nil
}

// Save writes the state to path, creating the parent directory if needed.
// JSON is written indented so the file stays diff-friendly even though it is
// tool-owned.
func (s *State) Save(path string) error {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating state dir %s: %w", dir, err)
		}
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding state: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing state %s: %w", path, err)
	}
	return nil
}

// Exists reports whether a state file is present at path.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || !errors.Is(err, fs.ErrNotExist)
}
