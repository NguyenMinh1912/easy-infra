package config

import (
	"fmt"

	"github.com/minhnc/easy-infra/internal/service"
)

// Scaffold builds a starter Config with a single profile named profileName
// containing the given services, each populated from its registered default
// configuration. It returns an error if a requested service is not registered.
//
// Pulling defaults from the registry keeps scaffolding open for extension: a
// newly registered service is scaffoldable without touching this function.
func Scaffold(reg *service.Registry, profileName string, services ...string) (*Config, error) {
	if profileName == "" {
		return nil, fmt.Errorf("profile name must not be empty")
	}
	svcConfigs := make(map[string]service.Config, len(services))
	for _, name := range services {
		svc, ok := reg.Get(name)
		if !ok {
			return nil, fmt.Errorf("unknown service %q", name)
		}
		svcConfigs[name] = svc.DefaultConfig()
	}
	return &Config{
		Version: CurrentVersion,
		Profiles: map[string]Profile{
			profileName: {Services: svcConfigs},
		},
	}, nil
}
