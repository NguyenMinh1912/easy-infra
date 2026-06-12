// Package profile loads, validates, and saves per-profile environment configs.
//
// A profile describes how to reach each service in a particular environment —
// host, port, user, password, database URL — for the services the project
// defines (see package config). Each profile is its own file under
// .easy-infra/profiles/<name>.yml. These files hold credentials and are
// tool-managed/local, so they are gitignored along with the rest of
// .easy-infra/.
package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/minhnc/easy-infra/internal/service"
	"gopkg.in/yaml.v3"
)

// DefaultDir is the conventional directory holding profile env files.
var DefaultDir = filepath.Join(".easy-infra", "profiles")

// ext is the profile file extension.
const ext = ".yml"

// ServiceEntry is one service instance within a profile. A profile may hold
// several instances of the same service type (e.g. two postgres databases), so
// each entry is identified by its own unique id (the map key) rather than by
// the service type. The entry records:
//
//   - Type — which service this is (the registry key, e.g. "postgres"). When
//     empty it defaults to the entry's id, so a profile written before instance
//     ids existed (keyed directly by service type) still loads unchanged.
//   - Name — a user-facing label that can be renamed freely. When empty it
//     defaults to the id.
//   - Config — the merged definition + environment block (host, port, version,
//     …), inlined so an entry reads as a flat mapping.
//
// Type and Name are reserved keys within an entry; a service's own config must
// not use them.
type ServiceEntry struct {
	Type   string         `yaml:"type,omitempty"`
	Name   string         `yaml:"name,omitempty"`
	Config service.Config `yaml:",inline"`
}

// ResolveType returns the entry's service type, falling back to id when the
// Type field is empty (the backward-compatible case where the key is the type).
func (e ServiceEntry) ResolveType(id string) string {
	if e.Type != "" {
		return e.Type
	}
	return id
}

// ResolveName returns the entry's display name, falling back to id when the
// Name field is empty.
func (e ServiceEntry) ResolveName(id string) string {
	if e.Name != "" {
		return e.Name
	}
	return id
}

// Profile is one environment's settings: a set of service instances keyed by a
// unique id. The map is inlined so a profile file reads as a flat mapping of
// service id to its settings.
type Profile struct {
	Services map[string]ServiceEntry `yaml:",inline"`
}

// Path returns the file path for the named profile within dir.
func Path(dir, name string) string {
	return filepath.Join(dir, name+ext)
}

// Load reads and parses the profile at path. It does not validate; call
// Validate separately so callers control the registry and defined services.
func Load(path string) (*Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Profile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing profile %s: %w", path, err)
	}
	if p.Services == nil {
		p.Services = map[string]ServiceEntry{}
	}
	// An entry with no config keys unmarshals to a nil inline map; normalise it
	// to an empty map so callers can always read (and overlay onto) Config.
	for id, entry := range p.Services {
		if entry.Config == nil {
			entry.Config = service.Config{}
			p.Services[id] = entry
		}
	}
	return &p, nil
}

// Save writes the profile to path as YAML, creating the parent directory if
// needed.
func (p *Profile) Save(path string) error {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating profiles dir %s: %w", dir, err)
		}
	}
	// Drop a Type/Name that merely echoes the id, so an entry whose id already
	// is its type (the common single-instance case) serialises as a plain
	// `type:`-less block — identical to profiles written before instance ids.
	out := make(map[string]ServiceEntry, len(p.Services))
	for id, e := range p.Services {
		if e.Type == id {
			e.Type = ""
		}
		if e.Name == id {
			e.Name = ""
		}
		out[id] = e
	}
	data, err := yaml.Marshal(Profile{Services: out})
	if err != nil {
		return fmt.Errorf("encoding profile: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing profile %s: %w", path, err)
	}
	return nil
}

// Remove deletes the profile file at path. A missing file yields an
// fs.ErrNotExist error so callers can surface an actionable message.
func Remove(path string) error {
	return os.Remove(path)
}

// List returns the names of the profiles defined in dir, sorted. A missing
// directory yields an empty list rather than an error.
func List(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading profiles dir %s: %w", dir, err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ext) {
			continue
		}
		names = append(names, strings.TrimSuffix(e.Name(), ext))
	}
	sort.Strings(names)
	return names, nil
}

// Validate checks the profile owns a coherent set of services: it must define
// at least one, every service must be one the registry knows, and every block
// must pass the service's config validation (definition and environment).
func (p *Profile) Validate(reg *service.Registry) error {
	if len(p.Services) == 0 {
		return fmt.Errorf("profile must define at least one service")
	}
	for id, entry := range p.Services {
		svcType := entry.ResolveType(id)
		svc, ok := reg.Get(svcType)
		if !ok {
			return fmt.Errorf("unknown service %q", svcType)
		}
		if err := service.ValidateConfig(svc, entry.Config); err != nil {
			return fmt.Errorf("service %q: %w", id, err)
		}
	}
	return nil
}
