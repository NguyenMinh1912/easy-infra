package service

import "context"

// Redis provisions a Redis in-memory data store service.
type Redis struct{}

// Name implements Service.
func (Redis) Name() string { return "redis" }

// DefaultDefinition implements Service.
func (Redis) DefaultDefinition() Config {
	return Config{"version": "7", cleanableKey: true}
}

// ValidateDefinition implements Service.
func (Redis) ValidateDefinition(cfg Config) error {
	if _, err := optionalString(cfg, "version", "7"); err != nil {
		return err
	}
	return validateCleanable(cfg)
}

// DefaultEnv implements Service.
func (Redis) DefaultEnv() Config {
	return Config{
		"host": "localhost",
		"port": 6379,
	}
}

// ValidateEnv implements Service.
func (Redis) ValidateEnv(cfg Config) error {
	if _, err := requireString(cfg, "host"); err != nil {
		return err
	}
	if _, err := optionalPort(cfg, "port", 6379); err != nil {
		return err
	}
	return nil
}

// Lifecycle operations are the per-service seam for Docker-backed provisioning,
// which is future work; until a provider lands they report ErrNotImplemented.

// Apply implements Service.
func (Redis) Apply(context.Context, Spec) error { return notImplemented("redis", "apply") }

// Health implements Service.
func (Redis) Health(context.Context, Spec) error { return notImplemented("redis", "health") }

// Backup implements Service.
func (Redis) Backup(context.Context, Spec) error { return notImplemented("redis", "backup") }

// Clean implements Service.
func (Redis) Clean(_ context.Context, spec Spec) error {
	if err := spec.ensureCleanable(); err != nil {
		return err
	}
	return notImplemented("redis", "clean")
}
