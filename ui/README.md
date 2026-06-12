# easy-infra UI

Web frontend for `easy-infra`, built with [Vite](https://vitejs.dev/) + React +
TypeScript. It talks to the JSON API served by `easy-infra serve` and is
embedded into the Go binary (via `//go:embed`) for production.

## Develop

Run the Go backend and the Vite dev server side by side:

```sh
go run . serve          # backend API on :8080 (from the repo root)
make ui-dev             # Vite dev server on :5173, proxying /api to :8080
```

Open the URL Vite prints (default <http://localhost:5173>).

## Build & ship

```sh
make ui                 # builds ui/dist, which the Go binary embeds
go run . serve          # serves the built UI + API on :8080
```

`ui/dist` is gitignored; `easy-infra serve` shows a "run `make ui`" page until
the bundle has been built.

## API

`GET /api/status` → `{ initialized, activeProfile, profiles[], services[] }`.
