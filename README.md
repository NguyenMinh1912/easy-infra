# easy-infra

A command-line tool for managing a project's local/dev infrastructure. From a
project folder you initialize the project, define one or more **profiles**
describing the services that project needs, then apply a profile to bring those
services up.

Supported services: **postgres**, **minio**, **redis**, **localstack**.

## Concepts

Configuration is organised around **profiles**:

- **Project config** (`easy-infra.yml`, YAML) is a thin marker that records the
  schema version and tells the tool a folder is an easy-infra project. It is
  tracked in git and holds no services or secrets.
- **Profile config** (`.easy-infra/profiles/<name>.yml`, YAML) is where services
  live. Each profile **owns its own services** and can add, edit, or remove them
  independently. A profile is keyed by service name, and each service block holds
  the full config in one place — both *what* the service is (e.g. `version`,
  `cleanable`) and *how to reach* it (host, port, user, password, database URL).
  A **profile** is one environment (e.g. `default`, `ci`, `staging`); profiles
  hold credentials and are gitignored.
- **State** (`.easy-infra/state.json`, JSON) is tool-owned. It records derived
  facts — most importantly the active profile — and is not hand-edited. Exactly
  one profile is active at a time.

See [`easy-infra.profile.example.yml`](./easy-infra.profile.example.yml) for a
worked profile example.

## Commands

| Command | Purpose |
| --- | --- |
| `easy-infra init` | Scaffold the YAML config and create the JSON state file. |
| `easy-infra profile list` | List the project's profiles (active one marked `*`). |
| `easy-infra use <profile>` | Set `<profile>` as the active profile. |
| `easy-infra apply` | Reconcile the active profile: provision/start its services. |
| `easy-infra backup` | Back up data for the services in the active profile. |
| `easy-infra ui` | Run the web UI and a JSON API for inspecting the project (`--port`, default 8080). |

> `apply` and `backup` currently report the per-service actions they would take;
> Docker-based provisioning is the next step (see `CLAUDE.md`).

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
go run . init         # run locally without installing
```

## Web UI

`easy-infra ui` starts a local server with a React dashboard (Vite +
TypeScript, under `ui/`) and a JSON API. Build the bundle once, then run it:

```sh
make ui               # build the frontend into ui/dist (embedded in the binary)
go run . ui           # open http://localhost:8080
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
backend (`make ui-dev` + `go run . ui`); see [`ui/README.md`](./ui/README.md).

## Contributing

Read [`CONVENTIONS.md`](./CONVENTIONS.md) for the architecture, layering, and
SOLID rules this codebase follows. Adding a service is additive: implement the
`service.Service` interface and register it in `DefaultRegistry` — no other code
should reference the service by name.
