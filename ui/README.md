# easy-infra UI

Web frontend for `easy-infra`, built with [Vite](https://vitejs.dev/) + React +
TypeScript, styled with [Tailwind CSS v4](https://tailwindcss.com/) and
[shadcn/ui](https://ui.shadcn.com/). It talks to the JSON API served by
`easy-infra serve` and is embedded into the Go binary (via `//go:embed`) for
production. It is served by `easy-infra ui`.

See [`CONVENTIONS.md`](./CONVENTIONS.md) for the architecture and coding rules.

## Project layout

```
src/
├── app/                 # app shell (layout, mounts features)
├── components/
│   ├── ui/              # shadcn/ui primitives (button, card, badge, …)
│   └── layout/          # shared presentational layout (PageHeader)
├── features/
│   └── dashboard/       # feature slice: components/, hooks/, page container
├── hooks/               # generic reusable hooks (useAsync)
├── services/api/        # transport layer — the only place that calls fetch
├── types/               # domain models mirroring the API contract
├── lib/                 # pure helpers (cn)
└── index.css            # Tailwind + design tokens
```

## Develop

The fastest path is a single command from the repo root that hot-reloads both
the backend and the frontend:

```sh
make dev                # air-watched backend (:8080) + Vite dev server (:5173)
```

It rebuilds/reruns the Go backend on `.go` changes (via air) and serves the SPA
with Vite HMR, shutting both down on Ctrl+C. Open the URL Vite prints (default
<http://localhost:5173>).

To run the two processes by hand instead:

```sh
go run . ui             # backend API on :8080 (from the repo root)
make ui-dev             # Vite dev server on :5173, proxying /api to :8080
```

## Build & ship

```sh
make ui                 # builds ui/dist, which the Go binary embeds
go run . ui             # serves the built UI + API on :8080
```

`ui/dist` is gitignored; `easy-infra ui` shows a "run `make ui`" page until
the bundle has been built.

## Add a UI component

shadcn/ui components are copied into `src/components/ui` and owned by us:

```sh
cd ui && npx shadcn@latest add <name>   # e.g. dialog, input, table
```

## API

`GET /api/status` → `{ initialized, activeProfile, profiles[], services[] }`.
