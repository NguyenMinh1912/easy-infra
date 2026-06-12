// Services endpoints. Services belong to a profile, so create/update/delete are
// scoped to a profile name; the catalog is project-wide. Components and hooks
// depend on these functions, not on `fetch`.

import type { ProfileConfig } from "@/types/profiles";
import type { CatalogResponse, ServiceConfig } from "@/types/service";
import { apiGet, apiSend } from "./client";

/** Build the profile-scoped services path, encoding both segments. */
function servicesPath(profile: string, service?: string): string {
  const base = `/profiles/${encodeURIComponent(profile)}/services`;
  return service ? `${base}/${encodeURIComponent(service)}` : base;
}

/**
 * Append a `?profile=` query so a per-service action targets the profile the UI
 * is viewing rather than whatever profile is active server-side. Omitting the
 * profile (undefined) keeps the active-profile default for non-UI callers.
 */
function withProfile(path: string, profile?: string): string {
  if (!profile) return path;
  const sep = path.includes("?") ? "&" : "?";
  return `${path}${sep}profile=${encodeURIComponent(profile)}`;
}

/** Fetch the catalog of services easy-infra supports. */
export function getServiceCatalog(signal?: AbortSignal): Promise<CatalogResponse> {
  return apiGet<CatalogResponse>("/services/catalog", signal);
}

/**
 * Add an instance of a service `type` to a profile, optionally with a display
 * `name` and a starting `config`; returns the profile. The backend assigns the
 * new instance a unique id, so a profile may hold several of the same type.
 */
export function createService(
  profile: string,
  type: string,
  name?: string,
  config?: ServiceConfig,
): Promise<ProfileConfig> {
  return apiSend<ProfileConfig>("POST", servicesPath(profile), {
    type,
    name,
    config,
  });
}

/**
 * Replace a service instance's config within a profile (identified by its
 * `id`), optionally renaming it; returns the profile.
 */
export function updateService(
  profile: string,
  id: string,
  config: ServiceConfig,
  name?: string,
): Promise<ProfileConfig> {
  return apiSend<ProfileConfig>("PUT", servicesPath(profile, id), {
    config,
    name,
  });
}

/** Remove a service instance from a profile, identified by its `id`. */
export function deleteService(profile: string, id: string): Promise<void> {
  return apiSend<void>("DELETE", servicesPath(profile, id));
}

// --- Backup sessions ---------------------------------------------------------
//
// A backup runs server-side as a persisted session, decoupled from the request
// that started it. The UI starts one, then polls for status + new log lines
// (rather than holding a stream), so the browser can disconnect and reconnect.

/** Lifecycle state of a backup session, mirroring the server's statuses. */
export type BackupStatus =
  | "running"
  | "success"
  | "unsupported"
  | "error"
  | "cancelled";

/**
 * What a session is doing, mirroring the server's kinds: a plain backup, an
 * apply (restore from a snapshot), or a fork (stand up a local container seeded
 * from a snapshot).
 */
export type BackupKind = "backup" | "apply" | "fork";

/** A backup run as reported by the API. */
export interface BackupSession {
  id: string;
  service: string;
  profile: string;
  kind: BackupKind;
  status: BackupStatus;
  snapshot?: string;
  error?: string;
  createdAt: string;
  updatedAt: string;
}

/** One verbose log line, keyed by a per-session sequence number. */
export interface BackupLog {
  seq: number;
  line: string;
}

/** Response of GET /api/backups/{id}: current session plus new log lines. */
export interface BackupPoll {
  session: BackupSession;
  logs: BackupLog[];
}

/** Response of GET /api/backups: a page of sessions plus pagination info. */
export interface BackupList {
  /** False when the folder has no easy-infra project; sessions is then empty. */
  initialized: boolean;
  sessions: BackupSession[];
  /** Total sessions across all pages. */
  total: number;
  /** 1-based page number echoed back by the server. */
  page: number;
  pageSize: number;
}

/**
 * Response of GET /api/services/{name}/backup-options: the buckets a backup can
 * capture (`buckets`) and the subset selected by default (`selected`). Both are
 * empty for services without a bucket concept — the UI then offers a plain
 * confirmation. `error` carries a store-unreachable reason without failing.
 */
export interface BackupOptions {
  buckets: string[];
  selected: string[];
  error?: string;
}

/**
 * Fetch the buckets a service's backup can capture and the default selection,
 * for the given profile (defaulting to the active one). Returns empty lists for
 * services that have no buckets, so callers can fall back to a plain "back up
 * everything" flow.
 */
