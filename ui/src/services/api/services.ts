// Services endpoints. Translate the /api/services REST surface into the domain
// types. Components and hooks depend on these functions, not on `fetch`.

import type {
  CatalogResponse,
  ServiceConfig,
  ServiceDefinition,
  ServicesResponse,
} from "@/types/service";
import { apiGet, apiSend } from "./client";

/** Fetch the project's service definitions. */
export function listServices(signal?: AbortSignal): Promise<ServicesResponse> {
  return apiGet<ServicesResponse>("/services", signal);
}

/** Fetch the catalog of services easy-infra supports. */
export function getServiceCatalog(signal?: AbortSignal): Promise<CatalogResponse> {
  return apiGet<CatalogResponse>("/services/catalog", signal);
}

/** Add a service to the project using its default definition. */
export function createService(name: string): Promise<ServiceDefinition> {
  return apiSend<ServiceDefinition>("POST", "/services", { name });
}

/** Replace a service's project-level definition. */
export function updateService(
  name: string,
  definition: ServiceConfig,
): Promise<ServiceDefinition> {
  return apiSend<ServiceDefinition>("PUT", `/services/${encodeURIComponent(name)}`, {
    definition,
  });
}

/** Remove a service from the project. */
export function deleteService(name: string): Promise<void> {
  return apiSend<void>("DELETE", `/services/${encodeURIComponent(name)}`);
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

/** A backup run as reported by the API. */
export interface BackupSession {
  id: string;
  service: string;
  profile: string;
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
 * Start (or re-attach to) a background backup of a service for the active
 * profile. If one is already running for the service, the server returns that
 * session instead of starting a second.
 */
export function startServiceBackup(name: string): Promise<BackupSession> {
  return apiSend<BackupSession>(
    "POST",
    `/services/${encodeURIComponent(name)}/backup`,
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
  signal?: AbortSignal,
): Promise<SnapshotsResponse> {
  return apiGet<SnapshotsResponse>(
    `/services/${encodeURIComponent(name)}/snapshots`,
    signal,
  );
}

/**
 * Start (or re-attach to) a background apply of a service for the active
 * profile, restoring from `snapshot` (an empty string means the latest). If one
 * is already running for the service, the server returns that session instead
 * of starting a second.
 */
export function startServiceApply(
  name: string,
  snapshot: string,
): Promise<BackupSession> {
  return apiSend<BackupSession>(
    "POST",
    `/services/${encodeURIComponent(name)}/apply`,
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
 * Start (or re-attach to) a background fork of a service from the active
 * profile to a local container, seeded from `snapshot`. An empty `snapshot`
 * means "create a new backup of the source first, then fork from it".
 */
export function startServiceFork(
  name: string,
  snapshot: string,
): Promise<BackupSession> {
  return apiSend<BackupSession>(
    "POST",
    `/services/${encodeURIComponent(name)}/fork`,
    { snapshot },
  );
}
