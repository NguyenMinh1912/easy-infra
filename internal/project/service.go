package project

import (
	"errors"
	"fmt"
	"io/fs"
	"sort"

	"github.com/minhnc/easy-infra/internal/profile"
	"github.com/minhnc/easy-infra/internal/service"
)

// Errors returned by the per-profile service CRUD operations. They are
// sentinels so callers (e.g. the HTTP server) can map each condition onto an
// appropriate response without string matching.
var (
	// ErrUnknownService means the name is not a service easy-infra supports.
	ErrUnknownService = errors.New("unknown service")
	// ErrServiceExists means the profile already defines the service.
	ErrServiceExists = errors.New("service already defined")
	// ErrServiceNotDefined means the profile does not define the service.
	ErrServiceNotDefined = errors.New("service not defined")
	// ErrLastService means removing the service would leave the profile with
	// none, which a profile does not allow.
	ErrLastService = errors.New("cannot remove the last service")
	// ErrInvalidDefinition means a supplied config failed the service's own
	// validation.
	ErrInvalidDefinition = errors.New("invalid service config")
)

// ServiceConfig pairs a service name with its config within a profile (the
// merged definition + environment block). It is the shape the command and HTTP
// layers read.
type ServiceConfig struct {
	Name   string
	Config service.Config
}

// ProfileServices returns the named profile's service configs, sorted by name.
// It reports a missing profile as an actionable error.
func (p *Project) ProfileServices(profileName string) ([]ServiceConfig, error) {
	services, err := p.ProfileConfig(profileName)
	if err != nil {
		return nil, err
	}
	defs := make([]ServiceConfig, 0, len(services))
	for _, name := range sortedServiceNames(services) {
		defs = append(defs, ServiceConfig{Name: name, Config: services[name]})
	}
	return defs, nil
}

// AddProfileService adds the named service to a profile using its default
// config, then saves the profile. It errors if the service is unknown or the
// profile already defines it.
func (p *Project) AddProfileService(profileName, name string) error {
	svc, ok := p.Registry.Get(name)
	if !ok {
		return fmt.Errorf("%q: %w", name, ErrUnknownService)
	}
	prof, err := p.loadProfileForEdit(profileName)
	if err != nil {
		return err
	}
	if _, exists := prof.Services[name]; exists {
		return fmt.Errorf("%q: %w", name, ErrServiceExists)
	}
	prof.Services[name] = service.DefaultConfig(svc)
	return p.SaveProfile(profileName, prof)
}

// UpdateProfileService replaces the named service's config in a profile with
// cfg, after validating it against the service, then saves the profile.
func (p *Project) UpdateProfileService(profileName, name string, cfg service.Config) error {
	svc, ok := p.Registry.Get(name)
	if !ok {
		return fmt.Errorf("%q: %w", name, ErrUnknownService)
	}
	prof, err := p.loadProfileForEdit(profileName)
	if err != nil {
		return err
	}
	if _, exists := prof.Services[name]; !exists {
		return fmt.Errorf("%q: %w", name, ErrServiceNotDefined)
	}
	if err := service.ValidateConfig(svc, cfg); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidDefinition, err)
	}
	prof.Services[name] = cfg
	return p.SaveProfile(profileName, prof)
}

// RemoveProfileService removes the named service from a profile. It refuses to
// remove the profile's last remaining service, since a profile must define at
// least one.
func (p *Project) RemoveProfileService(profileName, name string) error {
	prof, err := p.loadProfileForEdit(profileName)
	if err != nil {
		return err
	}
	if _, exists := prof.Services[name]; !exists {
		return fmt.Errorf("%q: %w", name, ErrServiceNotDefined)
	}
	if len(prof.Services) == 1 {
		return fmt.Errorf("%q: %w", name, ErrLastService)
	}
	delete(prof.Services, name)
	return p.SaveProfile(profileName, prof)
}

// loadProfileForEdit loads a profile without validation (via profile.Load) so a
// block momentarily out of sync can still be edited, reporting a missing
// profile as an actionable error.
func (p *Project) loadProfileForEdit(name string) (*profile.Profile, error) {
	prof, err := profile.Load(p.Paths.ProfilePath(name))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("profile %q does not exist", name)
		}
		return nil, err
	}
	return prof, nil
}

// sortedServiceNames returns the keys of services in sorted order.
func sortedServiceNames(services map[string]service.Config) []string {
	names := make([]string, 0, len(services))
	for name := range services {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
