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

	// CreateQueue creates a new SQS queue with the given name on the emulated
	// account. A name ending in ".fifo" creates a FIFO queue.
	CreateQueue(ctx context.Context, spec Spec, name string) error

	// DeleteQueue deletes the SQS queue identified by its URL.
	DeleteQueue(ctx context.Context, spec Spec, url string) error

	// PurgeQueue removes all messages from the SQS queue identified by its URL,
	// leaving the queue itself in place.
	PurgeQueue(ctx context.Context, spec Spec, url string) error

	// Messages peeks at up to limit messages on the SQS queue identified by its
	// URL — the backend of the queue's message list view. It is a non-destructive
	// preview: messages are received with a zero visibility timeout so they stay
	// available to real consumers.
	Messages(ctx context.Context, spec Spec, url string, limit int) ([]MessageInfo, error)

	// Identities lists the SES email/domain identities on the emulated account,
	// with their verification status — the backend of the SES detail page.
	Identities(ctx context.Context, spec Spec) ([]IdentityInfo, error)

	// IdentityMessages lists the SES messages sent through the emulated account
	// that involve the given identity (as sender or recipient) — the backend of
	// an identity's mail list page.
	IdentityMessages(ctx context.Context, spec Spec, identity string) ([]MailInfo, error)

	// CreateIdentity registers a new SES identity for verification. An identity
	// containing "@" is verified as an email address, otherwise as a domain.
	CreateIdentity(ctx context.Context, spec Spec, identity string) error

	// DeleteIdentity removes the SES identity (email address or domain) from the
	// emulated account.
	DeleteIdentity(ctx context.Context, spec Spec, identity string) error
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

// MessageInfo is one SQS message previewed from a queue, shaped for JSON. It
// carries the body and the metadata most useful for inspecting a queue, not the
// receipt handle — the preview never deletes or mutates messages.
type MessageInfo struct {
	// ID is the message's MessageId.
	ID string `json:"id"`
	// Body is the message payload.
	Body string `json:"body"`
	// SentAt is the SentTimestamp in Unix milliseconds, 0 when not reported.
	SentAt int64 `json:"sentAt"`
	// ReceiveCount is the ApproximateReceiveCount — how many times the message
	// has been received, a hint at redelivery/processing trouble.
	ReceiveCount int64 `json:"receiveCount"`
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

// MailInfo is one SES message recorded by the emulator, shaped for JSON. It
// covers the SendEmail shape (subject + body); raw messages contribute their
// recipients and source but leave subject/body empty.
type MailInfo struct {
	// ID is the emulator's message id.
	ID string `json:"id"`
	// Source is the sender (the "From" address).
	Source string `json:"source"`
	// Destination lists every recipient (To, Cc and Bcc combined).
	Destination []string `json:"destination"`
	// Subject is the message subject, when the message carries one.
	Subject string `json:"subject"`
	// Body is the message's text body, falling back to the HTML body.
	Body string `json:"body"`
	// Timestamp is when the emulator recorded the message (RFC 3339), as the
	// emulator reports it.
	Timestamp string `json:"timestamp"`
}
