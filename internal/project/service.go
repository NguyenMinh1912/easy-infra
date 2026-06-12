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

// ServiceConfig describes one service instance within a profile: its unique id,
// its service type (the registry key), its user-facing name, and the merged
// definition + environment block. It is the shape the command and HTTP layers
// read.
type ServiceConfig struct {
	ID     string
	Type   string
	Name   string
	Config service.Config
}

// ProfileServices returns the named profile's service instances, sorted by id.
// It reports a missing profile as an actionable error.
func (p *Project) ProfileServices(profileName string) ([]ServiceConfig, error) {
	services, err := p.ProfileConfig(profileName)
	if err != nil {
		return nil, err
	}
	defs := make([]ServiceConfig, 0, len(services))
	for _, id := range sortedServiceIDs(services) {
		entry := services[id]
		defs = append(defs, ServiceConfig{
			ID:     id,
			Type:   entry.ResolveType(id),
			Name:   entry.ResolveName(id),
			Config: entry.Config,
		})
	}
	return defs, nil
}

// AddProfileService adds an instance of the given service type to a profile,
// then saves it and returns the new instance's id. A profile may hold several
// instances of the same type, so a unique id is generated rather than rejecting
// a duplicate. name defaults to the id when empty; cfg defaults to the service's
// default config when nil, and is otherwise validated against the service.
func (p *Project) AddProfileService(profileName, svcType, name string, cfg service.Config) (string, error) {
	svc, ok := p.Registry.Get(svcType)
	if !ok {
		return "", fmt.Errorf("%q: %w", svcType, ErrUnknownService)
	}
	prof, err := p.loadProfileForEdit(profileName)
	if err != nil {
		return "", err
	}
	if cfg == nil {
		cfg = service.DefaultConfig(svc)
	} else if err := service.ValidateConfig(svc, cfg); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidDefinition, err)
	}
	id := uniqueServiceID(prof.Services, svcType)
	if name == "" {
		name = id
	}
	prof.Services[id] = profile.ServiceEntry{Type: svcType, Name: name, Config: cfg}
	if err := p.SaveProfile(profileName, prof); err != nil {
		return "", err
	}
	return id, nil
}

// UpdateProfileService replaces the config of the instance identified by id in a
// profile, after validating it against the instance's service type, then saves
// the profile. A non-empty name renames the instance; an empty name leaves it
// unchanged.
func (p *Project) UpdateProfileService(profileName, id, name string, cfg service.Config) error {
	prof, err := p.loadProfileForEdit(profileName)
	if err != nil {
		return err
	}
	entry, exists := prof.Services[id]
	if !exists {
		return fmt.Errorf("%q: %w", id, ErrServiceNotDefined)
	}
	svcType := entry.ResolveType(id)
	svc, ok := p.Registry.Get(svcType)
	if !ok {
		return fmt.Errorf("%q: %w", svcType, ErrUnknownService)
	}
	if err := service.ValidateConfig(svc, cfg); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidDefinition, err)
	}
	entry.Type = svcType
	entry.Config = cfg
	if name != "" {
		entry.Name = name
	} else {
		entry.Name = entry.ResolveName(id)
	}
	prof.Services[id] = entry
	return p.SaveProfile(profileName, prof)
}

// RemoveProfileService removes the instance identified by id from a profile. It
// refuses to remove the profile's last remaining service, since a profile must
// define at least one.
func (p *Project) RemoveProfileService(profileName, id string) error {
	prof, err := p.loadProfileForEdit(profileName)
	if err != nil {
		return err
	}
	if _, exists := prof.Services[id]; !exists {
		return fmt.Errorf("%q: %w", id, ErrServiceNotDefined)
	}
	if len(prof.Services) == 1 {
		return fmt.Errorf("%q: %w", id, ErrLastService)
	}
	delete(prof.Services, id)
	return p.SaveProfile(profileName, prof)
}

// uniqueServiceID returns an id for a new instance of svcType that does not
// collide with an existing one. The first instance of a type takes the type
// name itself (so a single-instance profile reads naturally and matches the
// pre-instance-id layout); subsequent instances get a numeric suffix.
func uniqueServiceID(services map[string]profile.ServiceEntry, svcType string) string {
	if _, exists := services[svcType]; !exists {
		return svcType
	}
	for i := 2; ; i++ {
		id := fmt.Sprintf("%s-%d", svcType, i)
		if _, exists := services[id]; !exists {
			return id
		}
	}
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

// sortedServiceIDs returns the ids of a profile's service instances in sorted
// order.
func sortedServiceIDs(services map[string]profile.ServiceEntry) []string {
	ids := make([]string, 0, len(services))
	for id := range services {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
