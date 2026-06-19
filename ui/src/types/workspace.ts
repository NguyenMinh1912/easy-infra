// Domain models for workspaces — the project folders the web UI knows about.
// These mirror the JSON contract exposed by the /api/workspaces endpoints.

export interface Workspace {
  name: string;
  path: string;
  /** False when the folder has been moved or deleted under the tool. */
  exists: boolean;
}

export interface WorkspacesResult {
  /** Name of the active workspace ("" when none). */
  active: string;
  workspaces: Workspace[];
  /** User home directory — a sensible starting point for the folder browser. */
  home: string;
  /** OS path separator, so the UI never hard-codes "/". */
  separator: string;
}

/** One subdirectory in a folder-browse listing. */
export interface DirEntry {
  name: string;
  path: string;
  /** True when the folder already holds an easy-infra project. */
  isProject: boolean;
}

/** Listing of the subdirectories of a folder. `parent` is "" at the FS root. */
export interface DirListing {
  path: string;
  parent: string;
  entries: DirEntry[];
}
