// Domain model for the profiles screen. Mirrors the JSON contract exposed by
// the /api/profiles endpoints of `easy-infra serve`.

import type { Profile } from "./status";

/** The project's profiles plus which one is currently active. */
export interface ProfilesResult {
  activeProfile: string;
  profiles: Profile[];
}

/**
 * One service's environment config within a profile: a free-form string→value
 * map (host, port, credentials, …), mirroring the backend's `service.Config`.
 */
export interface ProfileServiceConfig {
  name: string;
  config: Record<string, unknown>;
}

/** A single profile's per-service environment configuration. */
export interface ProfileConfig {
  name: string;
  services: ProfileServiceConfig[];
}
