package service

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	sestypes "github.com/aws/aws-sdk-go-v2/service/ses/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// fakeSQS is an in-memory sqsAPI: it returns a fixed queue list and a per-URL
// attribute map, so a test declares only the replies it cares about. It also
// records the inputs of the mutating calls so tests can assert on them.
type fakeSQS struct {
	urls    []string
	attrs   map[string]map[string]string
	listErr error

	messages   []sqstypes.Message
	receiveErr error
	received   *sqs.ReceiveMessageInput

	created *sqs.CreateQueueInput
	deleted *sqs.DeleteQueueInput
	purged  *sqs.PurgeQueueInput
	mutErr  error
}

func (f fakeSQS) ListQueues(context.Context, *sqs.ListQueuesInput, ...func(*sqs.Options)) (*sqs.ListQueuesOutput, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return &sqs.ListQueuesOutput{QueueUrls: f.urls}, nil
}

func (f fakeSQS) GetQueueAttributes(_ context.Context, in *sqs.GetQueueAttributesInput, _ ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error) {
	return &sqs.GetQueueAttributesOutput{Attributes: f.attrs[aws.ToString(in.QueueUrl)]}, nil
}

func (f *fakeSQS) CreateQueue(_ context.Context, in *sqs.CreateQueueInput, _ ...func(*sqs.Options)) (*sqs.CreateQueueOutput, error) {
	if f.mutErr != nil {
		return nil, f.mutErr
	}
	f.created = in
	return &sqs.CreateQueueOutput{QueueUrl: aws.String("http://localhost:4566/000000000000/" + aws.ToString(in.QueueName))}, nil
}

func (f *fakeSQS) DeleteQueue(_ context.Context, in *sqs.DeleteQueueInput, _ ...func(*sqs.Options)) (*sqs.DeleteQueueOutput, error) {
	if f.mutErr != nil {
		return nil, f.mutErr
	}
	f.deleted = in
	return &sqs.DeleteQueueOutput{}, nil
}

func (f *fakeSQS) PurgeQueue(_ context.Context, in *sqs.PurgeQueueInput, _ ...func(*sqs.Options)) (*sqs.PurgeQueueOutput, error) {
	if f.mutErr != nil {
		return nil, f.mutErr
	}
	f.purged = in
	return &sqs.PurgeQueueOutput{}, nil
}

func (f *fakeSQS) ReceiveMessage(_ context.Context, in *sqs.ReceiveMessageInput, _ ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	if f.receiveErr != nil {
		return nil, f.receiveErr
	}
	f.received = in
	return &sqs.ReceiveMessageOutput{Messages: f.messages}, nil
}

// fakeSES is an in-memory sesAPI returning a fixed identity list and a per-name
// verification status map, mirroring the SES v1 two-call shape.
type fakeSES struct {
	ids     []string
	verify  map[string]sestypes.VerificationStatus
	listErr error
}

func (f fakeSES) ListIdentities(context.Context, *ses.ListIdentitiesInput, ...func(*ses.Options)) (*ses.ListIdentitiesOutput, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return &ses.ListIdentitiesOutput{Identities: f.ids}, nil
}

func (f fakeSES) GetIdentityVerificationAttributes(_ context.Context, in *ses.GetIdentityVerificationAttributesInput, _ ...func(*ses.Options)) (*ses.GetIdentityVerificationAttributesOutput, error) {
	attrs := make(map[string]sestypes.IdentityVerificationAttributes, len(in.Identities))
	for _, id := range in.Identities {
		status, ok := f.verify[id]
		if !ok {
			status = sestypes.VerificationStatusPending
		}
		attrs[id] = sestypes.IdentityVerificationAttributes{VerificationStatus: status}
	}
	return &ses.GetIdentityVerificationAttributesOutput{VerificationAttributes: attrs}, nil
}

func TestLocalStackQueues(t *testing.T) {
	const prefix = "http://localhost:4566/000000000000/"
	fake := fakeSQS{
		urls: []string{prefix + "orders", prefix + "emails"},
		attrs: map[string]map[string]string{
			prefix + "orders": {
				"ApproximateNumberOfMessages":           "5",
				"ApproximateNumberOfMessagesNotVisible": "2",
			},
			// "emails" has no attributes: counts must default to zero, not error.
		},
	}
	ls := LocalStack{openSQS: func(Config) (sqsAPI, error) { return &fake, nil }}

	queues, err := ls.Queues(context.Background(), Spec{Env: Config{"host": "localhost"}})
	if err != nil {
		t.Fatalf("Queues: %v", err)
	}
	if len(queues) != 2 {
		t.Fatalf("got %d queues, want 2", len(queues))
	}
	if queues[0].Name != "orders" || queues[0].URL != prefix+"orders" {
		t.Errorf("queue[0] = %+v, want name=orders url=%sorders", queues[0], prefix)
	}
	if queues[0].Messages != 5 || queues[0].InFlight != 2 {
		t.Errorf("queue[0] counts = (%d,%d), want (5,2)", queues[0].Messages, queues[0].InFlight)
	}
	if queues[1].Name != "emails" || queues[1].Messages != 0 || queues[1].InFlight != 0 {
		t.Errorf("queue[1] = %+v, want name=emails with zero counts", queues[1])
	}
}

