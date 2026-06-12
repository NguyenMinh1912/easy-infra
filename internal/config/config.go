// Package config loads, validates, and saves the user-authored project config
// (easy-infra.yml). The project config is a thin project marker: it records the
// schema version and exists so the tool can tell an initialized folder from an
// uninitialized one.
//
// Services no longer live here. Each profile owns its own services and their
// full config — definition and environment in one block (see package profile).
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// DefaultPath is the conventional project-config filename within a project.
const DefaultPath = "easy-infra.yml"

// CurrentVersion is the config schema version this build understands.
const CurrentVersion = 1

// Config is the top-level project-config document. It marks a folder as an
// easy-infra project and pins the schema version; profiles hold the services.
type Config struct {
	Version int `yaml:"version"`
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

// Validate checks the config's structure: the schema version must be one this
// build understands.
func (c *Config) Validate() error {
	if c.Version != CurrentVersion {
		return fmt.Errorf("unsupported config version %d (expected %d)", c.Version, CurrentVersion)
	}
	return nil
}
