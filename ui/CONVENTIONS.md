# Frontend conventions

Conventions for the `easy-infra` web UI (`ui/`). These complement the root
[`CONVENTIONS.md`](../CONVENTIONS.md) (Go) and [`CLAUDE.md`](../CLAUDE.md)
(what we build) by codifying *how* we write the React frontend. When a rule and
the surrounding code disagree, prefer the surrounding code and raise the
discrepancy.

## Stack

- **Build:** [Vite](https://vitejs.dev/) + React 18 + TypeScript (strict).
- **Styling:** [Tailwind CSS v4](https://tailwindcss.com/) via
  `@tailwindcss/vite` — no `tailwind.config.js`; tokens live in
  `src/index.css`.
- **Components:** [shadcn/ui](https://ui.shadcn.com/) (new-york style, neutral
  base) — copied into `src/components/ui`, owned and editable by us.
- **Icons:** [lucide-react](https://lucide.dev/).
- **Data:** the JSON API served by `easy-infra serve` (`GET /api/status`).

## Architecture & layering

The UI follows a clean, one-directional dependency flow. Dependencies point
**downward only**; lower layers never import upward.

```
app/  ->  features/  ->  components/ (ui, layout)  ->  hooks/ + lib/
                  \-->  services/api/  ->  types/
```

| Layer | Path | Responsibility |
| --- | --- | --- |
| **App shell** | `src/app/` | Page chrome, providers, routing. Thin — wires features, holds no business logic. |
| **Features** | `src/features/<feature>/` | A vertical slice: its own `components/`, `hooks/`, and a page container. Self-contained; features don't import each other's internals. |
| **UI primitives** | `src/components/ui/` | shadcn/ui primitives (Button, Card, …). Presentational, app-agnostic. |
| **Shared layout** | `src/components/layout/` | Cross-feature presentational pieces (e.g. `PageHeader`). |
| **Shared hooks** | `src/hooks/` | Generic, reusable hooks (e.g. `useAsync`). No feature/domain knowledge. |
| **Services** | `src/services/api/` | Transport layer. The only place that calls `fetch`. Maps responses onto domain types. |
| **Types** | `src/types/` | Domain models mirroring the API contract. The single source of truth for data shapes. |
| **Utilities** | `src/lib/` | Pure helpers (e.g. `cn`). No React. |

- **Container vs. presentation.** A feature's page container (e.g.
  `DashboardPage`) owns data loading and maps async state to views. The
  components it renders are pure: they take props and render, never fetch.
- **Public surface via barrels.** A feature exposes only what the app shell
  needs through its `index.ts` (e.g. `DashboardPage`); everything else is
  feature-internal. Likewise `services/api/index.ts`.
- **No `fetch` outside `services/`.** Components and hooks call service
  functions, never the network directly.

## SOLID

- **Single responsibility** — one file, one concern. Each dashboard card
  (`ActiveProfileCard`, `ProfilesCard`, `ServicesCard`) renders one thing; the
  container composes them.
- **Open/closed** — add a new endpoint by adding a module under `services/api`
  and a domain type; reuse `apiGet`/`useAsync` rather than editing them. Add a
  card by writing a component and composing it — don't grow a god-component.
- **Liskov** — a `ui/` primitive is interchangeable with the native element it
  wraps; it forwards refs and spreads native props.
- **Interface segregation** — component props expose only what the component
  needs (`ProfilesCard` takes `profiles`, not the whole `Status`).
- **Dependency inversion** — high-level code depends on abstractions:
  `useStatus` depends on the generic `useAsync` + a `getStatus` service, not on
  `fetch`. Pass data in via props/args; don't reach for globals.

## File & naming conventions

- **Components:** `PascalCase.tsx`, one component per file, named export
  matching the filename. Default export only for the app shell and shadcn
  convention (`Button`).
- **Hooks:** `useThing.ts`, `camelCase`, prefixed `use`.
- **Services / utils / types:** `camelCase.ts`.
- **Imports:** use the `@/` alias for cross-layer imports (`@/services/api`);
  relative imports only within the same feature/folder.
- **Import order:** external packages, then `@/` modules, then relative — each
  group separated by a blank line.

## Components & styling

- **Prefer composing shadcn primitives** over hand-rolled markup. Add new ones
  with `npx shadcn@latest add <name>` (they land in `src/components/ui`).
- **Style with Tailwind utility classes**, merged through `cn()` so consumers
  can override. Never inline `style={{…}}` for static styling.
- **Use semantic theme tokens** (`bg-background`, `text-muted-foreground`,
  `border-border`, `bg-success`) — never raw hex or `bg-[#fff]`. Adjust the
  palette in `src/index.css` only; light/dark both derive from those tokens.
- **Accessibility:** every interactive element is keyboard reachable and
  labelled; decorative icons get `aria-hidden`; use semantic HTML
  (`header`, `main`, `ul`).

## State & data fetching

- Model async resources with the discriminated `AsyncState<T>`
  (`loading | error | success`) from `useAsync` — render each state explicitly
  (skeleton, error alert, data). No `isLoading` + `data` boolean soup.
- Always thread the `AbortSignal` from `useAsync` into the request so in-flight
  fetches cancel on unmount.
- Keep state local to where it's used; lift only when shared. No global store
  until two features genuinely need the same state.

## Errors

- The transport layer throws a typed `ApiError(status, message)`; UI presents
  it. Surface clear, actionable messages (what failed + what to do, e.g. "make
  sure `easy-infra serve` is running").

## TypeScript

- `strict` is on and stays on; no `any`. Use `unknown` + narrowing at
  boundaries (see `useAsync`'s catch).
- Prefer `interface` for object/props shapes, `type` for unions/aliases.
- Type-only imports use `import type { … }`.

## Definition of done

A frontend change is not done until:

```sh
npm run build      # tsc -b (typecheck) + vite build — must pass clean
```

Build before committing; `ui/dist` is embedded into the Go binary, so a broken
build breaks `easy-infra ui`.
