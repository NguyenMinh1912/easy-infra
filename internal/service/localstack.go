package service

import "context"

// LocalStack provisions a LocalStack service emulating AWS APIs locally.
type LocalStack struct{}

// Name implements Service.
func (LocalStack) Name() string { return "localstack" }

// DefaultDefinition implements Service.
func (LocalStack) DefaultDefinition() Config {
	// Which AWS services to emulate is a property of the service itself, so it
	// lives in the project-level definition rather than per environment.
	return Config{
		"version":    "latest",
		"services":   "s3,sqs,sns",
		cleanableKey: true,
	}
}

// ValidateDefinition implements Service.
func (LocalStack) ValidateDefinition(cfg Config) error {
	if _, err := optionalString(cfg, "version", "latest"); err != nil {
		return err
	}
	if _, err := requireString(cfg, "services"); err != nil {
		return err
	}
	return validateCleanable(cfg)
}

// DefaultEnv implements Service.
func (LocalStack) DefaultEnv() Config {
	return Config{
		"host": "localhost",
		"port": 4566,
	}
}

// ValidateEnv implements Service.
func (LocalStack) ValidateEnv(cfg Config) error {
	if _, err := requireString(cfg, "host"); err != nil {
		return err
	}
	if _, err := optionalPort(cfg, "port", 4566); err != nil {
		return err
	}
	return nil
}

// Lifecycle operations are the per-service seam for Docker-backed provisioning,
// which is future work; until a provider lands they report ErrNotImplemented.

// Apply implements Service.
func (LocalStack) Apply(context.Context, Spec) error { return notImplemented("localstack", "apply") }

// Health implements Service.
func (LocalStack) Health(context.Context, Spec) error { return notImplemented("localstack", "health") }

// Backup implements Service.
func (LocalStack) Backup(context.Context, Spec) error { return notImplemented("localstack", "backup") }

// Clean implements Service.
func (LocalStack) Clean(_ context.Context, spec Spec) error {
	if err := spec.ensureCleanable(); err != nil {
		return err
	}
	return notImplemented("localstack", "clean")
}
