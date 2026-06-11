# Coding conventions

Conventions for contributing to `easy-infra`. These complement `CLAUDE.md`
(which describes *what* we build) by codifying *how* we write it. Keep changes
consistent with these rules; when a rule and the surrounding code disagree,
prefer the surrounding code and raise the discrepancy.

## Architecture & layering

The codebase follows a strict, one-directional dependency flow:

```
main.go  ->  cmd/  ->  internal/project  ->  internal/{config,state,service}
```

- **`cmd/` is thin.** A command parses flags and delegates. No business logic,
  file I/O, or validation lives in `cmd/` — it calls into `internal/`.
- **`internal/service`** is the lowest layer and depends on nothing else in the
  repo. `config` may import `service`; `service` must never import `config`.
- **`internal/project`** is the facade the command layer talks to. It wires
  config, state, and the service registry together so commands have a single
  dependency.
- Dependencies point inward/downward only. If you need an upward reference,
  invert it with an interface instead of creating an import cycle.

## SOLID in this codebase

- **Single responsibility** — one package, one concern: `config` owns YAML,
  `state` owns JSON, `service` owns service definitions, `project` orchestrates.
- **Open/closed** — adding a service is additive: implement the `service.Service`
  interface in its own file and register it in `DefaultRegistry`. Never add a
  `switch` on service names elsewhere; discover services through the registry.
- **Liskov** — every service is interchangeable behind `service.Service`;
  callers must not type-assert to a concrete service.
- **Interface segregation** — keep interfaces small (`Service` exposes only
  `Name`, `DefaultConfig`, `Validate`). Don't add methods a caller won't use.
- **Dependency inversion** — high-level code depends on abstractions. Commands
  receive a `*service.Registry` and `project.Paths` by injection
  (see `newRootCmd`); they don't construct global singletons.

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
- Make messages actionable — name the file, the profile, the allowed values
  (e.g. `unknown profile "ci" (available: [default])`).
- Export sentinel errors (e.g. `project.ErrNotInitialized`) when callers need to
  branch on a condition; check them with `errors.Is`.

## Config vs. state

- **Config (YAML)** is user-authored and the source of truth. Validate it on
  load and never silently rewrite it.
- **State (JSON)** is tool-owned. Never require users to hand-edit it. Write it
  indented so diffs stay readable.
- Commands operate on the **active profile** unless one is passed explicitly.

## Testing

- Unit-test the `internal/` packages; that is where logic lives. Table-driven
  tests are the default for validation and parsing.
- Use `t.TempDir()` for any filesystem interaction — tests must not touch the
  working tree or depend on external services.
- A change is not done until `go build ./...`, `go test ./...`, `go vet ./...`,
  and `gofmt -l .` are all clean.

[Effective Go]: https://go.dev/doc/effective_go
