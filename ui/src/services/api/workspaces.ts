// Workspace endpoints. Translate the /api/workspaces REST surface into the
// domain types. Components and hooks depend on these functions, not on `fetch`.

import type { DirListing, WorkspacesResult } from "@/types/workspace";
import { apiGet, apiSend } from "./client";

/** List the known workspaces and the active one. */
export function getWorkspaces(signal?: AbortSignal): Promise<WorkspacesResult> {
  return apiGet<WorkspacesResult>("/workspaces", signal);
}

/**
 * Create (or adopt) a workspace at `path` and make it active. The backend
 * scaffolds a project when the folder is not already one. Returns the updated
 * list.
 */
export function createWorkspace(
  name: string,
  path: string,
): Promise<WorkspacesResult> {
  return apiSend<WorkspacesResult>("POST", "/workspaces", { name, path });
}

/** Switch the active workspace; returns the updated list. */
export function activateWorkspace(name: string): Promise<WorkspacesResult> {
  return apiSend<WorkspacesResult>("POST", "/workspaces/activate", { name });
}

/** Remove a workspace from the registry (leaves files on disk). */
export function removeWorkspace(name: string): Promise<WorkspacesResult> {
  return apiSend<WorkspacesResult>(
    "DELETE",
    `/workspaces/${encodeURIComponent(name)}`,
  );
}

/**
 * List the subdirectories of a folder so the user can navigate the server's
 * filesystem. An empty `path` defaults to the user's home directory.
 */
export function browseDirs(
  path?: string,
  signal?: AbortSignal,
): Promise<DirListing> {
  const query = path ? `?path=${encodeURIComponent(path)}` : "";
  return apiGet<DirListing>(`/workspaces/browse${query}`, signal);
}
