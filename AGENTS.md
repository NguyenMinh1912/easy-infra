# AGENTS.md

Entry point for AI coding agents working in `easy-infra`. This is the
tool-agnostic companion to [`CLAUDE.md`](./CLAUDE.md); both describe the same
project. Read this first for orientation, then follow the deeper docs linked
below for the area you're touching.

## What this project is

`easy-infra` is an **app** for managing a project's local/dev infrastructure.
Running the binary launches a local web UI + JSON API; from the UI a user
creates **workspaces**, defines **profiles** of services within them, and
applies a profile to bring those services up. There is no separate CLI for these
operations — the binary's job is to serve the app.

Supported services: **postgres**, **minio**, **redis**, **localstack**.

All data lives in a **single SQLite database** in the user config directory
(`$EASY_INFRA_CONFIG_DIR` or `os.UserConfigDir()/easy-infra/easy-infra.db`). It
is tool-owned — never require users to hand-edit it. There are no project
folders, `easy-infra.yml` files, or JSON state files.

## Where things live

```
main.go                 # thin entrypoint -> cmd.Execute()
cmd/                     # cobra root command (serves the app) + serve/ui alias
internal/
  store/                 # the SQLite store: workspaces, profiles, services, active selections
  profile/               # profile value types + validation (no I/O)
  project/               # facade over the store for a workspace (used by the server)
  backup/                # backup session store (shares the central DB)
  server/                # HTTP API + embedded UI
  service/               # postgres, minio, redis, localstack providers (common interface)
ui/                      # React + Vite + TS web app, embedded into the binary
```

Dependency flow is strictly one-directional:

```
main.go -> cmd/ -> internal/server -> internal/project -> internal/{store,profile,service}
```

`internal/service` is the lowest layer and imports nothing else in the repo.
Dependencies point inward/downward only; invert with an interface rather than
creating an import cycle.

## Golden rules

- **Add a service additively.** Implement the `service.Service` interface in its
  own file under `internal/service/` and register it in `DefaultRegistry`
  (`internal/service/service.go`). Never add a `switch` on service names
  anywhere else — discover services through the registry.
- **The SQLite store is the single source of truth** and tool-owned.
  `internal/store` holds *persistence only* — no policy.
- **Domain rules live in `internal/project`** (and `internal/{profile,service}`),
  not in the store: a profile needs ≥1 valid service, the active profile can't
  be removed, and so on. Validate on read/write; don't push policy into the store.
- **The server operates on the active workspace and its active profile** unless
  one is passed explicitly.
- **Return errors up to the handler/`cmd/` layer.** Don't `os.Exit` or print to
  stderr from `internal/`. Wrap with `fmt.Errorf("...: %w", err)` and surface
  clear, actionable messages (name the profile, list allowed values). Export
  sentinel errors when callers need to branch on them.

## Build, test, run

A change is **not done** until `go build`, `go test`, `go vet`, and `gofmt -l .`
are all clean. Run these before considering work complete:

```sh
make build            # or: go build ./...
make test             # or: go test ./...
make vet              # or: go vet ./...
make fmt              # gofmt -w over all sources (must be clean)
go run .              # run locally; serves the UI on http://localhost:8080
make dev              # backend (air hot-reload) + Vite HMR together
```

The web UI lives in `ui/` and is embedded into the binary. After changing it:

```sh
make ui               # build ui/dist (embedded by the Go binary)
npm --prefix ui run build   # typecheck (tsc -b) + vite build — must pass clean
```

A broken `ui/` build breaks the embedded binary, so build before committing.

## Further reading

| Doc | Scope |
| --- | --- |
| [`CLAUDE.md`](./CLAUDE.md) | Project overview, data model, commands, layout (Claude-specific notes). |
| [`CONVENTIONS.md`](./CONVENTIONS.md) | Go architecture, layering, SOLID, errors, persistence, testing. |
| [`ui/CONVENTIONS.md`](./ui/CONVENTIONS.md) | Frontend (React/Vite/TS) architecture, styling, state, definition of done. |
| [`README.md`](./README.md) | User-facing concepts, install, and run instructions. |
| [`Makefile`](./Makefile) | Every build/test/dev target and what it does. |

When a rule here and the surrounding code disagree, prefer the surrounding code
and raise the discrepancy.
