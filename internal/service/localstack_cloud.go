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
