// Status endpoint. Translates `GET /api/status` into the domain `Status` type.
// Components and hooks depend on this function, not on `fetch` directly.

import type { Status } from "@/types/status";
import { apiGet } from "./client";

/** Fetch the current project status from the backend. */
export function getStatus(signal?: AbortSignal): Promise<Status> {
  return apiGet<Status>("/status", signal);
}
