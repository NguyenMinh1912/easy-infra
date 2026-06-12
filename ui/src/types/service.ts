// Domain models for managing a profile's services. These mirror the JSON
// contract exposed by `easy-infra serve` and are the single source of truth for
// the shapes flowing through the services feature.
//
// Services belong to profiles: each profile owns its own services, and each
// service block is a single merged config — both what the service is (e.g.
// `version`, `cleanable`) and how to reach it (host, port, credentials).

/** A service's config: a free-form string→value map. */
export type ServiceConfig = Record<string, unknown>;

/**
 * One service instance within a profile. A profile may hold several instances
 * of the same type, so each carries:
 *
 *   - `id` — the stable, unique identifier within the profile (used in routes
 *     and per-instance API paths);
 *   - `type` — the service type (e.g. "postgres"), which drives behaviour,
 *     icons, and the per-service config/overview;
 *   - `name` — a user-facing label that can be renamed freely;
 *   - `config` — the merged definition + environment block.
 */
export interface ServiceInstance {
  id: string;
  type: string;
  name: string;
  config: ServiceConfig;
}

/**
 * A service easy-infra supports, with the default config used when adding it.
 * `name` is the canonical service type (the registry key).
 */
export interface CatalogEntry {
  name: string;
  defaultConfig: ServiceConfig;
}

/** Response of GET /api/services/catalog. */
export interface CatalogResponse {
  services: CatalogEntry[];
}
