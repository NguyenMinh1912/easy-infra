package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	sestypes "github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

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
// the emulated account with their verification status.
func (l LocalStack) Identities(ctx context.Context, spec Spec) ([]IdentityInfo, error) {
	client, err := l.sesOpener()(spec.Env)
	if err != nil {
		return nil, fmt.Errorf("connecting to ses: %w", err)
	}

	out, err := client.ListEmailIdentities(ctx, &sesv2.ListEmailIdentitiesInput{})
	if err != nil {
		return nil, fmt.Errorf("listing identities: %w", err)
	}

	identities := make([]IdentityInfo, 0, len(out.EmailIdentities))
	for _, id := range out.EmailIdentities {
		identities = append(identities, IdentityInfo{
			Identity: aws.ToString(id.IdentityName),
			Type:     string(id.IdentityType),
			Verified: id.VerificationStatus == sestypes.VerificationStatusSuccess,
		})
	}
	return identities, nil
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
