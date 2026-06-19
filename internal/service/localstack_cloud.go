package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	sestypes "github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// CloudHealth implements CloudBrowser: read LocalStack's `/_localstack/health`
// endpoint and return the reported version/edition and per-service state map.
// Unlike the SDK-backed listings this is a plain HTTP GET — the health endpoint
// is unsigned — so it doubles as a cheap reachability probe for the overview.
func (l LocalStack) CloudHealth(ctx context.Context, spec Spec) (CloudHealth, error) {
	p, err := localstackParamsFrom(spec.Env)
	if err != nil {
		return CloudHealth{}, err
	}
	body, err := l.healthGetter()(ctx, p.endpoint())
	if err != nil {
		return CloudHealth{}, fmt.Errorf("reaching localstack: %w", err)
	}
	// The health payload nests the per-service map under "services" and reports
	// the version/edition alongside it; decode only the fields we surface.
	var raw struct {
		Services map[string]string `json:"services"`
		Version  string            `json:"version"`
		Edition  string            `json:"edition"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return CloudHealth{}, fmt.Errorf("parsing localstack health: %w", err)
	}
	if raw.Services == nil {
		raw.Services = map[string]string{}
	}
	return CloudHealth{Version: raw.Version, Edition: raw.Edition, Services: raw.Services}, nil
}

// Queues implements CloudBrowser: list the SQS queues on the emulated account,
// annotating each with its approximate message counts. A per-queue attribute
// read that fails is non-fatal — the queue is still listed, with zero counts —
// so one bad queue does not blank the whole page.
func (l LocalStack) Queues(ctx context.Context, spec Spec) ([]QueueInfo, error) {
	client, err := l.sqsOpener()(spec.Env)
	if err != nil {
		return nil, fmt.Errorf("connecting to sqs: %w", err)
	}

	out, err := client.ListQueues(ctx, &sqs.ListQueuesInput{})
	if err != nil {
		return nil, fmt.Errorf("listing queues: %w", err)
	}

	queues := make([]QueueInfo, 0, len(out.QueueUrls))
	for _, url := range out.QueueUrls {
		info := QueueInfo{Name: queueNameFromURL(url), URL: url}
		attrs, err := client.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
			QueueUrl: aws.String(url),
			AttributeNames: []sqstypes.QueueAttributeName{
				sqstypes.QueueAttributeNameApproximateNumberOfMessages,
				sqstypes.QueueAttributeNameApproximateNumberOfMessagesNotVisible,
			},
		})
		if err == nil {
			info.Messages = parseAttrInt(attrs.Attributes, string(sqstypes.QueueAttributeNameApproximateNumberOfMessages))
			info.InFlight = parseAttrInt(attrs.Attributes, string(sqstypes.QueueAttributeNameApproximateNumberOfMessagesNotVisible))
		}
		queues = append(queues, info)
	}
	return queues, nil
}

// CreateQueue implements CloudBrowser: create a new SQS queue. A name ending in
// ".fifo" is created as a FIFO queue, which SQS requires the FifoQueue attribute
// to be set for.
func (l LocalStack) CreateQueue(ctx context.Context, spec Spec, name string) error {
	client, err := l.sqsOpener()(spec.Env)
	if err != nil {
		return fmt.Errorf("connecting to sqs: %w", err)
	}

	in := &sqs.CreateQueueInput{QueueName: aws.String(name)}
	if strings.HasSuffix(name, ".fifo") {
		in.Attributes = map[string]string{
			string(sqstypes.QueueAttributeNameFifoQueue): "true",
		}
	}
	if _, err := client.CreateQueue(ctx, in); err != nil {
		return fmt.Errorf("creating queue: %w", err)
	}
	return nil
}

// DeleteQueue implements CloudBrowser: delete the SQS queue at the given URL.
func (l LocalStack) DeleteQueue(ctx context.Context, spec Spec, url string) error {
	client, err := l.sqsOpener()(spec.Env)
	if err != nil {
		return fmt.Errorf("connecting to sqs: %w", err)
	}
	if _, err := client.DeleteQueue(ctx, &sqs.DeleteQueueInput{QueueUrl: aws.String(url)}); err != nil {
		return fmt.Errorf("deleting queue: %w", err)
	}
	return nil
}

// PurgeQueue implements CloudBrowser: remove all messages from the SQS queue at
// the given URL.
func (l LocalStack) PurgeQueue(ctx context.Context, spec Spec, url string) error {
	client, err := l.sqsOpener()(spec.Env)
	if err != nil {
		return fmt.Errorf("connecting to sqs: %w", err)
	}
	if _, err := client.PurgeQueue(ctx, &sqs.PurgeQueueInput{QueueUrl: aws.String(url)}); err != nil {
		return fmt.Errorf("purging queue: %w", err)
	}
	return nil
}

// maxQueueMessages bounds a single message preview. SQS caps ReceiveMessage at
// 10 messages per call, so a non-paginating preview can show at most that many.
const maxQueueMessages = 10

// Messages implements CloudBrowser: peek at up to limit messages on the queue at
// the given URL. It is non-destructive — messages are received with a zero
// visibility timeout so they reappear immediately for real consumers — and so it
// shows whatever SQS happens to return, up to the per-call cap of 10.
func (l LocalStack) Messages(ctx context.Context, spec Spec, url string, limit int) ([]MessageInfo, error) {
	client, err := l.sqsOpener()(spec.Env)
	if err != nil {
		return nil, fmt.Errorf("connecting to sqs: %w", err)
	}
	if limit <= 0 || limit > maxQueueMessages {
		limit = maxQueueMessages
	}

	out, err := client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(url),
		MaxNumberOfMessages: int32(limit),
		VisibilityTimeout:   0,
		MessageSystemAttributeNames: []sqstypes.MessageSystemAttributeName{
			sqstypes.MessageSystemAttributeNameSentTimestamp,
			sqstypes.MessageSystemAttributeNameApproximateReceiveCount,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("receiving messages: %w", err)
	}

	messages := make([]MessageInfo, 0, len(out.Messages))
	for _, m := range out.Messages {
		messages = append(messages, MessageInfo{
			ID:           aws.ToString(m.MessageId),
			Body:         aws.ToString(m.Body),
			SentAt:       parseAttrInt(m.Attributes, string(sqstypes.MessageSystemAttributeNameSentTimestamp)),
			ReceiveCount: parseAttrInt(m.Attributes, string(sqstypes.MessageSystemAttributeNameApproximateReceiveCount)),
		})
	}
	return messages, nil
}

// Identities implements CloudBrowser: list the SES email/domain identities on
// the emulated account with their verification status. It uses the SES v1 API
// (ListIdentities + GetIdentityVerificationAttributes) because LocalStack's
// community edition does not implement sesv2. v1 ListIdentities returns plain
// identity names without a type, so the type is inferred from the name: a name
// containing "@" is an email address, otherwise a domain.
func (l LocalStack) Identities(ctx context.Context, spec Spec) ([]IdentityInfo, error) {
	client, err := l.sesOpener()(spec.Env)
	if err != nil {
		return nil, fmt.Errorf("connecting to ses: %w", err)
	}

	out, err := client.ListIdentities(ctx, &ses.ListIdentitiesInput{})
	if err != nil {
		return nil, fmt.Errorf("listing identities: %w", err)
	}

	// Verification status is a separate call in v1; a failure here is
	// non-fatal — the identities are still listed, just as unverified.
	var attrs map[string]sestypes.IdentityVerificationAttributes
	if len(out.Identities) > 0 {
		va, err := client.GetIdentityVerificationAttributes(ctx, &ses.GetIdentityVerificationAttributesInput{
			Identities: out.Identities,
		})
		if err == nil {
			attrs = va.VerificationAttributes
		}
	}

	identities := make([]IdentityInfo, 0, len(out.Identities))
	for _, name := range out.Identities {
		identities = append(identities, IdentityInfo{
			Identity: name,
			Type:     identityType(name),
			Verified: attrs[name].VerificationStatus == sestypes.VerificationStatusSuccess,
		})
	}
	return identities, nil
}

// Messages implements CloudBrowser: list the SES messages the emulator recorded
// that involve the given identity, newest first. LocalStack keeps every message
// sent through SES in an in-memory store exposed at the `/_aws/ses` developer
// endpoint (there is no SDK call for it), so this is a plain HTTP GET like the
// health probe. A message matches the identity when the identity is its sender
// or one of its recipients; a domain identity also matches any address at that
// domain.
func (l LocalStack) Messages(ctx context.Context, spec Spec, identity string) ([]MessageInfo, error) {
	p, err := localstackParamsFrom(spec.Env)
	if err != nil {
		return nil, err
	}
	body, err := l.messagesGetter()(ctx, p.endpoint())
	if err != nil {
		return nil, fmt.Errorf("reaching localstack: %w", err)
	}

	// The store nests messages under "messages"; each carries its sender,
	// recipients split across To/Cc/Bcc, and either a subject+body (SendEmail)
	// or only raw data (SendRawEmail). Decode the fields we surface and tolerate
	// the rest.
	var raw struct {
		Messages []struct {
			ID          string `json:"Id"`
			Source      string `json:"Source"`
			Subject     string `json:"Subject"`
			Timestamp   string `json:"Timestamp"`
			Destination struct {
				ToAddresses  []string `json:"ToAddresses"`
				CcAddresses  []string `json:"CcAddresses"`
				BccAddresses []string `json:"BccAddresses"`
			} `json:"Destination"`
			Body struct {
				TextPart string `json:"text_part"`
				HTMLPart string `json:"html_part"`
			} `json:"Body"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parsing localstack ses messages: %w", err)
	}

	messages := make([]MessageInfo, 0, len(raw.Messages))
	for _, m := range raw.Messages {
		dest := make([]string, 0, len(m.Destination.ToAddresses)+len(m.Destination.CcAddresses)+len(m.Destination.BccAddresses))
		dest = append(dest, m.Destination.ToAddresses...)
		dest = append(dest, m.Destination.CcAddresses...)
		dest = append(dest, m.Destination.BccAddresses...)

		if !messageInvolves(identity, m.Source, dest) {
			continue
		}

		body := m.Body.TextPart
		if body == "" {
			body = m.Body.HTMLPart
		}
		messages = append(messages, MessageInfo{
			ID:          m.ID,
			Source:      m.Source,
			Destination: dest,
			Subject:     m.Subject,
			Body:        body,
			Timestamp:   m.Timestamp,
		})
	}
	// Newest first, so the most recent mail is at the top of the list. The store
	// returns messages oldest-first.
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, nil
}

