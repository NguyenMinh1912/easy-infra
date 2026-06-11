package service

// MinIO provisions a MinIO S3-compatible object storage service.
type MinIO struct{}

// Name implements Service.
func (MinIO) Name() string { return "minio" }

// DefaultConfig implements Service.
func (MinIO) DefaultConfig() Config {
	return Config{
		"port":        9000,
		"consolePort": 9001,
	}
}

// Validate implements Service.
func (MinIO) Validate(cfg Config) error {
	if _, err := optionalPort(cfg, "port", 9000); err != nil {
		return err
	}
	if _, err := optionalPort(cfg, "consolePort", 9001); err != nil {
		return err
	}
	return nil
}
