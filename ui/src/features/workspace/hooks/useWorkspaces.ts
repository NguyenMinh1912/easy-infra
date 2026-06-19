import { useCallback, useMemo, useState } from "react";

import { useAsync, type AsyncState } from "@/hooks/useAsync";
import {
  activateWorkspace,
  createWorkspace,
  getWorkspaces,
  importWorkspace,
  removeWorkspace,
  renameWorkspace,
} from "@/services/api";
import type { WorkspacesResult } from "@/types/workspace";

/** Mutations the workspace switcher can perform. */
export interface WorkspaceActions {
  activate: (id: number) => Promise<void>;
  create: (name: string) => Promise<void>;
  importFile: (file: File) => Promise<void>;
  rename: (id: number, name: string) => Promise<void>;
  remove: (id: number) => Promise<void>;
}

/**
 * Load the known workspaces and expose actions to switch, create, rename, and
 * remove them. Switching or creating changes the data behind every screen, so
 * the caller is expected to reload the app after a successful activate/create —
 * the switcher does this via a full reload. Rename and remove only refetch the
 * list (handled here via {@link reload}).
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
      activate: async (id) => {
        await activateWorkspace(id);
      },
      create: async (name) => {
        await createWorkspace(name);
      },
      importFile: async (file) => {
        await importWorkspace(file);
      },
      rename: async (id, name) => {
        await renameWorkspace(id, name);
        reload();
      },
      remove: async (id) => {
        await removeWorkspace(id);
        reload();
      },
    }),
    [reload],
  );

  return { state, actions };
}
