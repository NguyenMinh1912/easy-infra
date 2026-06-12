// Console endpoints: execute SQL against a profile's service and fetch its
// schema for editor autocomplete.

import type { QueryResult, SchemaInfo } from "@/types/console";

import { apiGet, apiSend } from "./client";

/**
 * Execute one statement against the named profile's service. Statement
 * failures (bad SQL, unreachable database) resolve successfully with `error`
 * set on the result — only transport/protocol problems reject.
 */
export async function executeQuery(
  profile: string,
  service: string,
  sql: string,
  signal?: AbortSignal,
): Promise<QueryResult> {
  return apiSend<QueryResult>(
    "POST",
    `/profiles/${encodeURIComponent(profile)}/services/${encodeURIComponent(service)}/query`,
    { sql },
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
