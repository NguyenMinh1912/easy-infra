package service

import (
	"context"
	"fmt"
)

// Redis provisions a Redis in-memory data store service.
//
// open is a seam for testing: when nil the console and key browser dial a real
// server via go-redis (realRedisOpener); tests set it to inject a fake client.
type Redis struct {
	open redisOpener
}

// opener returns the client opener to use, defaulting to a real go-redis dial.
func (r Redis) opener() redisOpener {
	if r.open != nil {
		return r.open
	}
	return realRedisOpener
}

// connect opens a client for the profile env, selecting logical database db.
func (r Redis) connect(_ context.Context, env Config, db int) (redisClient, error) {
	client, err := r.opener()(env, db)
	if err != nil {
		return nil, fmt.Errorf("connecting to redis: %w", err)
	}
	return client, nil
}

// Name implements Service.
func (Redis) Name() string { return "redis" }

// DefaultDefinition implements Service.
func (Redis) DefaultDefinition() Config {
	return Config{"version": "7", cleanableKey: true}
}

// ValidateDefinition implements Service.
func (Redis) ValidateDefinition(cfg Config) error {
	if _, err := optionalString(cfg, "version", "7"); err != nil {
		return err
	}
	return validateCleanable(cfg)
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
	if _, err := optionalString(cfg, "password", ""); err != nil {
		return err
	}
	if _, err := redisDB(cfg); err != nil {
		return err
	}
	return nil
}

// redisDB reads the optional "db" logical-database index from an env config,
// defaulting to 0. It must be a non-negative whole number.
func redisDB(env Config) (int, error) {
	raw, ok := env["db"]
	if !ok {
		return 0, nil
	}
	n, err := asInt(raw)
	if err != nil {
		return 0, fmt.Errorf("%q must be a whole number, got %v", "db", raw)
	}
	if n < 0 {
		return 0, fmt.Errorf("%q must not be negative, got %d", "db", n)
	}
	return n, nil
}

// Health implements Service: connect to the configured database and confirm it
// answers a PING.
func (r Redis) Health(ctx context.Context, spec Spec) error {
	db, err := redisDB(spec.Env)
	if err != nil {
		return err
	}
	client, err := r.connect(ctx, spec.Env, db)
	if err != nil {
		return err
	}
	defer client.Close()
	if _, err := client.Do(ctx, "PING"); err != nil {
		return fmt.Errorf("redis not ready: %w", err)
	}
	return nil
}

// Lifecycle provisioning (Apply/Backup/Clean) is the per-service seam for
// Docker-backed provisioning, which is future work; until a provider lands they
// report ErrNotImplemented.

// Apply implements Service.
func (Redis) Apply(context.Context, Spec) error { return notImplemented("redis", "apply") }

// Backup implements Service.
func (Redis) Backup(context.Context, Spec) error { return notImplemented("redis", "backup") }

// Clean implements Service.
func (Redis) Clean(_ context.Context, spec Spec) error {
	if err := spec.ensureCleanable(); err != nil {
		return err
	}
	return notImplemented("redis", "clean")
}
