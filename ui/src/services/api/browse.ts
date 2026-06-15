// Object-browser endpoints: list a profile's object-store buckets and walk the
// folder-organised objects within them.

import type { BucketsResponse, ObjectListing } from "@/types/browse";

import { ApiError, apiGet, apiSend } from "./client";

/**
 * List the buckets of the named profile's service. Listing failures (e.g.
 * store unreachable) resolve successfully with `error` set on the result —
 * only transport/protocol problems reject.
 */
export async function listBuckets(
  profile: string,
  service: string,
  signal?: AbortSignal,
): Promise<BucketsResponse> {
  return apiGet<BucketsResponse>(
    `/profiles/${encodeURIComponent(profile)}/services/${encodeURIComponent(service)}/buckets`,
    signal,
  );
}

/**
 * List the immediate sub-folders and objects under `prefix` within `bucket`.
 * An empty prefix lists the bucket root.
 */
export async function listObjects(
  profile: string,
  service: string,
  bucket: string,
  prefix: string,
  signal?: AbortSignal,
): Promise<ObjectListing> {
  const query = new URLSearchParams({ bucket, prefix });
  return apiGet<ObjectListing>(
    `/profiles/${encodeURIComponent(profile)}/services/${encodeURIComponent(service)}/objects?${query.toString()}`,
    signal,
  );
}

/**
 * The URL that streams one object's contents as a download. The endpoint sets
 * `Content-Disposition: attachment`, so navigating to it (or following an
 * anchor) saves the file rather than rendering it.
 */
export function objectDownloadUrl(
  profile: string,
  service: string,
  bucket: string,
  key: string,
): string {
  const query = new URLSearchParams({ bucket, key });
  return `/api/profiles/${encodeURIComponent(profile)}/services/${encodeURIComponent(service)}/object?${query.toString()}`;
}

/**
 * Upload one file into `bucket` under `key`, streaming the file's bytes as the
 * request body so the browser never buffers the whole file in memory. The
 * file's MIME type tags the stored object. Resolves on success (204) and
 * rejects with an {@link ApiError} otherwise; pass `signal` to cancel an
 * in-flight upload. Callers upload several files by invoking this once per
 * file, bounding how many run at once to keep the page responsive.
 */
export async function uploadObject(
  profile: string,
  service: string,
  bucket: string,
  key: string,
  file: File,
  signal?: AbortSignal,
): Promise<void> {
  const query = new URLSearchParams({ bucket, key });
  const path = `/api/profiles/${encodeURIComponent(profile)}/services/${encodeURIComponent(service)}/object?${query.toString()}`;
  let res: Response;
  try {
    res = await fetch(path, {
      method: "PUT",
      headers: { "Content-Type": file.type || "application/octet-stream" },
      body: file,
      signal,
    });
  } catch (cause) {
    throw new ApiError(0, `network error: ${String(cause)}`);
  }
  if (!res.ok) {
    let message = `upload of ${file.name} failed (${res.status})`;
    try {
      const data = (await res.json()) as { error?: unknown };
      if (typeof data.error === "string" && data.error) message = data.error;
    } catch {
      // Non-JSON body; keep the generic message.
    }
    throw new ApiError(res.status, message);
  }
}

/**
 * The URL that streams a zip of the selected objects (`keys`) and folders
 * (`prefixes`, expanded recursively server-side) within `bucket`. Like
 * {@link objectDownloadUrl} the endpoint responds as an attachment, so an
 * anchor pointing at it saves the archive.
 */
export function objectsArchiveUrl(
  profile: string,
  service: string,
  bucket: string,
  selection: { keys: string[]; prefixes: string[] },
): string {
  const query = new URLSearchParams({ bucket });
  for (const key of selection.keys) query.append("key", key);
  for (const prefix of selection.prefixes) query.append("prefix", prefix);
  return `/api/profiles/${encodeURIComponent(profile)}/services/${encodeURIComponent(service)}/objects/archive?${query.toString()}`;
}

/**
 * Delete the selected objects (`keys`) and folders (`prefixes`, expanded
 * recursively server-side) from `bucket` — the same selection shape as
 * {@link objectsArchiveUrl}. Resolves on success (204) and rejects with an
 * {@link ApiError} otherwise.
 */
export async function deleteObjects(
  profile: string,
  service: string,
  bucket: string,
  selection: { keys: string[]; prefixes: string[] },
  signal?: AbortSignal,
): Promise<void> {
  const query = new URLSearchParams({ bucket });
  for (const key of selection.keys) query.append("key", key);
  for (const prefix of selection.prefixes) query.append("prefix", prefix);
  return apiSend<void>(
    "DELETE",
    `/profiles/${encodeURIComponent(profile)}/services/${encodeURIComponent(service)}/objects?${query.toString()}`,
    undefined,
    signal,
  );
}
