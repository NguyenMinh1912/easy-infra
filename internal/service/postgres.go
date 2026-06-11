package service

// Postgres provisions a PostgreSQL database service.
type Postgres struct{}

// Name implements Service.
func (Postgres) Name() string { return "postgres" }

// DefaultDefinition implements Service.
func (Postgres) DefaultDefinition() Config {
	return Config{"version": "16"}
}

// ValidateDefinition implements Service.
func (Postgres) ValidateDefinition(cfg Config) error {
	_, err := optionalString(cfg, "version", "16")
	return err
}

// DefaultEnv implements Service.
func (Postgres) DefaultEnv() Config {
	return Config{
		"host":     "localhost",
		"port":     5432,
		"user":     "app",
		"password": "app",
		"database": "app",
	}
}

// ValidateEnv implements Service.
func (Postgres) ValidateEnv(cfg Config) error {
	if _, err := requireString(cfg, "host"); err != nil {
		return err
	}
	if _, err := optionalPort(cfg, "port", 5432); err != nil {
		return err
	}
	if _, err := requireString(cfg, "user"); err != nil {
		return err
	}
	if _, err := requireString(cfg, "database"); err != nil {
		return err
	}
	return nil
}
