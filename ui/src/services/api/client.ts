// Transport layer: a single typed wrapper around `fetch` for the easy-infra
// JSON API. Every endpoint module (status.ts, …) goes through `apiGet` so that
// base URL, error handling, and JSON parsing live in exactly one place.

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
  let res: Response;
  try {
    res = await fetch(`${API_BASE}${path}`, { signal });
  } catch (cause) {
    throw new ApiError(0, `network error: ${String(cause)}`);
  }

  if (!res.ok) {
    throw new ApiError(res.status, `request to ${path} failed (${res.status})`);
  }

  return (await res.json()) as T;
}
