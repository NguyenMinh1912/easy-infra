package service

// Redis provisions a Redis in-memory data store service.
type Redis struct{}

// Name implements Service.
func (Redis) Name() string { return "redis" }

// DefaultDefinition implements Service.
func (Redis) DefaultDefinition() Config {
	return Config{"version": "7"}
}

// ValidateDefinition implements Service.
func (Redis) ValidateDefinition(cfg Config) error {
	_, err := optionalString(cfg, "version", "7")
	return err
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
