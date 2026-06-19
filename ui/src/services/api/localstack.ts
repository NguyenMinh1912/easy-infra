// LocalStack cloud-browser endpoints: the health snapshot plus a profile's SQS
// queues and SES identities. Listing failures (endpoint unreachable) resolve
// successfully with `error` set on the result — only transport/protocol
// problems reject. An optional `region` re-scopes the query to that AWS region,
// overriding the profile's saved region without mutating it.

import type {
  HealthResponse,
  IdentitiesResponse,
  QueuesResponse,
} from "@/types/localstack";

import { apiGet, apiSend } from "./client";

const base = (profile: string, service: string) =>
  `/profiles/${encodeURIComponent(profile)}/services/${encodeURIComponent(service)}`;

/** Append a `region` query param when one is selected. */
const scoped = (path: string, region?: string) =>
  region ? `${path}?region=${encodeURIComponent(region)}` : path;

/** Read the LocalStack health snapshot: version and per-service state map. */
export async function getLocalstackHealth(
  profile: string,
  service: string,
  region?: string,
  signal?: AbortSignal,
): Promise<HealthResponse> {
  return apiGet<HealthResponse>(
    scoped(`${base(profile, service)}/health`, region),
    signal,
  );
}

/** List the profile's SQS queues with their message counts. */
export async function listQueues(
  profile: string,
  service: string,
  region?: string,
  signal?: AbortSignal,
): Promise<QueuesResponse> {
  return apiGet<QueuesResponse>(
    scoped(`${base(profile, service)}/queues`, region),
    signal,
  );
}

/**
 * Create an SQS queue named `name` in the profile. A name ending in `.fifo`
 * creates a FIFO queue. The queue is created in `region` when one is selected,
 * matching the region the listing reflects. Resolves on success (204) and
 * rejects with an {@link ApiError} otherwise.
 */
export async function createQueue(
  profile: string,
  service: string,
  name: string,
  region?: string,
  signal?: AbortSignal,
): Promise<void> {
  return apiSend<void>(
    "POST",
    scoped(`${base(profile, service)}/queues`, region),
    { name },
    signal,
  );
}

/** Delete the queue at `url` from the profile, in `region` when selected. */
export async function deleteQueue(
  profile: string,
  service: string,
  url: string,
  region?: string,
  signal?: AbortSignal,
): Promise<void> {
  const query = new URLSearchParams({ url });
  if (region) query.set("region", region);
  return apiSend<void>(
    "DELETE",
    `${base(profile, service)}/queues?${query.toString()}`,
    undefined,
    signal,
  );
}

/**
 * Remove all messages from the queue at `url`, leaving the queue in place, in
 * `region` when selected.
 */
export async function purgeQueue(
  profile: string,
  service: string,
  url: string,
  region?: string,
  signal?: AbortSignal,
): Promise<void> {
  const query = new URLSearchParams({ url });
  if (region) query.set("region", region);
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
  region?: string,
  signal?: AbortSignal,
): Promise<IdentitiesResponse> {
  return apiGet<IdentitiesResponse>(
    scoped(`${base(profile, service)}/identities`, region),
    signal,
  );
}

/**
 * Register an SES identity for verification in the profile. An identity
 * containing `@` is verified as an email address, otherwise as a domain. The
 * identity is created in `region` when one is selected, matching the region the
 * listing reflects. Resolves on success (204) and rejects with an
 * {@link ApiError} otherwise.
 */
export async function createIdentity(
  profile: string,
  service: string,
  identity: string,
  region?: string,
  signal?: AbortSignal,
): Promise<void> {
  return apiSend<void>(
    "POST",
    scoped(`${base(profile, service)}/identities`, region),
    { identity },
    signal,
  );
}

/** Delete the SES `identity` from the profile, in `region` when selected. */
export async function deleteIdentity(
  profile: string,
  service: string,
  identity: string,
  region?: string,
  signal?: AbortSignal,
): Promise<void> {
  const query = new URLSearchParams({ identity });
  if (region) query.set("region", region);
  return apiSend<void>(
    "DELETE",
    `${base(profile, service)}/identities?${query.toString()}`,
    undefined,
    signal,
  );
}