func TestLocalStackQueuesListError(t *testing.T) {
	ls := LocalStack{openSQS: func(Config) (sqsAPI, error) {
		return &fakeSQS{listErr: errors.New("connection refused")}, nil
	}}

	if _, err := ls.Queues(context.Background(), Spec{Env: Config{"host": "localhost"}}); err == nil {
		t.Fatal("expected an error when ListQueues fails")
	}
}

func TestLocalStackCreateQueue(t *testing.T) {
	fake := &fakeSQS{}
	ls := LocalStack{openSQS: func(Config) (sqsAPI, error) { return fake, nil }}

	if err := ls.CreateQueue(context.Background(), Spec{Env: Config{"host": "localhost"}}, "orders"); err != nil {
		t.Fatalf("CreateQueue: %v", err)
	}
	if fake.created == nil || aws.ToString(fake.created.QueueName) != "orders" {
		t.Fatalf("CreateQueue input = %+v, want QueueName=orders", fake.created)
	}
	if len(fake.created.Attributes) != 0 {
		t.Errorf("non-FIFO queue got attributes %v, want none", fake.created.Attributes)
	}
}

func TestLocalStackCreateFifoQueue(t *testing.T) {
	fake := &fakeSQS{}
	ls := LocalStack{openSQS: func(Config) (sqsAPI, error) { return fake, nil }}

	if err := ls.CreateQueue(context.Background(), Spec{Env: Config{"host": "localhost"}}, "orders.fifo"); err != nil {
		t.Fatalf("CreateQueue: %v", err)
	}
	if got := fake.created.Attributes[string(sqstypes.QueueAttributeNameFifoQueue)]; got != "true" {
		t.Errorf("FifoQueue attribute = %q, want \"true\"", got)
	}
}

func TestLocalStackCreateQueueError(t *testing.T) {
	ls := LocalStack{openSQS: func(Config) (sqsAPI, error) {
		return &fakeSQS{mutErr: errors.New("boom")}, nil
	}}
	if err := ls.CreateQueue(context.Background(), Spec{Env: Config{"host": "localhost"}}, "orders"); err == nil {
		t.Fatal("expected an error when CreateQueue fails")
	}
}

func TestLocalStackDeleteQueue(t *testing.T) {
	const url = "http://localhost:4566/000000000000/orders"
	fake := &fakeSQS{}
	ls := LocalStack{openSQS: func(Config) (sqsAPI, error) { return fake, nil }}

	if err := ls.DeleteQueue(context.Background(), Spec{Env: Config{"host": "localhost"}}, url); err != nil {
		t.Fatalf("DeleteQueue: %v", err)
	}
	if fake.deleted == nil || aws.ToString(fake.deleted.QueueUrl) != url {
		t.Fatalf("DeleteQueue input = %+v, want QueueUrl=%s", fake.deleted, url)
	}
}

func TestLocalStackPurgeQueue(t *testing.T) {
	const url = "http://localhost:4566/000000000000/orders"
	fake := &fakeSQS{}
	ls := LocalStack{openSQS: func(Config) (sqsAPI, error) { return fake, nil }}

	if err := ls.PurgeQueue(context.Background(), Spec{Env: Config{"host": "localhost"}}, url); err != nil {
		t.Fatalf("PurgeQueue: %v", err)
	}
	if fake.purged == nil || aws.ToString(fake.purged.QueueUrl) != url {
		t.Fatalf("PurgeQueue input = %+v, want QueueUrl=%s", fake.purged, url)
	}
}

