// Client-side mirror of the backend's {{variable}} parsing and rendering
// (internal/sqltemplate). Used for the editor's live "detected variables" hint
// and the run dialog's rendered-SQL preview. The server performs the
// authoritative render at run time.

const PLACEHOLDER = /\{\{\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*\}\}/g;

/** Distinct variable names referenced in `sql`, in first-seen order. */
export function parseVariables(sql: string): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const match of sql.matchAll(PLACEHOLDER)) {
    const name = match[1];
    if (!seen.has(name)) {
      seen.add(name);
      out.push(name);
    }
  }
  return out;
}

/**
 * Substitute each {{variable}} in `sql` with its value from `values`, leaving a
 * placeholder untouched when no value is supplied (so the preview shows what is
 * still missing). Textual substitution, matching the backend.
 */
export function renderSql(
  sql: string,
  values: Record<string, string>,
): string {
  return sql.replace(PLACEHOLDER, (match, name: string) =>
    name in values && values[name] !== "" ? values[name] : match,
  );
}
