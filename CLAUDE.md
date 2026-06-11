# CLAUDE.md

Guidance for Claude Code (and humans) working in this repository.

## What we build

`easy-infra` is a command-line tool for managing a project's local/dev
infrastructure. From a project folder, a user initializes the project, defines
one or more **profiles** describing the services that project needs, and then
applies a profile to bring those services up.

Supported services:

- **postgres**
- **minio**
- **redis**

### Configuration & state

- **Config** is authored by the user in **YAML** (e.g. `easy-infra.yml`). It is
  the source of truth: it defines the project's profiles and, per profile, the
  configuration of each service.
- **State** is managed by the tool in **JSON** (e.g. `.easy-infra/state.json`).
  It records runtime/derived facts — most importantly the currently active
  profile — and is not meant to be hand-edited.

A **profile** is a named bundle of service configurations (e.g. `default`,
`ci`, `staging-like`). A project may have many profiles; exactly one is "active"
at a time, tracked in state.

## Commands

| Command | Purpose |
| --- | --- |
| `easy-infra init` | Initialize a project in the current folder: scaffold the YAML config and create the JSON state file. |
| `easy-infra profile ...` | Manage profiles — list, add, edit, remove profiles and their service config. |
| `easy-infra use <profile>` | Set `<profile>` as the active profile (records it in state). |
| `easy-infra apply` | Reconcile the active profile: provision/start the services it defines. |
| `easy-infra backup` | Back up data for the services in the active profile. |

## Tech stack

- **Language:** Go.
- **CLI framework:** [cobra](https://github.com/spf13/cobra) for commands;
  [viper](https://github.com/spf13/viper) (or `gopkg.in/yaml.v3`) for config.
- **State serialization:** standard library `encoding/json`.
- Services are provisioned via Docker (shelling out to `docker` / compose, or
  the Docker SDK) — confirm the approach before introducing a dependency.

## Suggested layout

```
.
├── main.go                 # thin entrypoint -> cmd.Execute()
├── cmd/                    # one file per command (init, profile, use, apply, backup)
├── internal/
│   ├── config/             # YAML config: load, validate, save
│   ├── state/              # JSON state: active profile, etc.
│   └── service/            # postgres, minio, redis providers (common interface)
└── easy-infra.yml          # example/scaffolded project config
```

Keep `cmd/` thin — parse flags and delegate. Put real logic in `internal/`.

## Conventions

- Add a new service by implementing the common `service` interface and
  registering it; don't special-case service names across the codebase.
- Config (YAML) is user-owned and validated on load; state (JSON) is
  tool-owned — never require users to edit it by hand.
- Commands operate on the **active profile** unless one is passed explicitly.
- Return errors up to `cmd/`; surface clear, actionable messages to the user.

## Build, test, run

```sh
go build ./...        # build
go test ./...         # run tests
go vet ./...          # static checks
gofmt -l .            # formatting (must be clean)
go run . <command>    # run locally, e.g. `go run . init`
```

Run `go test ./...` and `gofmt` before considering a change done.
