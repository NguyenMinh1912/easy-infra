import { useCallback, useMemo, useState } from "react";

import { useAsync, type AsyncState } from "@/hooks/useAsync";
import {
  activateProfile,
  createProfile,
  deleteProfile,
  listProfiles,
} from "@/services/api";
import type { ProfilesResult } from "@/types/profiles";

/** The mutations the profiles screen can perform. Each reloads on success. */
export interface ProfileActions {
  create: (name: string) => Promise<void>;
  activate: (name: string) => Promise<void>;
  remove: (name: string) => Promise<void>;
}

/**
 * Load the project's profiles and expose CRUD actions. Reads go through the
 * generic {@link useAsync}; each successful mutation bumps a nonce so the list
 * reloads. Mutations rethrow on failure so callers can surface the message.
 */
export function useProfiles(): {
  state: AsyncState<ProfilesResult>;
  actions: ProfileActions;
} {
  const [nonce, setNonce] = useState(0);
  const reload = useCallback(() => setNonce((n) => n + 1), []);
  const state = useAsync<ProfilesResult>((signal) => listProfiles(signal), [
    nonce,
  ]);

  const actions = useMemo<ProfileActions>(
    () => ({
      create: async (name) => {
        await createProfile(name);
        reload();
      },
      activate: async (name) => {
        await activateProfile(name);
        reload();
      },
      remove: async (name) => {
        await deleteProfile(name);
        reload();
      },
    }),
    [reload],
  );

  return { state, actions };
}
