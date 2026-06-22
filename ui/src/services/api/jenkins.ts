// Jenkins detail endpoints: the instance info plus a profile's jobs and a job's
// recent builds. Request failures (server unreachable) resolve successfully
// with `error` set on the result — only transport/protocol problems reject.

import type {
  BuildLogResponse,
  BuildsResponse,
  JenkinsInfoResponse,
  JobsResponse,
} from "@/types/jenkins";

import { apiGet, apiSend } from "./client";

const base = (profile: string, service: string) =>
  `/profiles/${encodeURIComponent(profile)}/services/${encodeURIComponent(service)}`;

/** Read the Jenkins instance summary: version, node and job count. */
export async function getJenkinsInfo(
  profile: string,
  service: string,
  signal?: AbortSignal,
): Promise<JenkinsInfoResponse> {
  return apiGet<JenkinsInfoResponse>(`${base(profile, service)}/info`, signal);
}

/** List the profile's Jenkins jobs with their last-build status. */
export async function listJobs(
  profile: string,
  service: string,
  signal?: AbortSignal,
): Promise<JobsResponse> {
  return apiGet<JobsResponse>(`${base(profile, service)}/jobs`, signal);
}

/** List the recent builds of `job`, most recent first. */
export async function listBuilds(
  profile: string,
  service: string,
  job: string,
  signal?: AbortSignal,
): Promise<BuildsResponse> {
  const query = new URLSearchParams({ job });
  return apiGet<BuildsResponse>(`${base(profile, service)}/builds?${query}`, signal);
}

/** Read the console output of build `number` of `job`. */
export async function getBuildLog(
  profile: string,
  service: string,
  job: string,
  number: number,
  signal?: AbortSignal,
): Promise<BuildLogResponse> {
  const query = new URLSearchParams({ job, number: String(number) });
  return apiGet<BuildLogResponse>(`${base(profile, service)}/log?${query}`, signal);
}

/**
 * Trigger a new (parameterless) build of `job`. The build is enqueued
 * asynchronously, so this resolves once Jenkins accepts the request (204), not
 * when the build finishes; it rejects with an {@link ApiError} otherwise.
 */
export async function triggerBuild(
  profile: string,
  service: string,
  job: string,
  signal?: AbortSignal,
): Promise<void> {
  return apiSend<void>("POST", `${base(profile, service)}/build`, { job }, signal);
}
