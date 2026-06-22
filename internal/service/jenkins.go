package service

import (
	"context"
	"fmt"
	"net/http"
)

// Jenkins manages a Jenkins continuous-integration automation server.
//
// ping is a seam for testing: when nil Health does a real HTTP GET against the
// server (realJenkinsPinger); tests set it to inject a canned result.
type Jenkins struct {
	ping jenkinsPinger
}

// jenkinsPinger probes a Jenkins server's base URL and reports whether it is
// reachable and ready. It is a seam like the other services' openers: the
// zero-value Jenkins does a real HTTP GET (realJenkinsPinger), while tests
// supply a fake without a live server.
type jenkinsPinger func(ctx context.Context, baseURL string) error

// pinger returns the prober to use, defaulting to a real HTTP GET.
func (j Jenkins) pinger() jenkinsPinger {
	if j.ping != nil {
		return j.ping
	}
	return realJenkinsPinger
}

// Name implements Service.
func (Jenkins) Name() string { return "jenkins" }

// DefaultDefinition implements Service.
func (Jenkins) DefaultDefinition() Config {
	return Config{"version": "lts", cleanableKey: true}
}

// ValidateDefinition implements Service.
func (Jenkins) ValidateDefinition(cfg Config) error {
	if _, err := optionalString(cfg, "version", "lts"); err != nil {
		return err
	}
	return validateCleanable(cfg)
}

// DefaultEnv implements Service.
func (Jenkins) DefaultEnv() Config {
	return Config{
		"host": "localhost",
		"port": 8080,
	}
}

// ValidateEnv implements Service. Credentials are optional: an anonymous-read
// Jenkins needs none, while a secured one is reached with a username and an API
// token (Jenkins's recommended alternative to a password for REST access).
func (Jenkins) ValidateEnv(cfg Config) error {
	if _, err := requireString(cfg, "host"); err != nil {
		return err
	}
	if _, err := optionalPort(cfg, "port", 8080); err != nil {
		return err
	}
	if _, err := optionalString(cfg, "user", ""); err != nil {
		return err
	}
	if _, err := optionalString(cfg, "token", ""); err != nil {
		return err
	}
	return nil
}

// jenkinsBaseURL builds the base URL a Jenkins server is reached on from a
// profile's env.
func jenkinsBaseURL(env Config) (string, error) {
	host, err := requireString(env, "host")
	if err != nil {
		return "", err
	}
	port, err := optionalPort(env, "port", 8080)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("http://%s:%d", host, port), nil
}

// Health implements Service: GET the Jenkins login page and confirm the server
// answers. Jenkins serves /login regardless of whether security is enabled and
// stamps every response with an X-Jenkins header carrying its version, so a
// reachable server is a ready one.
func (j Jenkins) Health(ctx context.Context, spec Spec) error {
	baseURL, err := jenkinsBaseURL(spec.Env)
	if err != nil {
		return err
	}
	if err := j.pinger()(ctx, baseURL); err != nil {
		return fmt.Errorf("jenkins not ready: %w", err)
	}
	return nil
}

// realJenkinsPinger does a live GET against the Jenkins login page and confirms
// the response carries the X-Jenkins header that identifies a Jenkins server,
// distinguishing a real Jenkins from any other service answering on the port.
func realJenkinsPinger(ctx context.Context, baseURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/login", nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.Header.Get("X-Jenkins") == "" {
		return fmt.Errorf("%s did not identify as a Jenkins server (status %s)", baseURL, resp.Status)
	}
	return nil
}

// Lifecycle provisioning (Apply/Backup/Clean) is the per-service seam for
// Docker-backed provisioning, which is future work; until a provider lands they
// report ErrNotImplemented.

// Apply implements Service.
func (Jenkins) Apply(context.Context, Spec) error { return notImplemented("jenkins", "apply") }

// Backup implements Service.
func (Jenkins) Backup(context.Context, Spec) error { return notImplemented("jenkins", "backup") }

// Clean implements Service.
func (Jenkins) Clean(_ context.Context, spec Spec) error {
	if err := spec.ensureCleanable(); err != nil {
		return err
	}
	return notImplemented("jenkins", "clean")
}
