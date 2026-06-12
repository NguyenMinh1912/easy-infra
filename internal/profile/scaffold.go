package profile

import (
	"fmt"

	"github.com/minhnc/easy-infra/internal/service"
)

// Scaffold builds a starter Profile with default config for each of the given
// services, pulled from the registry. Each block merges the service's default
// definition and environment, the shape a profile stores. It returns an error
// if a requested service is not registered.
func Scaffold(reg *service.Registry, services ...string) (*Profile, error) {
	if len(services) == 0 {
		return nil, fmt.Errorf("at least one service is required")
	}
	entries := make(map[string]ServiceEntry, len(services))
	for _, name := range services {
		svc, ok := reg.Get(name)
		if !ok {
			return nil, fmt.Errorf("unknown service %q", name)
		}
		// A scaffolded instance is keyed by its service type, so its id, type,
		// and name all coincide; Type/Name are left empty and resolve to the id.
		entries[name] = ServiceEntry{Config: service.DefaultConfig(svc)}
	}
	return &Profile{Services: entries}, nil
}
