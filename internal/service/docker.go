package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
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
	// Cmd is the command (and args) to run in the container, appended after the
	// image. It is optional: images with a suitable entrypoint/CMD (e.g. postgres)
	// leave it nil, while others (e.g. minio, which needs `server /data ...`)
	// supply it.
	Cmd []string
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
	state, err := dockerInspect(ctx, c.Name)
	if err != nil {
		return err
	}
	if state.exists {
		// Docker bakes published ports into a container at creation time, so a
		// restart re-binds whatever port the container was created with — ignoring
		// any change to the profile's port. If the existing container's ports no
		// longer match the desired spec (e.g. the user moved 5432 -> 5433 to dodge
		// a clash), reusing it would re-bind the stale port and fail; replace it.
		if !samePorts(state.ports, c.Ports) {
			if out, err := runDocker(ctx, "rm", "-f", c.Name); err != nil {
				return fmt.Errorf("replacing container %s: %w: %s", c.Name, err, out)
			}
		} else if state.running {
			return nil
		} else {
			// Present, stopped, and still matches the spec — start it back up
			// rather than re-creating.
			if out, err := runDocker(ctx, "start", c.Name); err != nil {
				return fmt.Errorf("starting container %s: %w: %s", c.Name, err, out)
			}
			return nil
		}
	}

	args := []string{"run", "-d", "--name", c.Name}
	for _, k := range sortedEnvKeys(c.Env) {
		args = append(args, "-e", k+"="+c.Env[k])
	}
	for _, p := range c.Ports {
		args = append(args, "-p", fmt.Sprintf("127.0.0.1:%d:%d", p.Host, p.Container))
	}
	args = append(args, c.Image)
	args = append(args, c.Cmd...)
	if out, err := runDocker(ctx, args...); err != nil {
		return fmt.Errorf("creating container %s: %w: %s", c.Name, err, out)
	}
	return nil
}

// containerState captures what EnsureContainer needs to know about an existing
// container: whether it exists, whether it is running, and the host ports it was
// created with (so a stale port binding can be detected).
type containerState struct {
	exists  bool
	running bool
	ports   []portMapping
}

// dockerInspect reports the state of the named container. A container that docker
// does not know about is reported as a zero containerState (exists=false) — the
// caller then creates it.
func dockerInspect(ctx context.Context, name string) (containerState, error) {
	// Combine the running flag and the configured port bindings into one inspect
	// so EnsureContainer can decide between reuse, restart, and replace.
	out, err := runDocker(ctx, "inspect", "-f", "{{.State.Running}}|{{json .HostConfig.PortBindings}}", name)
	if err != nil {
		// `docker inspect` exits non-zero for an unknown container; treat that as
		// "does not exist" rather than a hard failure, but surface a daemon that is
		// genuinely unreachable. The "no such object/container" wording varies by
		// docker version and casing, so match case-insensitively.
		if low := strings.ToLower(out); strings.Contains(low, "no such object") || strings.Contains(low, "no such container") {
			return containerState{}, nil
		}
		return containerState{}, fmt.Errorf("inspecting container %s: %w: %s", name, err, strings.TrimSpace(out))
	}
	runningStr, bindingsJSON, _ := strings.Cut(strings.TrimSpace(out), "|")
	ports, err := parsePortBindings(bindingsJSON)
	if err != nil {
		return containerState{}, fmt.Errorf("inspecting container %s: %w", name, err)
	}
	return containerState{exists: true, running: strings.TrimSpace(runningStr) == "true", ports: ports}, nil
}

// parsePortBindings decodes docker's HostConfig.PortBindings JSON (e.g.
// {"5432/tcp":[{"HostIp":"127.0.0.1","HostPort":"5433"}]}) into portMappings.
// An empty or null value means no published ports.
func parsePortBindings(s string) ([]portMapping, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "null" {
		return nil, nil
	}
	var raw map[string][]struct {
		HostIP   string `json:"HostIp"`
		HostPort string `json:"HostPort"`
	}
	if err := json.Unmarshal([]byte(s), &raw); err != nil {
		return nil, fmt.Errorf("parsing port bindings %q: %w", s, err)
	}
	var ports []portMapping
	for containerPort, binds := range raw {
		cp, err := strconv.Atoi(strings.SplitN(containerPort, "/", 2)[0])
		if err != nil {
			continue
		}
		for _, b := range binds {
			hp, err := strconv.Atoi(b.HostPort)
			if err != nil {
				continue
			}
			ports = append(ports, portMapping{Host: hp, Container: cp})
		}
	}
	return ports, nil
}

// samePorts reports whether two sets of port mappings are equal, ignoring order.
func samePorts(a, b []portMapping) bool {
	if len(a) != len(b) {
		return false
	}
	type key struct{ host, container int }
	counts := make(map[key]int, len(a))
	for _, p := range a {
		counts[key{p.Host, p.Container}]++
	}
	for _, p := range b {
		counts[key{p.Host, p.Container}]--
	}
	for _, n := range counts {
		if n != 0 {
			return false
		}
	}
	return true
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
