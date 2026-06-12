package service

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Container provisioning is what turns a profile's service config into a real,
// locally-running instance — the backbone of "fork to local". easy-infra shells
// out to the `docker` CLI rather than taking a Docker SDK dependency, keeping
// the surface small and matching what a developer would run by hand.

// containerSpec describes a container easy-infra wants running: a stable name
// (so the operation is idempotent), the image, the environment, and the host
// ports published from it.
type containerSpec struct {
	// Name is the deterministic container name, e.g. "easy-infra-local-postgres",
	// so repeated provisioning reuses the same container rather than stacking up.
	Name  string
	Image string
	// Env is the container's environment (e.g. POSTGRES_PASSWORD), passed as -e.
	Env map[string]string
	// Ports maps host ports to container ports, published on 127.0.0.1 so the
	// local instance is reachable only from this machine.
	Ports []portMapping
}

// portMapping publishes a container port on a host port.
type portMapping struct {
	Host      int
	Container int
}

// dockerRunner ensures containers exist and run. It is a seam: the zero-value
// Postgres uses realDocker, while tests inject a fake to assert what would be
// launched without touching a real daemon.
type dockerRunner interface {
	// EnsureContainer brings the container described by c to a running state,
	// creating it if absent and starting it if stopped. It is idempotent: a
	// container that is already running is left as-is.
	EnsureContainer(ctx context.Context, c containerSpec) error
}

// realDocker drives the local `docker` CLI.
type realDocker struct{}

// EnsureContainer implements dockerRunner against the docker binary. It inspects
// the named container first so an existing one is reused (started if stopped)
// rather than re-created, making provisioning safe to run repeatedly.
func (realDocker) EnsureContainer(ctx context.Context, c containerSpec) error {
	switch running, exists, err := dockerInspectRunning(ctx, c.Name); {
	case err != nil:
		return err
	case exists && running:
		return nil
	case exists:
		// Present but stopped — start it back up rather than re-creating.
		if out, err := runDocker(ctx, "start", c.Name); err != nil {
			return fmt.Errorf("starting container %s: %w: %s", c.Name, err, out)
		}
		return nil
	}

	args := []string{"run", "-d", "--name", c.Name}
	for _, k := range sortedEnvKeys(c.Env) {
		args = append(args, "-e", k+"="+c.Env[k])
	}
	for _, p := range c.Ports {
		args = append(args, "-p", fmt.Sprintf("127.0.0.1:%d:%d", p.Host, p.Container))
	}
	args = append(args, c.Image)
	if out, err := runDocker(ctx, args...); err != nil {
		return fmt.Errorf("creating container %s: %w: %s", c.Name, err, out)
	}
	return nil
}

// dockerInspectRunning reports whether the named container exists and, if so,
// whether it is running. A container that docker does not know about is reported
// as (false, false, nil) — the caller then creates it.
func dockerInspectRunning(ctx context.Context, name string) (running, exists bool, err error) {
	out, err := runDocker(ctx, "inspect", "-f", "{{.State.Running}}", name)
	if err != nil {
		// `docker inspect` exits non-zero for an unknown container; treat that as
		// "does not exist" rather than a hard failure, but surface a daemon that is
		// genuinely unreachable. The "no such object/container" wording varies by
		// docker version and casing, so match case-insensitively.
		if low := strings.ToLower(out); strings.Contains(low, "no such object") || strings.Contains(low, "no such container") {
			return false, false, nil
		}
		return false, false, fmt.Errorf("inspecting container %s: %w: %s", name, err, strings.TrimSpace(out))
	}
	return strings.TrimSpace(out) == "true", true, nil
}

// runDocker runs the docker CLI with args and returns its combined output. The
// output is included in errors so a failed command is actionable.
func runDocker(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}

// sortedEnvKeys returns the env map's keys in sorted order, so a container's
// `-e` flags are emitted deterministically (stable logs and tests).
func sortedEnvKeys(env map[string]string) []string {
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	// Small maps; a simple insertion keeps it dependency-free and deterministic.
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j-1] > keys[j]; j-- {
			keys[j-1], keys[j] = keys[j], keys[j-1]
		}
	}
	return keys
}
