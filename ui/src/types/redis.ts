// Domain types for the Redis key browser, mirroring the
// /api/profiles/{name}/services/{service}/databases, /keys and /key JSON
// contracts. A failing listing (server unreachable) is an expected outcome:
// the API responds 200 with `error` set and the payload empty.

/** One key with its summary metadata. */
export interface KeyEntry {
  name: string;
  type: string;
  /** TTL in seconds: -1 when the key has no expiry, -2 when it is missing. */
  ttl: number;
}

/** Response of GET …/databases — the server's logical-database count. */
export interface DatabasesResponse {
  count: number;
  error?: string;
}

/** Response of GET …/keys — one SCAN page. */
export interface KeysResponse {
  keys: KeyEntry[];
  /** Continues the scan; 0 means the keyspace has been fully walked. */
  cursor: number;
  error?: string;
}

/** One field/value pair of a hash value. */
export interface HashField {
  field: string;
  value: string;
}

/** One member of a sorted set with its score. */
export interface ZSetMember {
  member: string;
  score: number;
}

/**
 * One key's value, shaped by its Redis type. Exactly one of the type-specific
 * fields is populated for a known type; an unsupported type (e.g. stream)
 * carries only the metadata.
 */
export interface KeyValue {
  key: string;
  type: string;
  /** TTL in seconds: -1 no expiry, -2 the key is missing. */
  ttl: number;
  string?: string;
  list?: string[];
  set?: string[];
  hash?: HashField[];
  zset?: ZSetMember[];
  /** Full element count of a collection (or the string's byte length). */
  length: number;
  /** True when the value was longer than the page returned. */
  truncated: boolean;
  error?: string;
}
