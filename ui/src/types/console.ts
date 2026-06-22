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
  /**
   * Present when the result maps back to a single editable table, enabling
   * inline cell edits and row deletes. Absent for joins, expression results,
   * or tables without a usable primary key.
   */
  editable?: EditableInfo;
  error?: string;
}

/**
 * Describes how a single-table result maps back to its source table so the
 * console can edit rows in place.
 */
export interface EditableInfo {
  schema: string;
  table: string;
  /** Primary-key column names; all are present among the result columns. */
  primaryKey: string[];
  /**
   * Parallel to {@link QueryResult.columns}: the source table column each
   * result column maps to, or "" when the column isn't directly updatable
   * (an expression, or a column we couldn't resolve).
   */
  columns: string[];
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
