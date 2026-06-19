import { useCallback, useMemo, useState } from "react";

import { useAsync, type AsyncState } from "@/hooks/useAsync";
import {
  activateWorkspace,
  createWorkspace,
  getWorkspaces,
  removeWorkspace,
} from "@/services/api";
import type { WorkspacesResult } from "@/types/workspace";

/** Mutations the workspace switcher can perform. */
export interface WorkspaceActions {
  activate: (name: string) => Promise<void>;
  create: (name: string, path: string) => Promise<void>;
  remove: (name: string) => Promise<void>;
}

/**
 * Load the known workspaces and expose actions to switch, create, and remove
 * them. Switching or creating changes the folder behind every screen, so the
 * caller is expected to reload the app after a successful activate/create — the
 * switcher does this via a full reload (see {@link reloadApp}).
 */
export function useWorkspaces(): {
  state: AsyncState<WorkspacesResult>;
  actions: WorkspaceActions;
} {
  const [nonce, setNonce] = useState(0);
  const reload = useCallback(() => setNonce((n) => n + 1), []);
  const state = useAsync<WorkspacesResult>((signal) => getWorkspaces(signal), [
    nonce,
  ]);

  const actions = useMemo<WorkspaceActions>(
    () => ({
      activate: async (name) => {
        await activateWorkspace(name);
      },
      create: async (name, path) => {
        await createWorkspace(name, path);
      },
      remove: async (name) => {
        await removeWorkspace(name);
        reload();
      },
    }),
    [reload],
  );

  return { state, actions };
}
