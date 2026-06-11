package service

// Postgres provisions a PostgreSQL database service.
type Postgres struct{}

// Name implements Service.
func (Postgres) Name() string { return "postgres" }

// DefaultConfig implements Service.
func (Postgres) DefaultConfig() Config {
	return Config{
		"version":  "16",
		"port":     5432,
		"database": "app",
	}
}

// Validate implements Service.
func (Postgres) Validate(cfg Config) error {
	if _, err := optionalPort(cfg, "port", 5432); err != nil {
		return err
	}
	if _, err := requireString(cfg, "database"); err != nil {
		return err
	}
	return nil
}
