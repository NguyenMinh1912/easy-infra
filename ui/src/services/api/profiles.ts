// Profiles endpoints. Translate the /api/profiles REST surface into the domain
// `ProfilesResult` type. Components and hooks depend on these functions, not on
// `fetch` directly.

import type {
  ProfileConfig,
  ProfileServiceConfig,
  ProfilesResult,
} from "@/types/profiles";
import { apiGet, apiSend } from "./client";

/** List the project's profiles and the active one. */
export function listProfiles(signal?: AbortSignal): Promise<ProfilesResult> {
  return apiGet<ProfilesResult>("/profiles", signal);
}

/** Fetch a single profile's per-service environment config. */
export function getProfileConfig(
  name: string,
  signal?: AbortSignal,
): Promise<ProfileConfig> {
  return apiGet<ProfileConfig>(
    `/profiles/${encodeURIComponent(name)}`,
    signal,
  );
}

/** Replace a profile's per-service environment config; returns the saved config. */
export function updateProfileConfig(
  name: string,
  services: ProfileServiceConfig[],
): Promise<ProfileConfig> {
  return apiSend<ProfileConfig>(
    "PUT",
    `/profiles/${encodeURIComponent(name)}`,
    { services },
  );
}

/** Create an empty profile (no services); returns the new list. */
export function createProfile(name: string): Promise<ProfilesResult> {
  return apiSend<ProfilesResult>("POST", "/profiles", { name });
}

/** Set `name` as the active profile; returns the updated list. */
export function activateProfile(name: string): Promise<ProfilesResult> {
  return apiSend<ProfilesResult>(
    "POST",
    `/profiles/${encodeURIComponent(name)}/activate`,
  );
}

/** Remove a profile. The backend refuses to remove the active one. */
export function deleteProfile(name: string): Promise<void> {
  return apiSend<void>("DELETE", `/profiles/${encodeURIComponent(name)}`);
}

/** Result of probing a service with a candidate env config. */
export interface ConnectionCheck {
  ok: boolean;
  error?: string;
}

/**
 * Probe a service with the given (possibly unsaved) env config, so the user can
 * verify connectivity before saving. The probe failing is reported as
 * `ok: false` with a reason, not a thrown error.
 */
export function checkServiceConnection(
  profile: string,
  service: string,
  config: Record<string, unknown>,
  signal?: AbortSignal,
): Promise<ConnectionCheck> {
  return apiSend<ConnectionCheck>(
    "POST",
    `/profiles/${encodeURIComponent(profile)}/services/${encodeURIComponent(service)}/check`,
    { config },
    signal,
  );
}