export function getBackupOptions(
  name: string,
  profile?: string,
  signal?: AbortSignal,
): Promise<BackupOptions> {
  return apiGet<BackupOptions>(
    withProfile(`/services/${encodeURIComponent(name)}/backup-options`, profile),
    signal,
  );
}

/**
 * Start (or re-attach to) a background backup of a service for the given profile
 * (defaulting to the active one). `buckets` optionally narrows the backup to
 * those buckets (minio); omitting it (or passing an empty list) backs up
 * everything. If one is already running for the service, the server returns that
 * session instead of starting a second.
 */
export function startServiceBackup(
  name: string,
  buckets?: string[],
  profile?: string,
): Promise<BackupSession> {
  return apiSend<BackupSession>(
    "POST",
    withProfile(`/services/${encodeURIComponent(name)}/backup`, profile),
    buckets && buckets.length > 0 ? { buckets } : undefined,
  );
}

/**
 * Fetch a backup's current status and the log lines after `after` (the highest
 * seq seen so far), so polling only transfers new output.
 */
export function getBackup(
  id: string,
  after: number,
  signal?: AbortSignal,
): Promise<BackupPoll> {
  return apiGet<BackupPoll>(
    `/backups/${encodeURIComponent(id)}?after=${after}`,
    signal,
  );
}

/** Cancel a running backup; the session settles to "cancelled" shortly after. */
export function cancelBackup(id: string): Promise<BackupSession> {
  return apiSend<BackupSession>(
    "POST",
    `/backups/${encodeURIComponent(id)}/cancel`,
  );
}

/** Fetch a page of backup sessions (newest first) across all services. */
export function listBackups(
  page: number,
  pageSize: number,
  signal?: AbortSignal,
): Promise<BackupList> {
  return apiGet<BackupList>(`/backups?page=${page}&pageSize=${pageSize}`, signal);
}

/**
 * Delete a finished backup session, its logs, and its snapshot on disk. The
 * server rejects deleting a running session — cancel it first.
 */
export function deleteBackup(id: string): Promise<void> {
  return apiSend<void>("DELETE", `/backups/${encodeURIComponent(id)}`);
}

// --- Apply / restore ---------------------------------------------------------
//
// Apply restores a service from a backup snapshot. Like a backup it runs as a
// persisted server-side session: the UI starts one (choosing which version to
// restore) and polls it through the same /api/backups/{id} surface.

/**
 * Response of GET /api/services/{name}/snapshots: the backup versions available
 * to the active profile, newest first.
 */
export interface SnapshotsResponse {
  snapshots: string[];
}

/** List the backup versions a service can be restored from, newest first. */
export function listSnapshots(
  name: string,
  profile?: string,
  signal?: AbortSignal,
): Promise<SnapshotsResponse> {
  return apiGet<SnapshotsResponse>(
    withProfile(`/services/${encodeURIComponent(name)}/snapshots`, profile),
    signal,
  );
}

/**
 * Start (or re-attach to) a background apply of a service for the given profile
 * (defaulting to the active one), restoring from `snapshot` (an empty string
 * means the latest). If one is already running for the service, the server
 * returns that session instead of starting a second.
 */
export function startServiceApply(
  name: string,
  snapshot: string,
  profile?: string,
): Promise<BackupSession> {
  return apiSend<BackupSession>(
    "POST",
    withProfile(`/services/${encodeURIComponent(name)}/apply`, profile),
    { snapshot },
  );
}

// --- Fork to local -----------------------------------------------------------
//
// Fork stands the active profile's service up as a local Docker container with
// the same configuration and seeds it from a backup. Like backup/apply it runs
// as a persisted server-side session polled through /api/backups/{id}; an empty
// snapshot tells the server to take a fresh backup of the source first.

/**
 * Start (or re-attach to) a background fork of a service from the given source
 * profile (defaulting to the active one) to a local container, seeded from
 * `snapshot`. An empty `snapshot` means "create a new backup of the source
 * first, then fork from it". An optional `port` publishes the local container on
 * a different port than the source; omitting it keeps the source's port.
 */
export function startServiceFork(
  name: string,
  snapshot: string,
  port?: number,
  profile?: string,
): Promise<BackupSession> {
  return apiSend<BackupSession>(
    "POST",
    withProfile(`/services/${encodeURIComponent(name)}/fork`, profile),
    port ? { snapshot, port } : { snapshot },
  );
}
