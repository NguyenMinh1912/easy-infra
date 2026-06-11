package service

// Redis provisions a Redis in-memory data store service.
type Redis struct{}

// Name implements Service.
func (Redis) Name() string { return "redis" }

// DefaultConfig implements Service.
func (Redis) DefaultConfig() Config {
	return Config{
		"version": "7",
		"port":    6379,
	}
}

// Validate implements Service.
func (Redis) Validate(cfg Config) error {
	if _, err := optionalPort(cfg, "port", 6379); err != nil {
		return err
	}
	return nil
}
