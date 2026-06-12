// Object-browser endpoints: list a profile's object-store buckets and walk the
// folder-organised objects within them.

import type { BucketsResponse, ObjectListing } from "@/types/browse";

import { apiGet } from "./client";

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
