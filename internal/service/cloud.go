package service

import "context"

// CloudBrowser is an optional capability a Service implements when it emulates a
// bundle of AWS services that can be inspected — the backend of the UI's
// LocalStack detail page. Callers type-assert for it and degrade gracefully when
// a service does not provide it, mirroring how KeyBrowser models the Redis
// keyspace and Browser the object store. LocalStack is the only implementer
// today; each method backs one AWS service's detail page.
type CloudBrowser interface {
	// CloudHealth reports the emulator's state: its version/edition and the
	// per-service health map (e.g. {"sqs": "running"}) — the backend of the
	// LocalStack overview's data-driven service cards and Configuration panel.
	CloudHealth(ctx context.Context, spec Spec) (CloudHealth, error)

	// Queues lists the SQS queues on the emulated account, with their message
	// counts — the backend of the SQS detail page.
	Queues(ctx context.Context, spec Spec) ([]QueueInfo, error)

	// Identities lists the SES email/domain identities on the emulated account,
	// with their verification status — the backend of the SES detail page.
	Identities(ctx context.Context, spec Spec) ([]IdentityInfo, error)
}

// CloudHealth is the emulator's reported status, read from LocalStack's
// `/_localstack/health` endpoint and shaped for JSON. Services maps each
// emulated AWS service to its state ("running", "available", "disabled",
// "error", …); the UI drives both the service cards and the Configuration panel
// from it so they cannot disagree.
type CloudHealth struct {
	// Version is the running LocalStack version (e.g. "4.0.3"), when reported.
	Version string `json:"version,omitempty"`
	// Edition is the LocalStack edition (e.g. "community"), when reported.
	Edition string `json:"edition,omitempty"`
	// Services maps each emulated AWS service id to its health state.
	Services map[string]string `json:"services"`
}

// QueueInfo is one SQS queue and its summary metadata, shaped for JSON.
type QueueInfo struct {
	// Name is the queue name (the last path segment of its URL).
	Name string `json:"name"`
	// URL is the queue's fully-qualified URL.
	URL string `json:"url"`
	// Messages is the approximate number of visible messages
	// (ApproximateNumberOfMessages).
	Messages int64 `json:"messages"`
	// InFlight is the approximate number of in-flight messages
	// (ApproximateNumberOfMessagesNotVisible).
	InFlight int64 `json:"inFlight"`
}

// IdentityInfo is one SES identity (an email address or a domain) and whether
// it is verified, shaped for JSON.
type IdentityInfo struct {
	// Identity is the email address or domain name.
	Identity string `json:"identity"`
	// Type is the identity kind, "EMAIL_ADDRESS" or "DOMAIN".
	Type string `json:"type"`
	// Verified is true when the identity's verification has succeeded.
	Verified bool `json:"verified"`
}
