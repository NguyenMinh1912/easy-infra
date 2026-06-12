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
	cfgs := make(map[string]service.Config, len(services))
	for _, name := range services {
		svc, ok := reg.Get(name)
		if !ok {
			return nil, fmt.Errorf("unknown service %q", name)
		}
		cfgs[name] = service.DefaultConfig(svc)
	}
	return &Profile{Services: cfgs}, nil
}
