// Two-way translation between a PostgreSQL connection string and the discrete
// connection fields the profile config form edits. The headline UX is "paste a
// connection string and the fields fill in"; {@link buildPostgresUrl} provides
// the reverse so the two representations stay in sync.
//
// The mapping mirrors the backend (internal/service/postgres_db.go): JDBC-style
// URLs are accepted by stripping the leading `jdbc:` prefix, and JDBC's
// `currentSchema` query parameter is treated as an alias for PostgreSQL's
// `search_path` so either spelling fills the schema field.

/** The discrete PostgreSQL connection fields the form edits. */
export interface PostgresFields {
  host: string;
  port: string;
  user: string;
  password: string;
  database: string;
  schema: string;
}

/** {@link PostgresFields} plus any query params that don't map to a field. */
export interface ParsedPostgresUrl extends PostgresFields {
  /**
   * Query parameter names present in the URL that have no discrete field (e.g.
   * `sslmode`). Surfaced to the user so it's clear they aren't captured.
   */
  extraParams: string[];
}

const SCHEMES = new Set(["postgres:", "postgresql:"]);

/**
 * Parse a PostgreSQL connection string into discrete fields, or return null if
 * it isn't a recognisable postgres URL (so callers can keep the raw text while
 * the user is still typing).
 */
export function parsePostgresUrl(raw: string): ParsedPostgresUrl | null {
  const trimmed = raw.trim();
  if (!trimmed) return null;

  // JDBC URLs (jdbc:postgresql://…) are accepted, matching the backend.
  const normalized = trimmed.replace(/^jdbc:/i, "");
  let url: URL;
  try {
    url = new URL(normalized);
  } catch {
    return null;
  }
  if (!SCHEMES.has(url.protocol.toLowerCase())) return null;

  const params = url.searchParams;
  const schema =
    params.get("search_path") ?? params.get("currentSchema") ?? "";
  const extraParams: string[] = [];
  for (const key of params.keys()) {
    if (key !== "search_path" && key !== "currentSchema") {
      extraParams.push(key);
    }
  }

  return {
    host: url.hostname,
    port: url.port,
    user: safeDecode(url.username),
    password: safeDecode(url.password),
    database: safeDecode(url.pathname.replace(/^\//, "")),
    schema,
    extraParams,
  };
}

/**
 * Build a connection string from discrete fields. Returns "" when there's no
 * host to anchor a URL, so an empty form shows an empty connection string
 * rather than a malformed one.
 */
export function buildPostgresUrl(fields: PostgresFields): string {
  if (!fields.host.trim()) return "";

  let auth = "";
  if (fields.user) {
    auth = encodeURIComponent(fields.user);
    if (fields.password) auth += `:${encodeURIComponent(fields.password)}`;
    auth += "@";
  }
  const port = fields.port ? `:${fields.port}` : "";
  const path = `/${fields.database}`;
  const query = fields.schema
    ? `?search_path=${encodeURIComponent(fields.schema)}`
    : "";

  return `postgresql://${auth}${fields.host}${port}${path}${query}`;
}

// safeDecode percent-decodes a URL component, falling back to the raw value if
// it isn't valid encoding (so a stray "%" doesn't throw).
function safeDecode(value: string): string {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}
