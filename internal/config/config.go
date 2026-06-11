// Package config loads, validates, and saves the user-authored YAML config
// (easy-infra.yml). The config is the source of truth for a project's profiles
// and the per-service configuration within each profile.
//
// This package owns only the shape and integrity of the config. Service-level
// validation is delegated to a service.Registry, so config stays agnostic of
// which services exist (Dependency Inversion).
package config

import (
	"fmt"
	"os"
	"sort"

	"github.com/minhnc/easy-infra/internal/service"
	"gopkg.in/yaml.v3"
)

// DefaultPath is the conventional config filename within a project folder.
const DefaultPath = "easy-infra.yml"

// CurrentVersion is the config schema version this build understands.
const CurrentVersion = 1

// Config is the top-level YAML document.
type Config struct {
	Version  int                `yaml:"version"`
	Profiles map[string]Profile `yaml:"profiles"`
}

// Profile is a named bundle of service configurations.
type Profile struct {
	Services map[string]service.Config `yaml:"services"`
}

// Load reads and parses the config at path. It does not validate; call
// Validate separately so callers control which registry is used.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}
	return &cfg, nil
}

// Save writes the config to path as YAML.
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing config %s: %w", path, err)
	}
	return nil
}

// Validate checks the config's structure and the configuration of every
// service in every profile, using reg to resolve service-specific rules.
func (c *Config) Validate(reg *service.Registry) error {
	if c.Version != CurrentVersion {
		return fmt.Errorf("unsupported config version %d (expected %d)", c.Version, CurrentVersion)
	}
	if len(c.Profiles) == 0 {
		return fmt.Errorf("config must define at least one profile")
	}
	for _, name := range c.ProfileNames() {
		profile := c.Profiles[name]
		if len(profile.Services) == 0 {
			return fmt.Errorf("profile %q defines no services", name)
		}
		for svcName, svcCfg := range profile.Services {
			svc, ok := reg.Get(svcName)
			if !ok {
				return fmt.Errorf("profile %q references unknown service %q", name, svcName)
			}
			if err := svc.Validate(svcCfg); err != nil {
				return fmt.Errorf("profile %q, service %q: %w", name, svcName, err)
			}
		}
	}
	return nil
}

// Profile returns the named profile and whether it exists.
func (c *Config) Profile(name string) (Profile, bool) {
	p, ok := c.Profiles[name]
	return p, ok
}

// ProfileNames returns the profile names in sorted order.
func (c *Config) ProfileNames() []string {
	names := make([]string, 0, len(c.Profiles))
	for name := range c.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
