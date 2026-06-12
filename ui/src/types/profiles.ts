// Domain model for the profiles screen. Mirrors the JSON contract exposed by
// the /api/profiles endpoints of `easy-infra serve`.

import type { Profile } from "./status";

/** The project's profiles plus which one is currently active. */
export interface ProfilesResult {
  activeProfile: string;
  profiles: Profile[];
}
