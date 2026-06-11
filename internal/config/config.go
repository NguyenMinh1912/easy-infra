// Package config loads, validates, and saves the user-authored project config
// (easy-infra.yml). The project config is the source of truth for which
// services a project uses and their environment-independent definitions
// (image/version and the like).
//
// Per-environment settings — host, port, credentials, database URLs — do not
// live here; they belong to profiles (see package profile). This package only
// knows about service definitions, and delegates their validation to a
// service.Registry so it stays agnostic of which services exist.
package config

import (
	"fmt"
	"os"
	"sort"

	"github.com/minhnc/easy-infra/internal/service"
	"gopkg.in/yaml.v3"
)

// DefaultPath is the conventional project-config filename within a project.
const DefaultPath = "easy-infra.yml"

// CurrentVersion is the config schema version this build understands.
const CurrentVersion = 1

// Config is the top-level project-config document: the set of service
// definitions a project uses.
type Config struct {
	Version  int                       `yaml:"version"`
	Services map[string]service.Config `yaml:"services"`
}

// Load reads and parses the project config at path. It does not validate; call
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

// Save writes the project config to path as YAML.
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

// Validate checks the config's structure and the definition of every service,
// using reg to resolve service-specific rules.
func (c *Config) Validate(reg *service.Registry) error {
	if c.Version != CurrentVersion {
		return fmt.Errorf("unsupported config version %d (expected %d)", c.Version, CurrentVersion)
	}
	if len(c.Services) == 0 {
		return fmt.Errorf("config must define at least one service")
	}
	for _, name := range c.ServiceNames() {
		svc, ok := reg.Get(name)
		if !ok {
			return fmt.Errorf("unknown service %q", name)
		}
		if err := svc.ValidateDefinition(c.Services[name]); err != nil {
			return fmt.Errorf("service %q: %w", name, err)
		}
	}
	return nil
}

// HasService reports whether the project defines the named service.
func (c *Config) HasService(name string) bool {
	_, ok := c.Services[name]
	return ok
}

// ServiceNames returns the defined service names in sorted order.
func (c *Config) ServiceNames() []string {
	names := make([]string, 0, len(c.Services))
	for name := range c.Services {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
