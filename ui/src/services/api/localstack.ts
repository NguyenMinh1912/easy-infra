// LocalStack cloud-browser endpoints: list a profile's SQS queues and SES
// identities. Listing failures (endpoint unreachable) resolve successfully with
// `error` set on the result — only transport/protocol problems reject.

import type { IdentitiesResponse, QueuesResponse } from "@/types/localstack";

import { apiGet } from "./client";

const base = (profile: string, service: string) =>
  `/profiles/${encodeURIComponent(profile)}/services/${encodeURIComponent(service)}`;

/** List the profile's SQS queues with their message counts. */
export async function listQueues(
  profile: string,
  service: string,
  signal?: AbortSignal,
): Promise<QueuesResponse> {
  return apiGet<QueuesResponse>(`${base(profile, service)}/queues`, signal);
}

/** List the profile's SES identities with their verification status. */
export async function listIdentities(
  profile: string,
  service: string,
  signal?: AbortSignal,
): Promise<IdentitiesResponse> {
  return apiGet<IdentitiesResponse>(
    `${base(profile, service)}/identities`,
    signal,
  );
}
