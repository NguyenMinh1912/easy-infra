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

// Profile is one environment's settings: per-service environment config keyed
// by service name. The map is inlined so a profile file reads as a flat
// mapping of service name to its settings.
type Profile struct {
	Services map[string]service.Config `yaml:",inline"`
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
		p.Services = map[string]service.Config{}
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
	data, err := yaml.Marshal(p)
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

// Validate checks the profile against the services the project defines: every
// defined service must have an environment block here, every block must
// validate, and no block may reference an undefined service.
func (p *Profile) Validate(reg *service.Registry, definedServices []string) error {
	defined := make(map[string]bool, len(definedServices))
	for _, name := range definedServices {
		defined[name] = true
	}

	for name := range p.Services {
		if !defined[name] {
			return fmt.Errorf("configures service %q which the project does not define", name)
		}
	}

	for _, name := range definedServices {
		envCfg, ok := p.Services[name]
		if !ok {
			return fmt.Errorf("missing environment config for service %q", name)
		}
		svc, ok := reg.Get(name)
		if !ok {
			return fmt.Errorf("unknown service %q", name)
		}
		if err := svc.ValidateEnv(envCfg); err != nil {
			return fmt.Errorf("service %q: %w", name, err)
		}
	}
	return nil
}
