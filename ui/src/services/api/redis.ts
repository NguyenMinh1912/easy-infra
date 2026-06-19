// Redis key-browser endpoints: report the logical-database count, scan a
// profile's keyspace, and read one key's value. Listing failures (server
// unreachable) resolve successfully with `error` set on the result — only
// transport/protocol problems reject.

import type {
  DatabasesResponse,
  KeysResponse,
  KeyValue,
} from "@/types/redis";

import { apiGet } from "./client";

const base = (profile: string, service: string) =>
  `/profiles/${encodeURIComponent(profile)}/services/${encodeURIComponent(service)}`;

/** Fetch the server's logical-database count, for the database selector. */
export async function getDatabases(
  profile: string,
  service: string,
  signal?: AbortSignal,
): Promise<DatabasesResponse> {
  return apiGet<DatabasesResponse>(`${base(profile, service)}/databases`, signal);
}

/** Scan one page of the keyspace in database `db` matching `pattern`. */
export async function listKeys(
  profile: string,
  service: string,
  db: number,
  pattern: string,
  cursor: number,
  signal?: AbortSignal,
): Promise<KeysResponse> {
  const query = new URLSearchParams({
    db: String(db),
    pattern,
    cursor: String(cursor),
  });
  return apiGet<KeysResponse>(`${base(profile, service)}/keys?${query}`, signal);
}

/** Read one key's value (shaped by its type) from database `db`. */
export async function getKeyValue(
  profile: string,
  service: string,
  db: number,
  key: string,
  signal?: AbortSignal,
): Promise<KeyValue> {
  const query = new URLSearchParams({ db: String(db), key });
  return apiGet<KeyValue>(`${base(profile, service)}/key?${query}`, signal);
}
