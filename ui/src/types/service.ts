// Domain models for managing a profile's services. These mirror the JSON
// contract exposed by `easy-infra serve` and are the single source of truth for
// the shapes flowing through the services feature.
//
// Services belong to profiles: each profile owns its own services, and each
// service block is a single merged config — both what the service is (e.g.
// `version`, `cleanable`) and how to reach it (host, port, credentials).

/** A service's config: a free-form string→value map. */
export type ServiceConfig = Record<string, unknown>;

/** One service within a profile: its name and merged config block. */
export interface ServiceInstance {
  name: string;
  config: ServiceConfig;
}

/** A service easy-infra supports, with the default config used when adding it. */
export interface CatalogEntry {
  name: string;
  defaultConfig: ServiceConfig;
}

/** Response of GET /api/services/catalog. */
export interface CatalogResponse {
  services: CatalogEntry[];
}
