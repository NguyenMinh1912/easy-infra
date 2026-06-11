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
	"fmt"
	"sort"
)

// Config is the raw, per-service configuration block taken from a profile in
// the YAML config. Each Service implementation decodes and validates it into
// its own typed shape, so the rest of the codebase can stay service-agnostic.
type Config map[string]any

// Service is the common interface every supported service implements.
//
// Keeping the surface small (Interface Segregation) lets callers depend only
// on the behaviour they need, and lets new services satisfy the contract
// without inheriting unrelated concerns.
type Service interface {
	// Name is the stable identifier used in config and state, e.g. "postgres".
	Name() string

	// DefaultConfig returns a sensible starter configuration, used when
	// scaffolding a new profile.
	DefaultConfig() Config

	// Validate checks a profile's configuration block for this service and
	// returns an actionable error if it is invalid.
	Validate(cfg Config) error
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
