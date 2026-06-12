package service

import (
	"context"
	"fmt"
	"time"
)

// LocalEnv implements Provisioner: it produces the local profile's minio env by
// taking the source's credentials and ports and pointing the host at loopback.
// The result is in discrete-field form so the forked profile reads clearly and
// is easy to edit.
func (MinIO) LocalEnv(source Config) (Config, error) {
	p, err := minioParamsFrom(source)
	if err != nil {
		return nil, err
	}
	env := Config{
		"host":        localHost,
		"port":        p.port,
		"consolePort": p.consolePort,
		"user":        p.user,
		"password":    p.password,
	}
	// A local container terminates TLS itself only when explicitly asked to; the
	// fork mirrors the source's choice so the derived endpoint scheme matches.
	if p.secure {
		env["secure"] = true
	}
	return env, nil
}

// Provision implements Provisioner: launch (idempotently) a local minio
// container matching spec and wait until it accepts connections. The image
// version comes from the project definition; the published ports and root
// credentials come from the (already localised) profile env.
func (m MinIO) Provision(ctx context.Context, spec Spec) error {
	version, err := optionalString(spec.Definition, "version", "latest")
	if err != nil {
		return err
	}
	params, err := minioParamsFrom(spec.Env)
	if err != nil {
		return err
	}

	c := containerSpec{
		Name:  localContainerName(spec.Profile, "minio"),
		Image: "minio/minio:" + version,
		Env: map[string]string{
			"MINIO_ROOT_USER":     params.user,
			"MINIO_ROOT_PASSWORD": params.password,
		},
		// The S3 API listens on 9000 and the console on 9001 inside the container;
		// publish each on the profile's chosen host ports.
		Ports: []portMapping{
			{Host: params.port, Container: 9000},
			{Host: params.consolePort, Container: 9001},
		},
		// The official image's entrypoint expects the server command and a data
		// directory; pin the console to 9001 so the published mapping is stable.
		Cmd: []string{"server", "/data", "--console-address", ":9001"},
	}
	spec.logf("ensuring container %s (%s)\n", c.Name, c.Image)
	if err := m.dockerClient().EnsureContainer(ctx, c); err != nil {
		return err
	}

	spec.logf("waiting for minio to accept connections on %s:%d\n", localHost, params.port)
	return m.waitReady(ctx, spec)
}

// waitReady polls Health until the container is accepting connections or the
// timeout (or caller's context) elapses. A just-started minio takes a moment to
// initialise, so the first probes are expected to fail.
func (m MinIO) waitReady(ctx context.Context, spec Spec) error {
	deadline := time.NewTimer(provisionReadyTimeout)
	defer deadline.Stop()
	ticker := time.NewTicker(provisionPollInterval)
	defer ticker.Stop()

	var lastErr error
	for {
		if err := m.Health(ctx, spec); err == nil {
			spec.logf("minio is ready\n")
			return nil
		} else {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("minio did not become ready within %s: %w", provisionReadyTimeout, lastErr)
		case <-ticker.C:
		}
	}
}
