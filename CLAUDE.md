# CLAUDE.md

Guidance for Claude Code (and humans) working in this repository.

## What we build

`easy-infra` is an **app** for managing a project's local/dev infrastructure.
Running the binary launches a local web UI + JSON API; from the UI a user
creates one or more **workspaces**, defines **profiles** of services within
them, and applies a profile to bring those services up. There is no separate CLI
for these operations — the binary's job is to serve the app.

Supported services:

- **postgres**
- **minio**
- **redis**
- **localstack**

### Data model & storage

- **All data lives in a single SQLite database** in the user config directory
  (`$EASY_INFRA_CONFIG_DIR` or `os.UserConfigDir()/easy-infra/easy-infra.db`).
  It is tool-owned and not meant to be hand-edited. There are no project folders,
  no `easy-infra.yml`, and no JSON state files.
- A **workspace** is a named bundle of profiles (it replaces the old "project
  folder"). Exactly one workspace is active at a time.
- A **profile** is a named bundle of service configurations (e.g. `default`,
  `ci`, `staging-like`) within a workspace. Exactly one profile is active per
  workspace. A service instance's config is stored as a JSON column.
- Backup sessions live in the same database, scoped by workspace. Snapshot
  artifacts are written under the backups directory keyed by profile/service.

## Commands

The binary serves the app — running `easy-infra` (no subcommand) starts it.

| Command | Purpose |
| --- | --- |
| `easy-infra` (default) | Open the central store and serve the web UI + JSON API. |
| `easy-infra serve` (alias `ui`) | Same as the default; explicit form. `--port` to choose the port. |

Workspaces, profiles, services, apply, and backup are all managed from the web
UI (and the `/api` endpoints behind it), not from CLI subcommands.

## Tech stack

- **Language:** Go.
- **Entrypoint:** [cobra](https://github.com/spf13/cobra) for the root/serve
  command.
- **Storage:** [`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite), a
  pure-Go SQLite driver (no cgo), opened in WAL mode with a single writer.
- Service connections use `jackc/pgx` (postgres) and `minio-go` (minio).

## Suggested layout

```
.
├── main.go                 # thin entrypoint -> cmd.Execute()
├── cmd/                    # root command (serves the app) + serve alias
├── internal/
│   ├── store/              # the SQLite store: workspaces, profiles, services, active selections
│   ├── profile/            # profile value types + validation (no I/O)
│   ├── project/            # facade over the store for a workspace (used by the server)
│   ├── backup/             # backup session store (shares the central DB)
│   ├── server/             # HTTP API + embedded UI
│   └── service/            # postgres, minio, redis providers (common interface)
└── ui/                     # the web app (embedded into the binary)
```

Keep `cmd/` thin. Put persistence in `internal/store`, validation in
`internal/profile`/`internal/service`, and workflow logic in `internal/project`.

## Conventions

- Add a new service by implementing the common `service` interface and
  registering it; don't special-case service names across the codebase.
- The SQLite store is the single source of truth and tool-owned — never require
  users to edit it by hand. `internal/store` holds persistence only; domain
  rules (a profile needs ≥1 valid service, the active profile can't be removed)
  live in `internal/project`.
- The server operates on the **active workspace** and its **active profile**
  unless one is passed explicitly.
- Return errors up to the handler/`cmd/` layer; surface clear, actionable
  messages to the user.

## Build, test, run

```sh
go build ./...        # build
go test ./...         # run tests
go vet ./...          # static checks
gofmt -l .            # formatting (must be clean)
go run .              # run the app locally (serves the UI on :8080)
```

The web UI lives in `ui/`. Build it with `npm --prefix ui run build` (output is
embedded into the Go binary); typecheck with `npm --prefix ui run build`.

Run `go test ./...` and `gofmt` before considering a change done.
