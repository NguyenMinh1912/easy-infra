# easy-infra

A command-line tool for managing a project's local/dev infrastructure. From a
project folder you initialize the project, define one or more **profiles**
describing the services that project needs, then apply a profile to bring those
services up.

Supported services: **postgres**, **minio**, **redis**, **localstack**.

## Concepts

- **Config** (`easy-infra.yml`, YAML) is user-authored and the source of truth.
  It defines the project's profiles and the per-service configuration in each.
- **State** (`.easy-infra/state.json`, JSON) is tool-owned. It records derived
  facts — most importantly the active profile — and is not hand-edited.
- A **profile** is a named bundle of service configurations (e.g. `default`,
  `staging-like`). Exactly one profile is active at a time.

See [`easy-infra.example.yml`](./easy-infra.example.yml) for a worked example.

## Commands

| Command | Purpose |
| --- | --- |
| `easy-infra init` | Scaffold the YAML config and create the JSON state file. |
| `easy-infra profile list` | List the project's profiles (active one marked `*`). |
| `easy-infra use <profile>` | Set `<profile>` as the active profile. |
| `easy-infra apply` | Reconcile the active profile: provision/start its services. |
| `easy-infra backup` | Back up data for the services in the active profile. |

> `apply` and `backup` currently report the per-service actions they would take;
> Docker-based provisioning is the next step (see `CLAUDE.md`).

## Build, test, run

```sh
go build ./...        # build
go test ./...         # run tests
go vet ./...          # static checks
gofmt -l .            # formatting (must be clean)
go run . init         # run locally
```

## Contributing

Read [`CONVENTIONS.md`](./CONVENTIONS.md) for the architecture, layering, and
SOLID rules this codebase follows. Adding a service is additive: implement the
`service.Service` interface and register it in `DefaultRegistry` — no other code
should reference the service by name.
