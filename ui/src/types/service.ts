// Domain models for managing project service definitions. These mirror the
// JSON contract exposed by `easy-infra serve` under /api/services and are the
// single source of truth for the shapes flowing through the services feature.

/** A service's project-level definition: a free-form string→value map. */
export type ServiceConfig = Record<string, unknown>;

/** One project service definition (easy-infra.yml). */
export interface ServiceDefinition {
  name: string;
  definition: ServiceConfig;
}

/** Response of GET /api/services. */
export interface ServicesResponse {
  initialized: boolean;
  services: ServiceDefinition[];
}

/** A service easy-infra supports, with the definition used when adding it. */
export interface CatalogEntry {
  name: string;
  defaultDefinition: ServiceConfig;
}

/** Response of GET /api/services/catalog. */
export interface CatalogResponse {
  services: CatalogEntry[];
}
