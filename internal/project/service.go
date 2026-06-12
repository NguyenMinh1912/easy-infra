package project

import (
	"errors"
	"fmt"

	"github.com/minhnc/easy-infra/internal/profile"
	"github.com/minhnc/easy-infra/internal/service"
)

// Errors returned by the service-definition CRUD operations. They are sentinels
// so callers (e.g. the HTTP server) can map each condition onto an appropriate
// response without string matching.
var (
	// ErrUnknownService means the name is not a service easy-infra supports.
	ErrUnknownService = errors.New("unknown service")
	// ErrServiceExists means the project already defines the service.
	ErrServiceExists = errors.New("service already defined")
	// ErrServiceNotDefined means the project does not define the service.
	ErrServiceNotDefined = errors.New("service not defined")
	// ErrLastService means removing the service would leave the project with
	// none, which the config does not allow.
	ErrLastService = errors.New("cannot remove the last service")
	// ErrInvalidDefinition means a supplied definition failed the service's own
	// validation.
	ErrInvalidDefinition = errors.New("invalid service definition")
)

// ServiceDefinition pairs a service name with its project-level definition
// config (easy-infra.yml). It is the shape the command and HTTP layers read.
type ServiceDefinition struct {
	Name       string
	Definition service.Config
}

// Services returns the project's service definitions, sorted by name.
func (p *Project) Services() []ServiceDefinition {
	names := p.Config.ServiceNames()
	defs := make([]ServiceDefinition, 0, len(names))
	for _, name := range names {
		defs = append(defs, ServiceDefinition{Name: name, Definition: p.Config.Services[name]})
	}
	return defs
}

// AddService adds the named service to the project config using its default
// definition, then scaffolds default environment config for it into every
// existing profile so they remain valid. It errors if the service is unknown or
// already defined.
func (p *Project) AddService(name string) error {
	svc, ok := p.Registry.Get(name)
	if !ok {
		return fmt.Errorf("%q: %w", name, ErrUnknownService)
	}
	if p.Config.HasService(name) {
		return fmt.Errorf("%q: %w", name, ErrServiceExists)
	}

	// Scaffold env into existing profiles first; if a profile write fails the
	// config is still untouched, so the project stays consistent.
	if err := p.eachProfile(func(prof *profile.Profile) {
		prof.Services[name] = svc.DefaultEnv()
	}); err != nil {
		return err
	}

	if p.Config.Services == nil {
		p.Config.Services = map[string]service.Config{}
	}
	p.Config.Services[name] = svc.DefaultDefinition()
	return p.Config.Save(p.Paths.Config)
}

// UpdateService replaces the named service's project-level definition with def,
// after validating it against the service. The change is environment-independent
// so profiles are left untouched.
func (p *Project) UpdateService(name string, def service.Config) error {
	svc, ok := p.Registry.Get(name)
	if !ok {
		return fmt.Errorf("%q: %w", name, ErrUnknownService)
	}
	if !p.Config.HasService(name) {
		return fmt.Errorf("%q: %w", name, ErrServiceNotDefined)
	}
	if err := svc.ValidateDefinition(def); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidDefinition, err)
	}
	p.Config.Services[name] = def
	return p.Config.Save(p.Paths.Config)
}

// RemoveService removes the named service from the project config and from every
// profile's environment config. It refuses to remove the last remaining service,
// since a project must define at least one.
func (p *Project) RemoveService(name string) error {
	if !p.Config.HasService(name) {
		return fmt.Errorf("%q: %w", name, ErrServiceNotDefined)
	}
	if len(p.Config.Services) == 1 {
		return fmt.Errorf("%q: %w", name, ErrLastService)
	}

	if err := p.eachProfile(func(prof *profile.Profile) {
		delete(prof.Services, name)
	}); err != nil {
		return err
	}

	delete(p.Config.Services, name)
	return p.Config.Save(p.Paths.Config)
}

// eachProfile loads every profile, applies mutate, and saves it back. Profiles
// are loaded without validation (via profile.Load) so a partially-consistent
// state mid edit does not block the very edit that would fix it.
func (p *Project) eachProfile(mutate func(*profile.Profile)) error {
	names, err := p.Profiles()
	if err != nil {
		return err
	}
	for _, name := range names {
		prof, err := profile.Load(p.Paths.ProfilePath(name))
		if err != nil {
			return err
		}
		mutate(prof)
		if err := p.SaveProfile(name, prof); err != nil {
			return err
		}
	}
	return nil
}
