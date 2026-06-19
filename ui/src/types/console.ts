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
  /**
   * Foreign-key paths from the source table to other tables, both directions,
   * letting a row link out to the data it refers to and the rows that refer
   * back. Absent when the table has no foreign keys.
   */
  relations?: Relation[];
}

/**
 * One foreign-key path between the result's source table and a related table.
 * `direction` is "references" when the source table points at the related one
 * (follow to the parent row), or "referencedBy" when the related table points
 * back at the source (follow to the child rows).
 */
export interface Relation {
  constraint: string;
  direction: "references" | "referencedBy";
  schema: string;
  table: string;
  columns: RelationColumn[];
}

/**
 * Pairs a column on the source table (`local`) with the related table column it
 * joins to (`foreign`). To follow the relation, the related table is filtered
 * where each `foreign` column equals the source row's `local` value.
 */
export interface RelationColumn {
  local: string;
  foreign: string;
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
