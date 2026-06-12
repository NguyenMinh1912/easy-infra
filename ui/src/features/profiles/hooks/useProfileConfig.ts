import { useCallback, useMemo, useState } from "react";

import { useAsync, type AsyncState } from "@/hooks/useAsync";
import { getProfileConfig, updateProfileConfig } from "@/services/api";
import type { ProfileConfig, ProfileServiceConfig } from "@/types/profiles";

/** The mutations the profile settings screen can perform. */
export interface ProfileConfigActions {
  /** Replace the whole profile's per-service config; reloads on success. */
  save: (services: ProfileServiceConfig[]) => Promise<void>;
}

/**
 * Load a single profile's per-service environment config and expose a save
 * action. Mirrors {@link useProfiles}: reads go through {@link useAsync} keyed
 * by the profile name, and a successful save bumps a nonce so the config
 * reloads. Save rethrows on failure so callers can surface the message.
 */
export function useProfileConfig(name: string): {
  state: AsyncState<ProfileConfig>;
  actions: ProfileConfigActions;
} {
  const [nonce, setNonce] = useState(0);
  const reload = useCallback(() => setNonce((n) => n + 1), []);
  const state = useAsync<ProfileConfig>(
    (signal) => getProfileConfig(name, signal),
    [name, nonce],
  );

  const actions = useMemo<ProfileConfigActions>(
    () => ({
      save: async (services) => {
        await updateProfileConfig(name, services);
        reload();
      },
    }),
    [name, reload],
  );

  return { state, actions };
}