// messageInvolves reports whether the identity is the message's sender or one of
// its recipients. A domain identity (no "@") also matches any address at that
// domain, so a domain's mail list shows every message to or from it.
func messageInvolves(identity, source string, dest []string) bool {
	addresses := append([]string{source}, dest...)
	domain := identityType(identity) == "DOMAIN"
	suffix := "@" + identity
	for _, addr := range addresses {
		if strings.EqualFold(addr, identity) {
			return true
		}
		if domain && strings.HasSuffix(strings.ToLower(addr), strings.ToLower(suffix)) {
			return true
		}
	}
	return false
}

// CreateIdentity implements CloudBrowser: register an SES identity for
// verification. An identity containing "@" is verified as an email address,
// anything else as a domain — matching how identityType classifies them. Both
// calls are idempotent on LocalStack, which marks new identities as verified
// immediately rather than sending a real confirmation.
func (l LocalStack) CreateIdentity(ctx context.Context, spec Spec, identity string) error {
	client, err := l.sesOpener()(spec.Env)
	if err != nil {
		return fmt.Errorf("connecting to ses: %w", err)
	}

	if identityType(identity) == "EMAIL_ADDRESS" {
		_, err = client.VerifyEmailIdentity(ctx, &ses.VerifyEmailIdentityInput{
			EmailAddress: aws.String(identity),
		})
	} else {
		_, err = client.VerifyDomainIdentity(ctx, &ses.VerifyDomainIdentityInput{
			Domain: aws.String(identity),
		})
	}
	if err != nil {
		return fmt.Errorf("verifying identity: %w", err)
	}
	return nil
}

