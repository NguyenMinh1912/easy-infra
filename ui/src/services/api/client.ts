// Transport layer: a single typed wrapper around `fetch` for the easy-infra
// JSON API. Every endpoint module (status.ts, profiles.ts, …) goes through
// these helpers so that base URL, error handling, and JSON parsing live in
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

/** Issue a request and fail loudly on a network or non-2xx error. */
async function request(path: string, init?: RequestInit): Promise<Response> {
  let res: Response;
  try {
    res = await fetch(`${API_BASE}${path}`, init);
  } catch (cause) {
    throw new ApiError(0, `network error: ${String(cause)}`);
  }

  if (!res.ok) {
    // The API reports failures as a plain-text body; surface it so the UI can
    // show the actionable message rather than a bare status code.
    const detail = (await res.text().catch(() => "")).trim();
    throw new ApiError(
      res.status,
      detail || `request to ${path} failed (${res.status})`,
    );
  }

  return res;
}

/** GET `${API_BASE}${path}` and parse the JSON body as `T`. */
export async function apiGet<T>(path: string, signal?: AbortSignal): Promise<T> {
  const res = await request(path, { signal });
  return (await res.json()) as T;
}

/**
 * POST `${API_BASE}${path}` with an optional JSON body and parse the JSON
 * response as `T`. An empty (e.g. 204) response resolves to `undefined`.
 */
export async function apiPost<T>(
  path: string,
  body?: unknown,
  signal?: AbortSignal,
): Promise<T> {
  const res = await request(path, {
    method: "POST",
    headers: body === undefined ? undefined : { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
    signal,
  });
  const text = await res.text();
  return (text ? JSON.parse(text) : undefined) as T;
}

/** DELETE `${API_BASE}${path}`. */
export async function apiDelete(path: string, signal?: AbortSignal): Promise<void> {
  await request(path, { method: "DELETE", signal });
}
