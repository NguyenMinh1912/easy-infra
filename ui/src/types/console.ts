// Domain types for the service console (ad-hoc SQL against a profile's
// service), mirroring the /api/profiles/{name}/services/{service}/query and
// /schema JSON contracts.

/**
 * Outcome of one console statement. A failing statement is an expected
 * outcome: the API responds 200 with `error` set and the result fields empty.
 */
export interface QueryResult {
  columns: string[];
  rows: unknown[][];
  /** Rows returned (row-producing statements) or affected (DML). */
  rowCount: number;
  /** The server's command tag, e.g. "SELECT 3" or "UPDATE 1". */
  command: string;
  /** True when the server's row cap was hit and `rows` is a prefix. */
  truncated: boolean;
  /** Server-measured execution time in milliseconds. */
  durationMs: number;
  error?: string;
}

/** One queryable table and its columns, for editor autocomplete. */
export interface TableInfo {
  schema: string;
  name: string;
  columns: string[];
}

/**
 * The service's queryable namespace. Introspection failing (e.g. database
 * unreachable) is reported via `error`; the editor then falls back to
 * keyword-only completion.
 */
export interface SchemaInfo {
  tables: TableInfo[];
  /**
   * The schema the profile's connection resolves unqualified names against
   * (its search_path). The editor defaults completion to it, so suggestions
   * match the schema statements execute in.
   */
  currentSchema: string;
  error?: string;
}
