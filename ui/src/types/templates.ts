// Domain types for SQL templates: named, parameterized SQL scripts scoped to a
// workspace. Mirrors the /api/templates JSON contract of `easy-infra serve`.

/** A template in a list: everything but the SQL body. */
export interface TemplateSummary {
  name: string;
  description: string;
  /** Variable names parsed from the SQL body, in first-seen order. */
  variables: string[];
  /** RFC 3339 timestamp of the last update. */
  updatedAt: string;
}

/** A single template, including its SQL body and parsed variables. */
export interface Template {
  name: string;
  description: string;
  sql: string;
  variables: string[];
  createdAt: string;
  updatedAt: string;
}
