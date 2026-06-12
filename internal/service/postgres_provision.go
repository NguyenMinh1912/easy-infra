package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// localHost is the address a forked local container is reached on. The container
// publishes its port on the loopback interface, so the local profile connects
// here rather than at the source profile's (possibly remote) host.
const localHost = "127.0.0.1"

// provisionReadyTimeout bounds how long Provision waits for a freshly launched
// postgres to start accepting connections before giving up.
const provisionReadyTimeout = 60 * time.Second

// provisionPollInterval is how often Provision re-checks a starting container's
// readiness.
const provisionPollInterval = time.Second

// pgParams is a profile's postgres connection settings, normalised out of either
// the discrete host/port/... fields or a single "url" DSN so the rest of the
// provisioning code can treat both shapes uniformly.
type pgParams struct {
	host     string
	port     int
	user     string
	password string
	database string
	schema   string
}

// pgParamsFrom extracts the connection settings from env, accepting both the
// discrete-field form and the "url" DSN form a profile may use.
func pgParamsFrom(env Config) (pgParams, error) {
	if raw, ok := env["url"]; ok {
		s, err := urlString(raw)
		if err != nil {
			return pgParams{}, err
		}
		u, err := postgresURL(s)
		if err != nil {
			return pgParams{}, err
		}
		port := 5432
		if p := u.Port(); p != "" {
			if n, err := strconv.Atoi(p); err == nil {
				port = n
			}
		}
		database := strings.TrimPrefix(u.Path, "/")
		if database == "" {
			return pgParams{}, fmt.Errorf("%q must include a database name", "url")
		}
		password, _ := u.User.Password()
		return pgParams{
			host:     u.Hostname(),
			port:     port,
			user:     u.User.Username(),
			password: password,
			database: database,
			schema:   u.Query().Get("search_path"),
		}, nil
	}

	host, err := requireString(env, "host")
	if err != nil {
		return pgParams{}, err
	}
	port, err := optionalPort(env, "port", 5432)
	if err != nil {
		return pgParams{}, err
	}
	user, err := requireString(env, "user")
	if err != nil {
		return pgParams{}, err
	}
	password, err := optionalString(env, "password", "")
	if err != nil {
		return pgParams{}, err
	}
	database, err := requireString(env, "database")
	if err != nil {
		return pgParams{}, err
	}
	schema, err := optionalString(env, "schema", "")
	if err != nil {
		return pgParams{}, err
	}
	return pgParams{host: host, port: port, user: user, password: password, database: database, schema: schema}, nil
}

// LocalEnv implements Provisioner: it produces the local profile's postgres env
// by taking the source's credentials, port, and database and pointing the host
// at loopback. The result is always in discrete-field form (even when the source
// used a "url"), so the forked profile reads clearly and is easy to edit.
func (Postgres) LocalEnv(source Config) (Config, error) {
	p, err := pgParamsFrom(source)
	if err != nil {
		return nil, err
	}
	env := Config{
		"host":     localHost,
		"port":     p.port,
		"user":     p.user,
		"password": p.password,
		"database": p.database,
	}
	if p.schema != "" {
		env["schema"] = p.schema
	}
	return env, nil
}

// Provision implements Provisioner: launch (idempotently) a local postgres
// container matching spec and wait until it accepts connections. The image
// version comes from the project definition; the published port and credentials
// come from the (already localised) profile env.
func (p Postgres) Provision(ctx context.Context, spec Spec) error {
	version, err := optionalString(spec.Definition, "version", "16")
	if err != nil {
		return err
	}
	params, err := pgParamsFrom(spec.Env)
	if err != nil {
		return err
	}

	env := map[string]string{
		"POSTGRES_USER": params.user,
		"POSTGRES_DB":   params.database,
	}
	if params.password != "" {
		env["POSTGRES_PASSWORD"] = params.password
	} else {
		// The official image refuses to start without a password unless trust auth
		// is explicitly opted into.
		env["POSTGRES_HOST_AUTH_METHOD"] = "trust"
	}

	c := containerSpec{
		Name:  localContainerName(spec.Profile, "postgres"),
		Image: "postgres:" + version,
		Env:   env,
		Ports: []portMapping{{Host: params.port, Container: 5432}},
	}
	spec.logf("ensuring container %s (%s)\n", c.Name, c.Image)
	if err := p.dockerClient().EnsureContainer(ctx, c); err != nil {
		return err
	}

	spec.logf("waiting for postgres to accept connections on %s:%d\n", localHost, params.port)
	return p.waitReady(ctx, spec)
}

// waitReady polls Health until the container is accepting connections or the
// timeout (or caller's context) elapses. A just-started postgres takes a few
// seconds to initialise, so the first probes are expected to fail.
func (p Postgres) waitReady(ctx context.Context, spec Spec) error {
	deadline := time.NewTimer(provisionReadyTimeout)
	defer deadline.Stop()
	ticker := time.NewTicker(provisionPollInterval)
	defer ticker.Stop()

	var lastErr error
	for {
		if err := p.Health(ctx, spec); err == nil {
			spec.logf("postgres is ready\n")
			return nil
		} else {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("postgres did not become ready within %s: %w", provisionReadyTimeout, lastErr)
		case <-ticker.C:
		}
	}
}
