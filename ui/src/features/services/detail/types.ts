import type { ServiceDefinition } from "@/types/service";

/**
 * Props every service overview panel receives. Overviews are read-only and
 * service-specific (see {@link overviewFor}); the editable definition lives in
 * the shared Configuration panel instead.
 */
export interface OverviewProps {
  service: ServiceDefinition;
}