func TestLocalStackMessages(t *testing.T) {
	const url = "http://localhost:4566/000000000000/orders"
	fake := &fakeSQS{messages: []sqstypes.Message{
		{
			MessageId: aws.String("m-1"),
			Body:      aws.String(`{"order":1}`),
			Attributes: map[string]string{
				"SentTimestamp":           "1700000000000",
				"ApproximateReceiveCount": "3",
			},
		},
		// A message with no attributes: counts default to zero, not error.
		{MessageId: aws.String("m-2"), Body: aws.String("hello")},
	}}
	ls := LocalStack{openSQS: func(Config) (sqsAPI, error) { return fake, nil }}

	msgs, err := ls.Messages(context.Background(), Spec{Env: Config{"host": "localhost"}}, url, 0)
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}
	// A zero/over-cap limit must clamp to the SQS per-call maximum, and the
	// preview must be non-destructive (zero visibility timeout).
	if aws.ToString(fake.received.QueueUrl) != url {
		t.Errorf("QueueUrl = %q, want %q", aws.ToString(fake.received.QueueUrl), url)
	}
	if fake.received.MaxNumberOfMessages != maxQueueMessages {
		t.Errorf("MaxNumberOfMessages = %d, want %d", fake.received.MaxNumberOfMessages, maxQueueMessages)
	}
	if fake.received.VisibilityTimeout != 0 {
		t.Errorf("VisibilityTimeout = %d, want 0 (non-destructive peek)", fake.received.VisibilityTimeout)
	}
	if len(msgs) != 2 {
		t.Fatalf("got %d messages, want 2", len(msgs))
	}
	if msgs[0].ID != "m-1" || msgs[0].Body != `{"order":1}` {
		t.Errorf("msg[0] = %+v, want id=m-1 body={\"order\":1}", msgs[0])
	}
	if msgs[0].SentAt != 1700000000000 || msgs[0].ReceiveCount != 3 {
		t.Errorf("msg[0] meta = (sentAt=%d, receiveCount=%d), want (1700000000000, 3)", msgs[0].SentAt, msgs[0].ReceiveCount)
	}
	if msgs[1].ID != "m-2" || msgs[1].SentAt != 0 || msgs[1].ReceiveCount != 0 {
		t.Errorf("msg[1] = %+v, want id=m-2 with zero meta", msgs[1])
	}
}

func TestLocalStackMessagesLimit(t *testing.T) {
	fake := &fakeSQS{}
	ls := LocalStack{openSQS: func(Config) (sqsAPI, error) { return fake, nil }}

	if _, err := ls.Messages(context.Background(), Spec{Env: Config{"host": "localhost"}}, "url", 3); err != nil {
		t.Fatalf("Messages: %v", err)
	}
	if fake.received.MaxNumberOfMessages != 3 {
		t.Errorf("MaxNumberOfMessages = %d, want 3 (in-range limit honoured)", fake.received.MaxNumberOfMessages)
	}
}

func TestLocalStackMessagesError(t *testing.T) {
	ls := LocalStack{openSQS: func(Config) (sqsAPI, error) {
		return &fakeSQS{receiveErr: errors.New("connection refused")}, nil
	}}
	if _, err := ls.Messages(context.Background(), Spec{Env: Config{"host": "localhost"}}, "url", 0); err == nil {
		t.Fatal("expected an error when ReceiveMessage fails")
	}
}

func TestLocalStackIdentities(t *testing.T) {
	fake := fakeSES{
		ids: []string{"dev@example.com", "example.org"},
		verify: map[string]sestypes.VerificationStatus{
			"dev@example.com": sestypes.VerificationStatusSuccess,
			"example.org":     sestypes.VerificationStatusPending,
		},
	}
	ls := LocalStack{openSES: func(Config) (sesAPI, error) { return fake, nil }}

	ids, err := ls.Identities(context.Background(), Spec{Env: Config{"host": "localhost"}})
	if err != nil {
		t.Fatalf("Identities: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("got %d identities, want 2", len(ids))
	}
	if ids[0].Identity != "dev@example.com" || ids[0].Type != "EMAIL_ADDRESS" || !ids[0].Verified {
		t.Errorf("id[0] = %+v, want verified email", ids[0])
	}
	if ids[1].Identity != "example.org" || ids[1].Type != "DOMAIN" || ids[1].Verified {
		t.Errorf("id[1] = %+v, want unverified domain", ids[1])
	}
}

func TestLocalStackCloudHealth(t *testing.T) {
	const body = `{"services":{"sqs":"running","s3":"available"},"version":"4.0.3","edition":"community"}`
	ls := LocalStack{openHealth: func(_ context.Context, endpoint string) ([]byte, error) {
		if endpoint != "http://localhost:4566" {
			t.Errorf("endpoint = %q, want http://localhost:4566", endpoint)
		}
		return []byte(body), nil
	}}

	health, err := ls.CloudHealth(context.Background(), Spec{Env: Config{"host": "localhost"}})
	if err != nil {
		t.Fatalf("CloudHealth: %v", err)
	}
	if health.Version != "4.0.3" || health.Edition != "community" {
		t.Errorf("version/edition = %q/%q, want 4.0.3/community", health.Version, health.Edition)
	}
	if health.Services["sqs"] != "running" || health.Services["s3"] != "available" {
		t.Errorf("services = %v, want sqs=running s3=available", health.Services)
	}
}

func TestLocalStackCloudHealthUnreachable(t *testing.T) {
	ls := LocalStack{openHealth: func(context.Context, string) ([]byte, error) {
		return nil, errors.New("connection refused")
	}}

	if _, err := ls.CloudHealth(context.Background(), Spec{Env: Config{"host": "localhost"}}); err == nil {
		t.Fatal("expected an error when the health GET fails")
	}
}

// LocalStack must satisfy the CloudBrowser capability so the server can
// type-assert for it, mirroring how redis satisfies KeyBrowser.
var _ CloudBrowser = LocalStack{}
