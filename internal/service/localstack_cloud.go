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
