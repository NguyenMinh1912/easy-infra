// Profiles endpoints. Translate the /api/profiles REST surface into the domain
// `ProfilesResult` type. Components and hooks depend on these functions, not on
// `fetch` directly.

import type { ProfilesResult } from "@/types/profiles";
import { apiDelete, apiGet, apiPost } from "./client";

/** List the project's profiles and the active one. */
export function listProfiles(signal?: AbortSignal): Promise<ProfilesResult> {
  return apiGet<ProfilesResult>("/profiles", signal);
}

/** Create a profile scaffolded with default service config; returns the new list. */
export function createProfile(name: string): Promise<ProfilesResult> {
  return apiPost<ProfilesResult>("/profiles", { name });
}

/** Set `name` as the active profile; returns the updated list. */
export function activateProfile(name: string): Promise<ProfilesResult> {
  return apiPost<ProfilesResult>(`/profiles/${encodeURIComponent(name)}/activate`);
}

/** Remove a profile. The backend refuses to remove the active one. */
export function deleteProfile(name: string): Promise<void> {
  return apiDelete(`/profiles/${encodeURIComponent(name)}`);
}
