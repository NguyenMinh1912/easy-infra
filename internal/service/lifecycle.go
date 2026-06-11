package service

import (
	"errors"
	"fmt"
)

// Spec is a service's fully-resolved configuration for a lifecycle operation:
// the project-level Definition (what the service is — image/version) combined
// with the per-profile Env (how to reach it — host, port, credentials). The
// lifecycle methods act on a Spec rather than re-reading config, so they stay
// independent of where the two halves are loaded from.
type Spec struct {
	// Definition is the project-level (easy-infra.yml) service definition.
	Definition Config
	// Env is the active profile's environment config for the service.
	Env Config
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