// DeleteIdentity implements CloudBrowser: remove the SES identity (email
// address or domain) from the emulated account.
func (l LocalStack) DeleteIdentity(ctx context.Context, spec Spec, identity string) error {
	client, err := l.sesOpener()(spec.Env)
	if err != nil {
		return fmt.Errorf("connecting to ses: %w", err)
	}
	if _, err := client.DeleteIdentity(ctx, &ses.DeleteIdentityInput{Identity: aws.String(identity)}); err != nil {
		return fmt.Errorf("deleting identity: %w", err)
	}
	return nil
}

// identityType classifies an SES identity name the way the SESv2 API would
// report it — an email address contains "@", anything else is a domain — so
// the UI's type labels keep working across the v1/v2 switch.
func identityType(name string) string {
	if strings.Contains(name, "@") {
		return "EMAIL_ADDRESS"
	}
	return "DOMAIN"
}

// queueNameFromURL extracts the queue name (the last path segment) from a queue
// URL such as http://localhost:4566/000000000000/my-queue.
func queueNameFromURL(url string) string {
	if i := strings.LastIndex(url, "/"); i >= 0 && i < len(url)-1 {
		return url[i+1:]
	}
	return url
}

// parseAttrInt reads a numeric SQS attribute, treating a missing or malformed
// value as zero — SQS reports counts as decimal strings.
func parseAttrInt(attrs map[string]string, key string) int64 {
	n, err := strconv.ParseInt(attrs[key], 10, 64)
	if err != nil {
		return 0
	}
	return n
}
