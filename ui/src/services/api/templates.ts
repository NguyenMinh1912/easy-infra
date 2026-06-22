// SQL template endpoints: CRUD over a workspace's saved, parameterized SQL
// scripts plus a run endpoint that renders the template and executes it against
// a profile's service. Components and hooks depend on these, not on `fetch`.

import type { QueryResult } from "@/types/console";
import type { Template, TemplateSummary } from "@/types/templates";

import { apiGet, apiSend } from "./client";

/** List the workspace's templates (summaries, no SQL body). */
export function listTemplates(
  signal?: AbortSignal,
): Promise<TemplateSummary[]> {
  return apiGet<TemplateSummary[]>("/templates", signal);
}

/** Fetch a single template with its SQL body. */
export function getTemplate(
  name: string,
  signal?: AbortSignal,
): Promise<Template> {
  return apiGet<Template>(`/templates/${encodeURIComponent(name)}`, signal);
}

/** Create a template; returns the saved template. */
export function createTemplate(body: {
  name: string;
  description: string;
  sql: string;
}): Promise<Template> {
  return apiSend<Template>("POST", "/templates", body);
}

/** Replace a template's description and SQL; returns the saved template. */
export function updateTemplate(
  name: string,
  body: { description: string; sql: string },
): Promise<Template> {
  return apiSend<Template>(
    "PUT",
    `/templates/${encodeURIComponent(name)}`,
    body,
  );
}

/** Delete a template. */
export function deleteTemplate(name: string): Promise<void> {
  return apiSend<void>("DELETE", `/templates/${encodeURIComponent(name)}`);
}

/**
 * Render the named template with `variables` and run it against the chosen
 * profile/service. Like the console, statement failures resolve with `error`
 * set on the result; only transport problems reject.
 */
export function runTemplate(
  name: string,
  body: {
    profile: string;
    service: string;
    variables: Record<string, string>;
    db?: number;
  },
  signal?: AbortSignal,
): Promise<QueryResult> {
  return apiSend<QueryResult>(
    "POST",
    `/templates/${encodeURIComponent(name)}/run`,
    body,
    signal,
  );
}
