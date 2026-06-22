// SQL template endpoints: CRUD over a workspace's saved, parameterized SQL
// scripts. Templates are run from the SQL console via its "/" mention menu,
// which inserts a template's body and runs it through the console's own query
// path — so there is no dedicated run client here. Components and hooks depend
// on these functions, not on `fetch`.

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
