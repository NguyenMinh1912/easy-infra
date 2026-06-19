# Coding conventions

Conventions for contributing to `easy-infra`. These complement `CLAUDE.md`
(which describes *what* we build) by codifying *how* we write it. Keep changes
consistent with these rules; when a rule and the surrounding code disagree,
prefer the surrounding code and raise the discrepancy.

## Architecture & layering

The codebase follows a strict, one-directional dependency flow:

```
main.go  ->  cmd/  ->  internal/server  ->  internal/project  ->  internal/{store,profile,service}
```

- **`cmd/` is thin.** It opens the store and starts the server; no business
  logic, persistence, or validation lives there.
- **`internal/service`** is the lowest layer and depends on nothing else in the
  repo. `store` and `profile` may import `service`; `service` must never import
  them.
- **`internal/store`** owns persistence (the single SQLite database) and holds
  no policy: it returns domain types and maps storage errors to sentinels.
- **`internal/project`** is the facade the server talks to. It wires the store,
  profiles, and the service registry together and layers validation on top, so
  handlers have a single dependency.
- Dependencies point inward/downward only. If you need an upward reference,
  invert it with an interface instead of creating an import cycle.

### Service config: definition vs. environment

A service's config block holds two logical halves, both stored together as one
JSON column per service instance (see `internal/store`):

- *definition* — what the service is: environment-independent settings like
  image/version and `cleanable`.
- *environment* — how to reach it: host, port, user, password, database URL.

Each `service.Service` owns the schema for *both* halves
(`DefaultDefinition`/`ValidateDefinition` and `DefaultEnv`/`ValidateEnv`). Keep
definition fields and environment fields on the correct side of that line.

## SOLID in this codebase

- **Single responsibility** — one package, one concern: `store` owns
  persistence, `profile` owns the value types and validation, `service` owns
  service definitions, `project` orchestrates.
- **Open/closed** — adding a service is additive: implement the `service.Service`
  interface in its own file and register it in `DefaultRegistry`. Never add a
  `switch` on service names elsewhere; discover services through the registry.
- **Liskov** — every service is interchangeable behind `service.Service`;
  callers must not type-assert to a concrete service.
- **Interface segregation** — `Service` exposes only what a service must
  describe: its name plus the default/validate pair for each config half
  (definition and environment). Don't add methods a caller won't use.
- **Dependency inversion** — high-level code depends on abstractions. The server
  receives a `*service.Registry`, a `*store.Store`, and the UI filesystem by
  injection (see `server.New`); it doesn't construct global singletons.

## Go style

- Run `gofmt` (must be clean) and `go vet ./...` before considering work done.
- Follow [Effective Go] and the standard library's naming: short receiver
  names, `MixedCaps`, no `Get` prefix on simple accessors where avoidable.
- Exported identifiers have doc comments that start with the identifier name.
- Prefer accepting interfaces and returning concrete types.

## Errors

- Return errors up to `cmd/`; that layer is responsible for presenting them to
  the user. Don't `os.Exit` or print to stderr from `internal/`.
- Wrap with context using `fmt.Errorf("...: %w", err)` so the chain is
  inspectable with `errors.Is`/`errors.As`.
- Make messages actionable — name the profile, the allowed values
  (e.g. `unknown profile "ci" (available: [default])`).
- Export sentinel errors (e.g. `store.ErrProfileNotFound`,
  `project.ErrNotInitialized`) when callers need to branch on a condition; check
  them with `errors.Is`.

## Persistence

- **The SQLite store is the single source of truth** and tool-owned. Never
  require users to hand-edit it. `internal/store` holds persistence only.
- **Validation lives above the store**, in `internal/project` /
  `internal/service`: a profile must define at least one valid service, the
  active profile cannot be removed, and so on. Validate on read/write; don't
  push policy down into the store.
- The server operates on the **active workspace** and its **active profile**
  unless one is passed explicitly.

## Testing

- Unit-test the `internal/` packages; that is where logic lives. Table-driven
  tests are the default for validation and parsing.
- Use `t.TempDir()` for any filesystem interaction — tests must not touch the
  working tree or depend on external services.
- A change is not done until `go build ./...`, `go test ./...`, `go vet ./...`,
  and `gofmt -l .` are all clean.

[Effective Go]: https://go.dev/doc/effective_go
