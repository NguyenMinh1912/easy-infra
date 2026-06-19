package service

import "context"

// LocalStack provisions a LocalStack service emulating AWS APIs locally.
//
// openSQS / openSES are seams for testing: when nil the cloud browser dials a
// real endpoint via the AWS SDK (realSQSOpener / realSESOpener); tests set them
// to inject fake clients.
type LocalStack struct {
	openSQS sqsOpener
	openSES sesOpener
}

// sqsOpener returns the SQS client opener to use, defaulting to a real dial.
func (l LocalStack) sqsOpener() sqsOpener {
	if l.openSQS != nil {
		return l.openSQS
	}
	return realSQSOpener
}

// sesOpener returns the SES client opener to use, defaulting to a real dial.
func (l LocalStack) sesOpener() sesOpener {
	if l.openSES != nil {
		return l.openSES
	}
	return realSESOpener
}

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
		"host":   "localhost",
		"port":   4566,
		"region": "us-east-1",
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
	if _, err := optionalString(cfg, "region", "us-east-1"); err != nil {
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
