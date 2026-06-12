import { useCallback, useState } from "react";

import { useAsync } from "@/hooks/useAsync";
import { getProfileConfig, getServiceCatalog, getStatus } from "@/services/api";
import type { CatalogEntry, ServiceInstance } from "@/types/service";

/** Combined data backing the services screen. */
export interface ServicesData {
  initialized: boolean;
  /** Profile whose services are shown (the active one, unless overridden). */
  activeProfile: string;
  services: ServiceInstance[];
  catalog: CatalogEntry[];
}

/**
 * Load the services of a profile together with the catalog of available
 * services. Services belong to a profile, so the screen shows one profile's
 * services: the active profile by default, or `profileOverride` when a screen is
 * profile-scoped (e.g. a service detail reached via the profiles sidebar).
 * Returns the async state plus a `reload` callback so mutations can refresh.
 */
export function useServices(profileOverride?: string) {
  const [reloadKey, setReloadKey] = useState(0);
  const reload = useCallback(() => setReloadKey((k) => k + 1), []);

  const state = useAsync<ServicesData>(
    async (signal) => {
      const status = await getStatus(signal);
      if (!status.initialized) {
        return { initialized: false, activeProfile: "", services: [], catalog: [] };
      }
      const profile = profileOverride ?? status.activeProfile;
      const catalog = await getServiceCatalog(signal);
      const services = profile
        ? (await getProfileConfig(profile, signal)).services
        : [];
      return {
        initialized: true,
        activeProfile: profile,
        services,
        catalog: catalog.services,
      };
      // reloadKey is the intended trigger to refetch.
    },
    [reloadKey, profileOverride],
  );

  return { state, reload };
}
