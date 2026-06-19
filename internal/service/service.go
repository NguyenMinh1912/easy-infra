// Package service defines the common abstraction for the infrastructure
// services easy-infra can manage (postgres, minio, redis, localstack) and a
// registry for looking them up by name.
//
// Adding a new service is intentionally a closed-for-modification operation:
// implement the Service interface in its own file and register it in
// DefaultRegistry. No other part of the codebase should switch on service
// names.
package service

import (
	"context"
	"fmt"
	"sort"
)

// Config is a raw, per-service configuration block. Each Service implementation
// decodes and validates it into its own typed shape, so the rest of the
// codebase can stay service-agnostic.
//
// A service's config has two logical halves, stored together as one profile
// service block (a JSON column in the store):
//
//   - definition — what the service is: image/version and other settings that do
//     not vary by where it runs (e.g. `version`, `cleanable`);
//   - environment — how to reach it: host, port, credentials, database URL.
//
// Each Service still owns the schema for each half (DefaultDefinition/ValidateDefinition
// and DefaultEnv/ValidateEnv); DefaultConfig and ValidateConfig fold the two
// halves together so callers can treat a profile block as a single unit.
type Config map[string]any

// DefaultConfig returns a service's starter config: its default definition and
// default environment merged into one block, the shape a profile stores.
func DefaultConfig(svc Service) Config {
	cfg := Config{}
	for k, v := range svc.DefaultDefinition() {
		cfg[k] = v
	}
	for k, v := range svc.DefaultEnv() {
		cfg[k] = v
	}
	return cfg
}

// ValidateConfig validates a profile's merged service block against both halves
// of the service's schema. The field helpers read keys by name and ignore
// unknown ones, so the definition rules see only the definition keys and the
// environment rules only the environment keys, even though they share one map.
func ValidateConfig(svc Service, cfg Config) error {
	if err := svc.ValidateDefinition(cfg); err != nil {
		return err
	}
	return svc.ValidateEnv(cfg)
}

// cleanableKey is the definition flag controlling whether a service may be
// cleaned. It is a common, service-agnostic field: any service's definition may
// set `cleanable: false` to protect its data from Clean. When absent, a service
// is cleanable, preserving the prior behaviour.
const cleanableKey = "cleanable"

// cleanable reports whether a service definition permits Clean. A service is
// cleanable unless its definition explicitly sets `cleanable: false`.
func cleanable(def Config) (bool, error) {
	return optionalBool(def, cleanableKey, true)
}

// validateCleanable checks the common `cleanable` definition flag. Each
// service's ValidateDefinition calls it so the flag is validated uniformly,
// rather than every service re-implementing the same rule.
func validateCleanable(def Config) error {
	_, err := cleanable(def)
	return err
}

// Service is the common interface every supported service implements. Each
// service owns the schema for both halves of its config — the project-level
// definition and the per-profile environment — keeping that knowledge in one
// place rather than scattered across the codebase. It also owns its lifecycle:
// how to bring itself up, check its health, back it up, and tear it down.
type Service interface {
	// Name is the stable identifier used in config and state, e.g. "postgres".
	Name() string

	// DefaultDefinition returns a starter definition (project-level) config,
	// used when scaffolding the project config.
	DefaultDefinition() Config

	// ValidateDefinition checks a service's project-level definition config and
	// returns an actionable error if it is invalid.
	ValidateDefinition(cfg Config) error

	// DefaultEnv returns a starter environment (profile-level) config, used when
	// scaffolding a profile.
	DefaultEnv() Config

	// ValidateEnv checks a service's per-profile environment config and returns
	// an actionable error if it is invalid.
	ValidateEnv(cfg Config) error

	// Apply reconciles the service so it matches spec: provisioning and starting
	// it if absent, and bringing a running instance into line otherwise. It is
	// the per-service half of `easy-infra apply`.
	Apply(ctx context.Context, spec Spec) error

	// Health reports whether the running service described by spec is reachable
	// and ready. It returns nil when healthy and an actionable error otherwise.
	Health(ctx context.Context, spec Spec) error

	// Backup captures the service's current data for the instance described by
	// spec. It is the per-service half of `easy-infra backup`.
	Backup(ctx context.Context, spec Spec) error

	// Clean tears the service down and removes its data, returning it to a clean
	// (un-provisioned) state.
	Clean(ctx context.Context, spec Spec) error
}

// Registry holds the set of known services keyed by name. It is the single
// extension point for the system: commands and config validation discover
// services through a Registry rather than referencing concrete types.
type Registry struct {
	services map[string]Service
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{services: make(map[string]Service)}
}

// Register adds a service to the registry. It returns an error if a service
// with the same name is already registered, guarding against accidental
// double-registration.
func (r *Registry) Register(s Service) error {
	if s == nil {
		return fmt.Errorf("service: cannot register a nil service")
	}
	name := s.Name()
	if name == "" {
		return fmt.Errorf("service: cannot register a service with an empty name")
	}
	if _, exists := r.services[name]; exists {
		return fmt.Errorf("service: %q is already registered", name)
	}
	r.services[name] = s
	return nil
}

// Get returns the service registered under name and whether it was found.
func (r *Registry) Get(name string) (Service, bool) {
	s, ok := r.services[name]
	return s, ok
}

// Names returns the registered service names in sorted order, giving callers
// deterministic output.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.services))
	for name := range r.services {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// DefaultRegistry returns a Registry populated with every service easy-infra
// supports out of the box. This is the one place that enumerates the built-in
// services.
func DefaultRegistry() *Registry {
	r := NewRegistry()
	for _, s := range []Service{
		Postgres{},
		MinIO{},
		Redis{},
		LocalStack{},
	} {
		// Built-in services are known-good, so a registration failure is a
		// programming error rather than a runtime condition.
		if err := r.Register(s); err != nil {
			panic(fmt.Sprintf("service: registering built-in %q: %v", s.Name(), err))
		}
	}
	return r
}
