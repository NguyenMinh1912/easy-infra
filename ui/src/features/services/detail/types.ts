import type { ServiceInstance } from "@/types/service";

/**
 * Props every service overview panel receives. Overviews are read-only and
 * service-specific (see {@link overviewFor}); the editable config lives in the
 * profile settings screen instead.
 */
export interface OverviewProps {
  service: ServiceInstance;
  /**
   * Set when the page is profile-scoped (`#/profiles/{p}/services/{s}`).
   * Profile-aware panels (e.g. the postgres console) only render with it —
   * the connection env is per-profile.
   */
  profile?: string;
}
