# easy-infra

An app for managing a project's local/dev infrastructure. Running the binary
launches a local web UI and JSON API; from the UI you create one or more
**workspaces**, define **profiles** of services within them, and apply a profile
to bring those services up.

Supported services: **postgres**, **minio**, **redis**, **localstack**.

## Concepts

All data lives in a **single local SQLite database** in your user config
directory (`$EASY_INFRA_CONFIG_DIR`, or `os.UserConfigDir()/easy-infra/easy-infra.db`).
It is tool-owned and not meant to be hand-edited — there are no project folders,
no `easy-infra.yml`, and no JSON state files.

- A **workspace** is a named bundle of profiles. Exactly one workspace is active
  at a time. Create, rename, switch, and remove them from the UI.
- A **profile** is one environment (e.g. `default`, `ci`, `staging`) within a
  workspace. Each profile **owns its own services** and can add, edit, or remove
  them independently; exactly one profile is active per workspace. A service
  block holds the full config in one place — both *what* the service is (e.g.
  `version`, `cleanable`) and *how to reach* it (host, port, user, password,
  database URL).
- **Backups** are recorded in the same database, scoped to their workspace.

Everything — workspaces, profiles, services, apply, and backup — is managed in
the app and the `/api` endpoints behind it. There is no CLI for these operations.

## Run

The binary serves the app:

```sh
easy-infra            # serve the web UI + JSON API on http://localhost:8080
easy-infra --port 9000
easy-infra serve      # explicit form (alias: ui)
```

On first run the database is empty; create your first workspace from the UI.

## Install

### Quick install (recommended)

Install the latest prebuilt binary for your OS/arch with one command — no Go
toolchain required:

```sh
curl -fsSL https://raw.githubusercontent.com/NguyenMinh1912/easy-infra/main/install.sh | sh
```

This downloads the matching release asset, verifies its checksum, and installs
`easy-infra` to `/usr/local/bin` (or `~/.local/bin` if that isn't writable).
Pin a version with `EASY_INFRA_VERSION=v1.2.3` or change the target directory
with `EASY_INFRA_BIN_DIR`. Releases are published automatically from a version
tag by [`.github/workflows/release.yml`](./.github/workflows/release.yml).

### Windows

Windows ships as a self-contained `easy-infra.exe` (no install, no Go toolchain,
no extra runtime). Download `easy-infra_windows_amd64.zip` — or
`easy-infra_windows_arm64.zip` on ARM devices — from the
[latest release](https://github.com/NguyenMinh1912/easy-infra/releases/latest),
unzip it, and run the executable:

```powershell
.\easy-infra.exe            # serve the web UI + JSON API on http://localhost:8080
.\easy-infra.exe --port 9000
```

Double-clicking `easy-infra.exe` works too: it opens a console window serving
the app — leave it running and open <http://localhost:8080> in your browser.
Data lives in `%AppData%\easy-infra\easy-infra.db`. (Windows Defender
SmartScreen may warn about an unsigned binary on first run; choose *More info →
Run anyway*.)

### Build from source

Install `easy-infra` as a global command (drops the binary in `go env GOPATH`'s
`bin`):

```sh
make install
```

No Go toolchain installed? `make install` bootstraps one automatically: if `go`
isn't on your `PATH` it downloads a pinned Go release into `./.go` (via
`scripts/ensure-go.sh`) and builds with that — no manual setup required.

Make sure that directory is on your `PATH`, then run `easy-infra` from anywhere:

```sh
export PATH="$(go env GOPATH)/bin:$PATH"   # add to your shell profile to persist
easy-infra --help
```

`make uninstall` removes it.

## Build, test, run

```sh
make build            # compile the binary into ./easy-infra
make test             # run tests
make vet              # static checks
make fmt              # format the code
make clean            # remove build artifacts
go run .              # run locally without installing (serves the UI on :8080)
```

## Web UI

`easy-infra` starts a local server with a React dashboard (Vite + TypeScript,
under `ui/`) and a JSON API. Build the bundle once, then run it:

```sh
make ui               # build the frontend into ui/dist (embedded in the binary)
go run .              # open http://localhost:8080
```

For development, one command runs both apps with hot reload — no build needed:

```sh
make dev              # backend (auto-restarts on .go changes) + Vite HMR
```

Then open the Vite URL it prints (<http://localhost:5173>). `make dev` runs the
Go backend under [air](https://github.com/air-verse/air) — pinned as a Go `tool`
dependency, so it doesn't affect the release binary — and the Vite dev server
together, tearing both down on Ctrl+C. The frontend hot-reloads via Vite; the
backend rebuilds and restarts on `.go` changes.

To run the pieces manually instead, start the Vite dev server alongside the
backend (`make ui-dev` + `go run .`); see [`ui/README.md`](./ui/README.md).

## Contributing

Read [`CONVENTIONS.md`](./CONVENTIONS.md) for the architecture, layering, and
SOLID rules this codebase follows. Adding a service is additive: implement the
`service.Service` interface and register it in `DefaultRegistry` — no other code
should reference the service by name.
