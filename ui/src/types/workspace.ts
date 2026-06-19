// Domain models for workspaces — the named bundles of profiles the app manages.
// All data lives in the central store; a workspace is a record, not a folder.
// These mirror the JSON contract exposed by the /api/workspaces endpoints.

export interface Workspace {
  id: number;
  name: string;
}

export interface WorkspacesResult {
  /** Id of the active workspace (0 when none). */
  active: number;
  workspaces: Workspace[];
}
