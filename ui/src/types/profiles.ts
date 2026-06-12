// Domain model for the profiles screen. Mirrors the JSON contract exposed by
// the /api/profiles endpoints of `easy-infra serve`.

import type { Profile } from "./status";

/** The project's profiles plus which one is currently active. */
export interface ProfilesResult {
  activeProfile: string;
  profiles: Profile[];
}

/**
 * One service instance within a profile: its unique `id`, its service `type`,
 * its display `name`, and its environment config (a free-form string→value map
 * of host, port, credentials, …, mirroring the backend's `service.Config`). A
 * profile may hold several instances of the same type, so `id` is the stable
 * identifier.
 */
export interface ProfileServiceConfig {
  id: string;
  type: string;
  name: string;
  config: Record<string, unknown>;
}

/** A single profile's per-service environment configuration. */
export interface ProfileConfig {
  name: string;
  services: ProfileServiceConfig[];
}
