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
