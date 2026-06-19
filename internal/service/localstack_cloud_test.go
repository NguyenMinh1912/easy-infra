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
// verification status map, mirroring the SES v1 two-call shape. It also records
// the inputs of the mutating calls so tests can assert on them.
type fakeSES struct {
	ids     []string
	verify  map[string]sestypes.VerificationStatus
	listErr error

	verifiedEmail  *ses.VerifyEmailIdentityInput
	verifiedDomain *ses.VerifyDomainIdentityInput
	deleted        *ses.DeleteIdentityInput
	mutErr         error
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

func (f *fakeSES) VerifyEmailIdentity(_ context.Context, in *ses.VerifyEmailIdentityInput, _ ...func(*ses.Options)) (*ses.VerifyEmailIdentityOutput, error) {
	if f.mutErr != nil {
		return nil, f.mutErr
	}
	f.verifiedEmail = in
	return &ses.VerifyEmailIdentityOutput{}, nil
}

func (f *fakeSES) VerifyDomainIdentity(_ context.Context, in *ses.VerifyDomainIdentityInput, _ ...func(*ses.Options)) (*ses.VerifyDomainIdentityOutput, error) {
	if f.mutErr != nil {
		return nil, f.mutErr
	}
	f.verifiedDomain = in
	return &ses.VerifyDomainIdentityOutput{}, nil
}

func (f *fakeSES) DeleteIdentity(_ context.Context, in *ses.DeleteIdentityInput, _ ...func(*ses.Options)) (*ses.DeleteIdentityOutput, error) {
	if f.mutErr != nil {
		return nil, f.mutErr
	}
	f.deleted = in
	return &ses.DeleteIdentityOutput{}, nil
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
	ls := LocalStack{openSES: func(Config) (sesAPI, error) { return &fake, nil }}

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

func TestLocalStackCreateIdentityEmail(t *testing.T) {
	fake := &fakeSES{}
	ls := LocalStack{openSES: func(Config) (sesAPI, error) { return fake, nil }}

	if err := ls.CreateIdentity(context.Background(), Spec{Env: Config{"host": "localhost"}}, "dev@example.com"); err != nil {
		t.Fatalf("CreateIdentity: %v", err)
	}
	if fake.verifiedEmail == nil || aws.ToString(fake.verifiedEmail.EmailAddress) != "dev@example.com" {
		t.Fatalf("VerifyEmailIdentity input = %+v, want EmailAddress=dev@example.com", fake.verifiedEmail)
	}
	if fake.verifiedDomain != nil {
		t.Errorf("email identity must not verify a domain, got %+v", fake.verifiedDomain)
	}
}

func TestLocalStackCreateIdentityDomain(t *testing.T) {
	fake := &fakeSES{}
	ls := LocalStack{openSES: func(Config) (sesAPI, error) { return fake, nil }}

	if err := ls.CreateIdentity(context.Background(), Spec{Env: Config{"host": "localhost"}}, "example.org"); err != nil {
		t.Fatalf("CreateIdentity: %v", err)
	}
	if fake.verifiedDomain == nil || aws.ToString(fake.verifiedDomain.Domain) != "example.org" {
		t.Fatalf("VerifyDomainIdentity input = %+v, want Domain=example.org", fake.verifiedDomain)
	}
	if fake.verifiedEmail != nil {
		t.Errorf("domain identity must not verify an email, got %+v", fake.verifiedEmail)
	}
}

func TestLocalStackCreateIdentityError(t *testing.T) {
	ls := LocalStack{openSES: func(Config) (sesAPI, error) {
		return &fakeSES{mutErr: errors.New("boom")}, nil
	}}
	if err := ls.CreateIdentity(context.Background(), Spec{Env: Config{"host": "localhost"}}, "dev@example.com"); err == nil {
		t.Fatal("expected an error when VerifyEmailIdentity fails")
	}
}

func TestLocalStackDeleteIdentity(t *testing.T) {
	fake := &fakeSES{}
	ls := LocalStack{openSES: func(Config) (sesAPI, error) { return fake, nil }}

	if err := ls.DeleteIdentity(context.Background(), Spec{Env: Config{"host": "localhost"}}, "dev@example.com"); err != nil {
		t.Fatalf("DeleteIdentity: %v", err)
	}
	if fake.deleted == nil || aws.ToString(fake.deleted.Identity) != "dev@example.com" {
		t.Fatalf("DeleteIdentity input = %+v, want Identity=dev@example.com", fake.deleted)
	}
}

func TestLocalStackMessages(t *testing.T) {
	// Two messages from dev@example.com, one unrelated. Oldest first, as the
	// store returns them.
	const body = `{"messages":[
		{"Id":"1","Source":"dev@example.com","Subject":"Hello","Timestamp":"2024-01-01T00:00:00Z","Destination":{"ToAddresses":["a@b.com"],"CcAddresses":["c@b.com"]},"Body":{"text_part":"hi there"}},
		{"Id":"2","Source":"other@nope.com","Subject":"Nope","Destination":{"ToAddresses":["x@y.com"]},"Body":{"text_part":"ignore"}},
		{"Id":"3","Source":"sender@y.com","Subject":"To dev","Timestamp":"2024-01-02T00:00:00Z","Destination":{"ToAddresses":["dev@example.com"]},"Body":{"html_part":"<p>hey</p>"}}
	]}`
	ls := LocalStack{openMessages: func(_ context.Context, endpoint string) ([]byte, error) {
		if endpoint != "http://localhost:4566" {
			t.Errorf("endpoint = %q, want http://localhost:4566", endpoint)
		}
		return []byte(body), nil
	}}

	msgs, err := ls.Messages(context.Background(), Spec{Env: Config{"host": "localhost"}}, "dev@example.com")
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("got %d messages, want 2 (the unrelated one filtered out)", len(msgs))
	}
	// Newest first: id 3 (recipient match) then id 1 (sender match).
	if msgs[0].ID != "3" || msgs[1].ID != "1" {
		t.Errorf("order = [%s,%s], want [3,1] (newest first)", msgs[0].ID, msgs[1].ID)
	}
	if msgs[1].Subject != "Hello" || msgs[1].Body != "hi there" {
		t.Errorf("msg[1] = %+v, want subject=Hello body=\"hi there\"", msgs[1])
	}
	if len(msgs[1].Destination) != 2 {
		t.Errorf("msg[1] destinations = %v, want To+Cc combined", msgs[1].Destination)
	}
	// HTML body is the fallback when there is no text part.
	if msgs[0].Body != "<p>hey</p>" {
		t.Errorf("msg[0] body = %q, want the html part as fallback", msgs[0].Body)
	}
}

func TestLocalStackMessagesDomainIdentity(t *testing.T) {
	const body = `{"messages":[
		{"Id":"1","Source":"dev@example.com","Destination":{"ToAddresses":["a@b.com"]}},
		{"Id":"2","Source":"x@y.com","Destination":{"ToAddresses":["ops@example.com"]}},
		{"Id":"3","Source":"x@y.com","Destination":{"ToAddresses":["a@b.com"]}}
	]}`
	ls := LocalStack{openMessages: func(context.Context, string) ([]byte, error) {
		return []byte(body), nil
	}}

	msgs, err := ls.Messages(context.Background(), Spec{Env: Config{"host": "localhost"}}, "example.com")
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}
	// A domain identity matches any address at the domain — id 1 (sender) and
	// id 2 (recipient), but not id 3.
	if len(msgs) != 2 {
		t.Fatalf("got %d messages, want 2 at the domain", len(msgs))
	}
}

func TestLocalStackMessagesUnreachable(t *testing.T) {
	ls := LocalStack{openMessages: func(context.Context, string) ([]byte, error) {
		return nil, errors.New("connection refused")
	}}
	if _, err := ls.Messages(context.Background(), Spec{Env: Config{"host": "localhost"}}, "dev@example.com"); err == nil {
		t.Fatal("expected an error when the messages GET fails")
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
