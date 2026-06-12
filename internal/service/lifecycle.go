package service

import (
	"context"
	"errors"
	"fmt"
	"io"
)

// Spec is a service's fully-resolved configuration for a lifecycle operation:
// the project-level Definition (what the service is — image/version) combined
// with the per-profile Env (how to reach it — host, port, credentials). The
// lifecycle methods act on a Spec rather than re-reading config, so they stay
// independent of where the two halves are loaded from.
type Spec struct {
	// Profile is the active profile's name. It organises per-profile artifacts
	// such as the backup directory and selects which backup Apply restores.
	Profile string
	// Definition is the project-level (easy-infra.yml) service definition.
	Definition Config
	// Env is the active profile's environment config for the service.
	Env Config
	// BackupDir is the snapshot folder a Backup should write into. The command
	// layer sets it so every service in one snapshot shares a single folder. When
	// empty, Backup falls back to its own fresh snapshot.
	BackupDir string
	// Snapshot selects which backup version an Apply restores, by snapshot id
	// (the timestamp folder name). When empty, Apply restores the latest
	// snapshot, preserving the default `easy-infra apply` behaviour.
	Snapshot string
	// Buckets optionally narrows a Backup to a subset of an object store's
	// buckets, by name. When empty, services that organise data into buckets
	// (minio) back up every bucket — the default `easy-infra backup` behaviour;
	// services without a bucket concept ignore it.
	Buckets []string
	// Log receives human-readable progress messages emitted during a lifecycle
	// operation. The command layer wires it up (e.g. for `backup snapshot
	// --verbose`) so users can follow what a service is doing; when nil, the
	// operation runs quietly.
	Log io.Writer
}

// logf writes a progress line to spec.Log when one is set, and is a no-op
// otherwise, so lifecycle methods can narrate their work without each one
// guarding against a nil writer.
func (s Spec) logf(format string, a ...any) {
	if s.Log == nil {
		return
	}
	fmt.Fprintf(s.Log, format, a...)
}

// Provisioner is an optional capability a Service implements when it can stand
// itself up as a local container — the backbone of "fork to local". It is kept
// separate from Service (rather than folded into it) so the not-yet-containerised
// services need not implement it: the fork flow type-asserts for it and degrades
// gracefully when a service does not provide it, mirroring how ErrNotImplemented
// is handled elsewhere. Detecting the capability by interface keeps the codebase
// free of service-name special-casing.
type Provisioner interface {
	// LocalEnv derives the per-profile environment config for a local container
	// from a source profile's env: it points the connection at localhost while
	// preserving the credentials/port so the fork mirrors the source's config.
	LocalEnv(source Config) (Config, error)

	// Provision brings up (idempotently) the local container described by spec —
	// its Definition selects the image, its Env the credentials and published
	// port — and waits until the service is ready to accept connections.
	Provision(ctx context.Context, spec Spec) error
}

// localContainerName is the deterministic container name for a service in a
// profile, e.g. "easy-infra-local-postgres". Reusing the same name makes
// provisioning idempotent and keeps a fork's containers easy to find with
// `docker ps`.
func localContainerName(profile, service string) string {
	return "easy-infra-" + profile + "-" + service
}

// ErrProtected is returned by Clean when the service's definition marks it as
// not cleanable (`cleanable: false`). The service is left untouched, protecting
// its data from an accidental teardown.
var ErrProtected = errors.New("service: protected from clean (cleanable is false)")

// ensureCleanable guards a Clean against a protected service: it returns
// ErrProtected when the definition sets `cleanable: false`, and nil otherwise.
// Clean implementations call it before doing any destructive work, so the
// protection is honoured uniformly across services.
func (s Spec) ensureCleanable() error {
	ok, err := cleanable(s.Definition)
	if err != nil {
		return err
	}
	if !ok {
		return ErrProtected
	}
	return nil
}

// ErrNotImplemented is returned by lifecycle operations whose Docker-backed
// provisioning is not wired up yet. Callers can detect it with errors.Is and
// degrade gracefully (e.g. report the action they would take) until the
// providers land.
var ErrNotImplemented = errors.New("service: operation not implemented")

// notImplemented builds the sentinel error for a service's lifecycle operation
// that has no provider yet, keeping the not-yet-wired-up seams DRY and giving
// callers an actionable, service-scoped message.
func notImplemented(svc, op string) error {
	return fmt.Errorf("%s %s: %w", svc, op, ErrNotImplemented)
}
