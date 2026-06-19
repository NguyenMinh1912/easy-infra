// Workspace endpoints. Translate the /api/workspaces REST surface into the
// domain types. Components and hooks depend on these functions, not on `fetch`.

import type { WorkspacesResult } from "@/types/workspace";
import { apiGet, apiSend } from "./client";

/** List the known workspaces and the active one. */
export function getWorkspaces(signal?: AbortSignal): Promise<WorkspacesResult> {
  return apiGet<WorkspacesResult>("/workspaces", signal);
}

/**
 * Create a workspace (the backend scaffolds its default profile) and make it
 * active. Returns the updated list.
 */
export function createWorkspace(name: string): Promise<WorkspacesResult> {
  return apiSend<WorkspacesResult>("POST", "/workspaces", { name });
}

/** Rename a workspace by id; returns the updated list. */
export function renameWorkspace(
  id: number,
  name: string,
): Promise<WorkspacesResult> {
  return apiSend<WorkspacesResult>("PUT", `/workspaces/${id}`, { name });
}

/** Switch the active workspace by id; returns the updated list. */
export function activateWorkspace(id: number): Promise<WorkspacesResult> {
  return apiSend<WorkspacesResult>("POST", "/workspaces/activate", { id });
}

/** Remove a workspace by id (and its profiles/services); returns the list. */
export function removeWorkspace(id: number): Promise<WorkspacesResult> {
  return apiSend<WorkspacesResult>("DELETE", `/workspaces/${id}`);
}
