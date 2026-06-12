import type { ServiceDefinition } from "@/types/service";

/**
 * Props every service overview panel receives. Overviews are read-only and
 * service-specific (see {@link overviewFor}); the editable definition lives in
 * the shared Configuration panel instead.
 */
export interface OverviewProps {
  service: ServiceDefinition;
  /**
   * Set when the page is profile-scoped (`#/profiles/{p}/services/{s}`).
   * Profile-aware panels (e.g. the postgres console) only render with it —
   * the connection env is per-profile.
   */
  profile?: string;
}
