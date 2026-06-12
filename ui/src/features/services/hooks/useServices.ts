import { useCallback, useState } from "react";

import { useAsync } from "@/hooks/useAsync";
import { getServiceCatalog, listServices } from "@/services/api";
import type { CatalogEntry, ServiceDefinition } from "@/types/service";

/** Combined data backing the services screen. */
export interface ServicesData {
  initialized: boolean;
  services: ServiceDefinition[];
  catalog: CatalogEntry[];
}

/**
 * Load the project's service definitions together with the catalog of
 * available services. Returns the async state plus a `reload` callback so
 * mutations (add/edit/remove) can refresh the screen after they succeed.
 */
export function useServices() {
  const [reloadKey, setReloadKey] = useState(0);
  const reload = useCallback(() => setReloadKey((k) => k + 1), []);

  const state = useAsync<ServicesData>(async (signal) => {
    const [list, catalog] = await Promise.all([
      listServices(signal),
      getServiceCatalog(signal),
    ]);
    return {
      initialized: list.initialized,
      services: list.services,
      catalog: catalog.services,
    };
    // reloadKey is the intended trigger to refetch.
  }, [reloadKey]);

  return { state, reload };
}
