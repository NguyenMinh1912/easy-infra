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

/** Terminal outcome of a backup stream. */
export interface BackupResult {
  /** "ok" when the snapshot was written, "unsupported" for a service whose backup is not wired up yet. */
  status: "ok" | "unsupported";
  /** Snapshot folder name, present when status is "ok". */
  snapshot?: string;
}

/** Callbacks driven by {@link streamServiceBackup} as the SSE stream arrives. */
export interface BackupStreamHandlers {
  /** One verbose log line emitted by the service. */
  onLog: (line: string) => void;
  /** Backup finished; carries the terminal status. */
  onDone: (result: BackupResult) => void;
  /** Backup failed (server error, network drop, or abort). */
  onError: (message: string) => void;
}

/**
 * Back up a single service for the active profile, streaming the server's
 * verbose log via Server-Sent Events. EventSource is GET-only, so this uses
 * `fetch` with a streamed response body and parses the SSE frames itself,
 * dispatching each to the handlers. Pass an AbortSignal to cancel an in-flight
 * backup; aborting surfaces through `onError`.
 */
export async function streamServiceBackup(
  name: string,
  handlers: BackupStreamHandlers,
  signal?: AbortSignal,
): Promise<void> {
  let res: Response;
  try {
    res = await fetch(`/api/services/${encodeURIComponent(name)}/backup`, {
      method: "POST",
      signal,
    });
  } catch (cause) {
    handlers.onError(`network error: ${String(cause)}`);
    return;
  }

  if (!res.ok || !res.body) {
    handlers.onError(await backupErrorMessage(res));
    return;
  }

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  try {
    for (;;) {
      const { done, value } = await reader.read();
      if (done) break;
      buffer += decoder.decode(value, { stream: true });
      // SSE frames are separated by a blank line.
      let sep: number;
      while ((sep = buffer.indexOf("\n\n")) !== -1) {
        const frame = buffer.slice(0, sep);
        buffer = buffer.slice(sep + 2);
        dispatchFrame(frame, handlers);
      }
    }
  } catch (cause) {
    handlers.onError(
      signal?.aborted ? "backup cancelled" : `stream error: ${String(cause)}`,
    );
  }
}

/** Parse one SSE frame (`event:`/`data:` lines) and route it to a handler. */
function dispatchFrame(frame: string, handlers: BackupStreamHandlers): void {
  let event = "message";
  const dataLines: string[] = [];
  for (const line of frame.split("\n")) {
    if (line.startsWith("event:")) {
      event = line.slice(6).trim();
    } else if (line.startsWith("data:")) {
      dataLines.push(line.slice(5).replace(/^ /, ""));
    }
  }
  const data = dataLines.join("\n");

  switch (event) {
    case "log":
      handlers.onLog(data);
      break;
    case "done":
      handlers.onDone(safeParse<BackupResult>(data) ?? { status: "ok" });
      break;
    case "error":
      handlers.onError(safeParse<{ error?: string }>(data)?.error ?? "backup failed");
      break;
  }
}

function safeParse<T>(data: string): T | undefined {
  try {
    return JSON.parse(data) as T;
  } catch {
    return undefined;
  }
}

/** Read the JSON error envelope from a failed pre-stream response. */
async function backupErrorMessage(res: Response): Promise<string> {
  try {
    const data = (await res.json()) as { error?: unknown };
    if (typeof data.error === "string" && data.error) {
      return data.error;
    }
  } catch {
    // Non-JSON body; fall through to the generic message.
  }
  return `backup request failed (${res.status})`;
}
