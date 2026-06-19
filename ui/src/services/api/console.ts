// Console endpoints: execute SQL against a profile's service and fetch its
// schema for editor autocomplete.

import type { QueryResult, SchemaInfo } from "@/types/console";

import { apiGet, apiSend } from "./client";

/**
 * Execute one statement against the named profile's service. Statement
 * failures (bad SQL, unreachable database) resolve successfully with `error`
 * set on the result — only transport/protocol problems reject.
 *
 * `db` optionally overrides the logical database the statement runs against
 * (Redis only), so the console can target a database other than the profile's
 * saved one — matching the key browser's database selector.
 */
export async function executeQuery(
  profile: string,
  service: string,
  sql: string,
  signal?: AbortSignal,
  db?: number,
): Promise<QueryResult> {
  return apiSend<QueryResult>(
    "POST",
    `/profiles/${encodeURIComponent(profile)}/services/${encodeURIComponent(service)}/query`,
    db === undefined ? { sql } : { sql, db },
    signal,
  );
}

/** Fetch the service's queryable tables/columns for autocomplete. */
export async function getSchema(
  profile: string,
  service: string,
  signal?: AbortSignal,
): Promise<SchemaInfo> {
  return apiGet<SchemaInfo>(
    `/profiles/${encodeURIComponent(profile)}/services/${encodeURIComponent(service)}/schema`,
    signal,
  );
}
