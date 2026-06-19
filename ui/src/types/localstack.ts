// Domain types for the LocalStack cloud browser, mirroring the
// /api/profiles/{name}/services/{service}/queues and /identities JSON
// contracts. A failing listing (endpoint unreachable) is an expected outcome:
// the API responds 200 with `error` set and the payload empty.

/** One SQS queue with its message counts. */
export interface QueueInfo {
  name: string;
  url: string;
  /** Approximate number of visible messages. */
  messages: number;
  /** Approximate number of in-flight messages. */
  inFlight: number;
}

/** Response of GET …/queues — the profile's SQS queues. */
export interface QueuesResponse {
  queues: QueueInfo[];
  error?: string;
}

/** One SES identity (an email address or a domain) and its status. */
export interface IdentityInfo {
  identity: string;
  /** Identity kind: "EMAIL_ADDRESS" or "DOMAIN". */
  type: string;
  verified: boolean;
}

/** Response of GET …/identities — the profile's SES identities. */
export interface IdentitiesResponse {
  identities: IdentityInfo[];
  error?: string;
}

/**
 * Reported state of one emulated AWS service, as LocalStack's
 * `/_localstack/health` returns it. `running` services are active, `available`
 * ones are idle (lazily started), `disabled` are not emulated, and `error`
 * failed to start. Any other string is tolerated and treated as available.
 */
export type ServiceState = "running" | "available" | "disabled" | "error";

/**
 * Response of GET …/health — the LocalStack health snapshot driving the
 * overview's service cards and Configuration panel. An unreachable endpoint
 * comes back with `error` set and `services` empty, mirroring the listings.
 */
export interface HealthResponse {
  /** Running LocalStack version (e.g. "4.0.3"), when reported. */
  version?: string;
  /** LocalStack edition (e.g. "community"), when reported. */
  edition?: string;
  /** Per-service state map, keyed by AWS service id (e.g. `{ sqs: "running" }`). */
  services: Record<string, string>;
  error?: string;
}
