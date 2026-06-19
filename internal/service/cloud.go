package service

import "context"

// CloudBrowser is an optional capability a Service implements when it emulates a
// bundle of AWS services that can be inspected — the backend of the UI's
// LocalStack detail page. Callers type-assert for it and degrade gracefully when
// a service does not provide it, mirroring how KeyBrowser models the Redis
// keyspace and Browser the object store. LocalStack is the only implementer
// today; each method backs one AWS service's detail page.
type CloudBrowser interface {
	// Queues lists the SQS queues on the emulated account, with their message
	// counts — the backend of the SQS detail page.
	Queues(ctx context.Context, spec Spec) ([]QueueInfo, error)

	// CreateQueue creates a new SQS queue with the given name on the emulated
	// account. A name ending in ".fifo" creates a FIFO queue.
	CreateQueue(ctx context.Context, spec Spec, name string) error

	// DeleteQueue deletes the SQS queue identified by its URL.
	DeleteQueue(ctx context.Context, spec Spec, url string) error

	// PurgeQueue removes all messages from the SQS queue identified by its URL,
	// leaving the queue itself in place.
	PurgeQueue(ctx context.Context, spec Spec, url string) error

	// Identities lists the SES email/domain identities on the emulated account,
	// with their verification status — the backend of the SES detail page.
	Identities(ctx context.Context, spec Spec) ([]IdentityInfo, error)
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
