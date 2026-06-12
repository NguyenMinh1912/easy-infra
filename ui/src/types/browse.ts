// Domain types for the object browser (read-only walk of a profile's
// object-store buckets), mirroring the
// /api/profiles/{name}/services/{service}/buckets and /objects JSON contracts.

/**
 * The buckets of a profile's object-store service. A failed listing (e.g.
 * store unreachable) is an expected outcome: the API responds 200 with
 * `error` set and `buckets` empty.
 */
export interface BucketsResponse {
  buckets: string[];
  error?: string;
}

/** One object's key and metadata within a listing. */
export interface ObjectEntry {
  key: string;
  size: number;
  /** Modification time (RFC3339) when known. */
  lastModified?: string;
  contentType?: string;
}

/**
 * One folder level within a bucket: the immediate sub-folders (`prefixes`,
 * each a full key ending in "/") and the objects directly at that level.
 * Listing failures are reported via `error`, like {@link BucketsResponse}.
 */
export interface ObjectListing {
  prefixes: string[];
  objects: ObjectEntry[];
  error?: string;
}
