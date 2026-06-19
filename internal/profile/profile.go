// Package profile defines a profile's in-memory shape and validation.
//
// A profile describes how to reach each service in a particular environment —
// host, port, user, password, database URL — for the services it owns. Profiles
// are persisted by the store package (one row per service instance in the
// central SQLite database); this package holds only the value types and the
// validation rules, so the storage layer and the command/HTTP layers share one
// definition of what a profile is.
package profile

import (
	"fmt"

	"github.com/minhnc/easy-infra/internal/service"
)

// ServiceEntry is one service instance within a profile. A profile may hold
// several instances of the same service type (e.g. two postgres databases), so
// each entry is identified by its own unique id (the map key) rather than by
// the service type. The entry records:
//
//   - Type — which service this is (the registry key, e.g. "postgres"). When
//     empty it defaults to the entry's id, so a single-instance profile keyed
//     directly by service type needs no explicit type.
//   - Name — a user-facing label that can be renamed freely. When empty it
//     defaults to the id.
//   - Config — the merged definition + environment block (host, port, version,
//     …).
type ServiceEntry struct {
	Type   string
	Name   string
	Config service.Config
}

// ResolveType returns the entry's service type, falling back to id when the
// Type field is empty (the case where the key is the type).
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
// unique id.
type Profile struct {
	Services map[string]ServiceEntry
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
