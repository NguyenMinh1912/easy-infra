// Services endpoints. Translate the /api/services REST surface into the domain
// types. Components and hooks depend on these functions, not on `fetch`.

import type {
  CatalogResponse,
  ServiceConfig,
  ServiceDefinition,
  ServicesResponse,
} from "@/types/service";
import { apiGet, apiSend } from "./client";

/** Fetch the project's service definitions. */
export function listServices(signal?: AbortSignal): Promise<ServicesResponse> {
  return apiGet<ServicesResponse>("/services", signal);
}

/** Fetch the catalog of services easy-infra supports. */
export function getServiceCatalog(signal?: AbortSignal): Promise<CatalogResponse> {
  return apiGet<CatalogResponse>("/services/catalog", signal);
}

/** Add a service to the project using its default definition. */
export function createService(name: string): Promise<ServiceDefinition> {
  return apiSend<ServiceDefinition>("POST", "/services", { name });
}

/** Replace a service's project-level definition. */
export function updateService(
  name: string,
  definition: ServiceConfig,
): Promise<ServiceDefinition> {
  return apiSend<ServiceDefinition>("PUT", `/services/${encodeURIComponent(name)}`, {
    definition,
  });
}

/** Remove a service from the project. */
export function deleteService(name: string): Promise<void> {
  return apiSend<void>("DELETE", `/services/${encodeURIComponent(name)}`);
}
