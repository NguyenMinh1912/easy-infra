// Jenkins detail endpoints: the instance info plus a profile's jobs and a job's
// recent builds. Request failures (server unreachable) resolve successfully
// with `error` set on the result — only transport/protocol problems reject.

import type {
  BuildsResponse,
  JenkinsInfoResponse,
  JobsResponse,
} from "@/types/jenkins";

import { apiGet } from "./client";

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
