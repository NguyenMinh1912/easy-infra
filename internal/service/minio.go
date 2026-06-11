package service

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
