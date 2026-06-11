package service

import "context"

// MinIO provisions a MinIO S3-compatible object storage service.
type MinIO struct{}

// Name implements Service.
func (MinIO) Name() string { return "minio" }

// DefaultDefinition implements Service.
func (MinIO) DefaultDefinition() Config {
	return Config{"version": "latest"}
}

// ValidateDefinition implements Service.
func (MinIO) ValidateDefinition(cfg Config) error {
	_, err := optionalString(cfg, "version", "latest")
	return err
}

// DefaultEnv implements Service.
func (MinIO) DefaultEnv() Config {
	return Config{
		"host":        "localhost",
		"port":        9000,
		"consolePort": 9001,
		"user":        "minioadmin",
		"password":    "minioadmin",
	}
}

// ValidateEnv implements Service.
func (MinIO) ValidateEnv(cfg Config) error {
	if _, err := requireString(cfg, "host"); err != nil {
		return err
	}
	if _, err := optionalPort(cfg, "port", 9000); err != nil {
		return err
	}
	if _, err := optionalPort(cfg, "consolePort", 9001); err != nil {
		return err
	}
	if _, err := requireString(cfg, "user"); err != nil {
		return err
	}
	if _, err := requireString(cfg, "password"); err != nil {
		return err
	}
	return nil
}

// Lifecycle operations are the per-service seam for Docker-backed provisioning,
// which is future work; until a provider lands they report ErrNotImplemented.

// Apply implements Service.
func (MinIO) Apply(context.Context, Spec) error { return notImplemented("minio", "apply") }

// Health implements Service.
func (MinIO) Health(context.Context, Spec) error { return notImplemented("minio", "health") }

// Backup implements Service.
func (MinIO) Backup(context.Context, Spec) error { return notImplemented("minio", "backup") }

// Clean implements Service.
func (MinIO) Clean(context.Context, Spec) error { return notImplemented("minio", "clean") }
