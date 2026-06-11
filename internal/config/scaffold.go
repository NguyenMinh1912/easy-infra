package config

import (
	"fmt"

	"github.com/minhnc/easy-infra/internal/service"
)

// Scaffold builds a starter project Config defining the given services, each
// populated from its registered default definition. It returns an error if a
// requested service is not registered.
//
// Pulling defaults from the registry keeps scaffolding open for extension: a
// newly registered service is scaffoldable without touching this function.
func Scaffold(reg *service.Registry, services ...string) (*Config, error) {
	if len(services) == 0 {
		return nil, fmt.Errorf("at least one service is required")
	}
	defs := make(map[string]service.Config, len(services))
	for _, name := range services {
		svc, ok := reg.Get(name)
		if !ok {
			return nil, fmt.Errorf("unknown service %q", name)
		}
		defs[name] = svc.DefaultDefinition()
	}
	return &Config{Version: CurrentVersion, Services: defs}, nil
}
