// Transport layer: a single typed wrapper around `fetch` for the easy-infra
// JSON API. Every endpoint module (status.ts, services.ts, …) goes through
// `apiGet`/`apiSend` so that base URL, error handling, and JSON parsing live in
// exactly one place.

/** Error thrown when the API responds with a non-2xx status. */
export class ApiError extends Error {
  constructor(
    readonly status: number,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

const API_BASE = "/api";

/** GET `${API_BASE}${path}` and parse the JSON body as `T`. */
export async function apiGet<T>(path: string, signal?: AbortSignal): Promise<T> {
  return request<T>("GET", path, undefined, signal);
}

/**
 * Send a mutating request (`POST`/`PUT`/`DELETE`) with an optional JSON body
 * and parse the JSON response as `T`. A 204 (No Content) resolves to
 * `undefined`, so callers expecting no body should use `apiSend<void>`.
 */
export async function apiSend<T>(
  method: "POST" | "PUT" | "DELETE",
  path: string,
  body?: unknown,
  signal?: AbortSignal,
): Promise<T> {
  return request<T>(method, path, body, signal);
}

async function request<T>(
  method: string,
  path: string,
  body: unknown,
  signal?: AbortSignal,
): Promise<T> {
  let res: Response;
  try {
    res = await fetch(`${API_BASE}${path}`, {
      method,
      signal,
      headers: body === undefined ? undefined : { "Content-Type": "application/json" },
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch (cause) {
    throw new ApiError(0, `network error: ${String(cause)}`);
  }

  if (!res.ok) {
    throw new ApiError(res.status, await errorMessage(res, path));
  }

  if (res.status === 204) {
    return undefined as T;
  }
  return (await res.json()) as T;
}

/**
 * Build a human-readable message for a failed response, preferring the API's
 * `{ "error": "..." }` envelope and falling back to the status code.
 */
async function errorMessage(res: Response, path: string): Promise<string> {
  try {
    const data = (await res.json()) as { error?: unknown };
    if (typeof data.error === "string" && data.error) {
      return data.error;
    }
  } catch {
    // Non-JSON body; fall through to the generic message.
  }
  return `request to ${path} failed (${res.status})`;
}
