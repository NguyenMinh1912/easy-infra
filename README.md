# easy-infra

A command-line tool for managing a project's local/dev infrastructure. From a
project folder you initialize the project, define one or more **profiles**
describing the services that project needs, then apply a profile to bring those
services up.

Supported services: **postgres**, **minio**, **redis**, **localstack**.

## Concepts

Configuration is split into two layers:

- **Project config** (`easy-infra.yml`, YAML) is user-authored and the source of
  truth for *which* services the project uses and their environment-independent
  definition (image/version). It is tracked in git and contains no secrets.
- **Profile config** (`.easy-infra/profiles/<name>.yml`, YAML) describes *how to
  reach* each service in one environment — host, port, user, password, database
  URL. A **profile** is one such environment (e.g. `default`, `ci`, `staging`).
  Profiles hold credentials and are gitignored.
- **State** (`.easy-infra/state.json`, JSON) is tool-owned. It records derived
  facts — most importantly the active profile — and is not hand-edited. Exactly
  one profile is active at a time.

See [`easy-infra.example.yml`](./easy-infra.example.yml) (project) and
[`easy-infra.profile.example.yml`](./easy-infra.profile.example.yml) (profile)
for worked examples.

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

Install `easy-infra` as a global command (drops the binary in `go env GOPATH`'s
`bin`):

```sh
make install
```

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

For frontend development run the Vite dev server alongside the backend
(`make ui-dev` + `go run . ui`); see [`ui/README.md`](./ui/README.md).

## Contributing

Read [`CONVENTIONS.md`](./CONVENTIONS.md) for the architecture, layering, and
SOLID rules this codebase follows. Adding a service is additive: implement the
`service.Service` interface and register it in `DefaultRegistry` — no other code
should reference the service by name.
