// LocalStack cloud-browser endpoints: list a profile's SQS queues and SES
// identities. Listing failures (endpoint unreachable) resolve successfully with
// `error` set on the result — only transport/protocol problems reject.

import type { IdentitiesResponse, QueuesResponse } from "@/types/localstack";

import { apiGet, apiSend } from "./client";

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

/**
 * Create an SQS queue named `name` in the profile. A name ending in `.fifo`
 * creates a FIFO queue. Resolves on success (204) and rejects with an
 * {@link ApiError} otherwise.
 */
export async function createQueue(
  profile: string,
  service: string,
  name: string,
  signal?: AbortSignal,
): Promise<void> {
  return apiSend<void>(
    "POST",
    `${base(profile, service)}/queues`,
    { name },
    signal,
  );
}

/** Delete the queue at `url` from the profile. */
export async function deleteQueue(
  profile: string,
  service: string,
  url: string,
  signal?: AbortSignal,
): Promise<void> {
  const query = new URLSearchParams({ url });
  return apiSend<void>(
    "DELETE",
    `${base(profile, service)}/queues?${query.toString()}`,
    undefined,
    signal,
  );
}

/** Remove all messages from the queue at `url`, leaving the queue in place. */
export async function purgeQueue(
  profile: string,
  service: string,
  url: string,
  signal?: AbortSignal,
): Promise<void> {
  const query = new URLSearchParams({ url });
  return apiSend<void>(
    "POST",
    `${base(profile, service)}/queues/purge?${query.toString()}`,
    undefined,
    signal,
  );
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
