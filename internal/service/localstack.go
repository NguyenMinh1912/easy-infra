package service

// LocalStack provisions a LocalStack service emulating AWS APIs locally.
type LocalStack struct{}

// Name implements Service.
func (LocalStack) Name() string { return "localstack" }

// DefaultConfig implements Service.
func (LocalStack) DefaultConfig() Config {
	return Config{
		"port":     4566,
		"services": "s3,sqs,sns",
	}
}

// Validate implements Service.
func (LocalStack) Validate(cfg Config) error {
	if _, err := optionalPort(cfg, "port", 4566); err != nil {
		return err
	}
	if _, err := requireString(cfg, "services"); err != nil {
		return err
	}
	return nil
}
