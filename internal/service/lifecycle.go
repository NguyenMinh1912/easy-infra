package service

import (
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
