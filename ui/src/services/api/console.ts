// Console endpoints: execute SQL against a profile's service and fetch its
// schema for editor autocomplete.

import type { QueryResult, Relation, SchemaInfo } from "@/types/console";

import { apiGet, apiSend } from "./client";

/** Identifies a single result row by its primary-key column values. */
export type RowKey = Record<string, string>;

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

/**
 * Update one column of a single result row, addressed by its primary key.
 * `value` is the new cell text, or null to set the column to NULL. The server
 * builds a parameterized UPDATE and coerces the text to the column's type;
 * resolves with the command tag (e.g. "UPDATE 1").
 */
export async function updateRow(
  profile: string,
  service: string,
  edit: {
    schema: string;
    table: string;
    key: RowKey;
    column: string;
    value: string | null;
  },
  signal?: AbortSignal,
): Promise<{ command: string }> {
  return apiSend<{ command: string }>(
    "PATCH",
    `/profiles/${encodeURIComponent(profile)}/services/${encodeURIComponent(service)}/row`,
    edit,
    signal,
  );
}

/** Delete a single result row, addressed by its primary key. */
export async function deleteRow(
  profile: string,
  service: string,
  row: { schema: string; table: string; key: RowKey },
  signal?: AbortSignal,
): Promise<{ command: string }> {
  return apiSend<{ command: string }>(
    "DELETE",
    `/profiles/${encodeURIComponent(profile)}/services/${encodeURIComponent(service)}/row`,
    row,
    signal,
  );
}

/** One column-equals-value predicate joining a related table to a row. */
export interface RelationFilter {
  column: string;
  /** The matching value as text, or null to match NULL. */
  value: string | null;
}

/**
 * Fetch the rows reachable through a foreign-key relation: the related table
 * filtered to where its `foreign` columns equal the originating row's values.
 * The result is shaped like any other query (it may itself be editable and
 * carry relations), so callers can render and explore it the same way. Like
 * {@link executeQuery}, statement failures resolve with `error` set.
 */
export async function relatedRows(
  profile: string,
  service: string,
  query: { schema: string; table: string; filters: RelationFilter[] },
  signal?: AbortSignal,
): Promise<QueryResult> {
  return apiSend<QueryResult>(
    "POST",
    `/profiles/${encodeURIComponent(profile)}/services/${encodeURIComponent(service)}/related`,
    query,
    signal,
  );
}

/**
 * Fetch the foreign-key relations of one table, by name, for the relationship
 * canvas. Works independently of any query result (including tables without a
 * primary key). Introspection failures resolve with `error` set.
 */
export async function getTableRelations(
  profile: string,
  service: string,
  schema: string,
  table: string,
  signal?: AbortSignal,
): Promise<{ relations: Relation[]; error?: string }> {
  const query = `?schema=${encodeURIComponent(schema)}&table=${encodeURIComponent(table)}`;
  return apiGet<{ relations: Relation[]; error?: string }>(
    `/profiles/${encodeURIComponent(profile)}/services/${encodeURIComponent(service)}/table-relations${query}`,
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
