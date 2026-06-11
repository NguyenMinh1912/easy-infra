package profile

import (
	"fmt"

	"github.com/minhnc/easy-infra/internal/service"
)

// Scaffold builds a starter Profile with default environment config for each
// of the given services, pulled from the registry. It returns an error if a
// requested service is not registered.
func Scaffold(reg *service.Registry, services ...string) (*Profile, error) {
	if len(services) == 0 {
		return nil, fmt.Errorf("at least one service is required")
	}
	envs := make(map[string]service.Config, len(services))
	for _, name := range services {
		svc, ok := reg.Get(name)
		if !ok {
			return nil, fmt.Errorf("unknown service %q", name)
		}
		envs[name] = svc.DefaultEnv()
	}
	return &Profile{Services: envs}, nil
}
