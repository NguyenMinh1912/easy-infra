package service

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// sqsAPI is the subset of the SQS client the LocalStack cloud browser relies on.
// Depending on an interface (rather than *sqs.Client directly) lets tests inject
// a fake that returns canned replies without a live LocalStack — mirroring how
// minio depends on s3Client and redis on redisClient. *sqs.Client satisfies it.
type sqsAPI interface {
	ListQueues(ctx context.Context, params *sqs.ListQueuesInput, optFns ...func(*sqs.Options)) (*sqs.ListQueuesOutput, error)
	GetQueueAttributes(ctx context.Context, params *sqs.GetQueueAttributesInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error)
	CreateQueue(ctx context.Context, params *sqs.CreateQueueInput, optFns ...func(*sqs.Options)) (*sqs.CreateQueueOutput, error)
	DeleteQueue(ctx context.Context, params *sqs.DeleteQueueInput, optFns ...func(*sqs.Options)) (*sqs.DeleteQueueOutput, error)
	PurgeQueue(ctx context.Context, params *sqs.PurgeQueueInput, optFns ...func(*sqs.Options)) (*sqs.PurgeQueueOutput, error)
}

// sesAPI is the subset of the SES (v1) client the LocalStack cloud browser
// relies on. *ses.Client satisfies it. The v1 API is used rather than SESv2
// because LocalStack's community edition does not implement sesv2 — its
// endpoints return HTTP 501 ("not yet implemented or pro feature").
type sesAPI interface {
	ListIdentities(ctx context.Context, params *ses.ListIdentitiesInput, optFns ...func(*ses.Options)) (*ses.ListIdentitiesOutput, error)
	GetIdentityVerificationAttributes(ctx context.Context, params *ses.GetIdentityVerificationAttributesInput, optFns ...func(*ses.Options)) (*ses.GetIdentityVerificationAttributesOutput, error)
}

// sqsOpener / sesOpener establish a client for the LocalStack endpoint described
// by env. They are seams: the zero-value LocalStack dials a real endpoint via
// the AWS SDK (realSQSOpener / realSESOpener), while tests supply a fake.
type sqsOpener func(env Config) (sqsAPI, error)
type sesOpener func(env Config) (sesAPI, error)

// healthGetter fetches the raw `/_localstack/health` JSON body for the endpoint.
// It is a seam like the openers: the zero-value LocalStack does a real HTTP GET
// (realHealthGetter), while tests supply canned bytes without a live endpoint.
type healthGetter func(ctx context.Context, endpoint string) ([]byte, error)

// realHealthGetter does the live GET against LocalStack's health endpoint,
// capping the body so a misbehaving endpoint can't stream unbounded data.
func realHealthGetter(ctx context.Context, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"/_localstack/health", nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("health endpoint returned %s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
}

// localstackParams is a profile's LocalStack connection settings, normalised out
// of the discrete host/port/region fields so the openers treat them uniformly.
type localstackParams struct {
	host   string
	port   int
	region string
}

// endpoint is the base URL the AWS APIs are reached on. LocalStack serves every
// service from one HTTP edge port (4566 by default), so all clients share it.
func (p localstackParams) endpoint() string {
	return fmt.Sprintf("http://%s:%d", p.host, p.port)
}

// localstackParamsFrom extracts the connection settings from a profile's env.
func localstackParamsFrom(env Config) (localstackParams, error) {
	host, err := requireString(env, "host")
	if err != nil {
		return localstackParams{}, err
	}
	port, err := optionalPort(env, "port", 4566)
	if err != nil {
		return localstackParams{}, err
	}
	region, err := optionalString(env, "region", "us-east-1")
	if err != nil {
		return localstackParams{}, err
	}
	return localstackParams{host: host, port: port, region: region}, nil
}

// awsConfig builds an SDK config pointed at the LocalStack endpoint. LocalStack
// ignores credentials, but the SDK requires non-empty ones to sign requests, so
// static dummy values are supplied.
func awsConfig(p localstackParams) aws.Config {
	return aws.Config{
		Region:      p.region,
		Credentials: credentials.NewStaticCredentialsProvider("test", "test", ""),
	}
}

// realSQSOpener builds an SQS client from the profile env, pinned to the
// LocalStack endpoint. The client is lazy — it does not connect until the first
// request — so reachability surfaces on the first call rather than here.
func realSQSOpener(env Config) (sqsAPI, error) {
	p, err := localstackParamsFrom(env)
	if err != nil {
		return nil, err
	}
	return sqs.NewFromConfig(awsConfig(p), func(o *sqs.Options) {
		o.BaseEndpoint = aws.String(p.endpoint())
	}), nil
}

// realSESOpener builds an SES (v1) client from the profile env, pinned to the
// LocalStack endpoint.
func realSESOpener(env Config) (sesAPI, error) {
	p, err := localstackParamsFrom(env)
	if err != nil {
		return nil, err
	}
	return ses.NewFromConfig(awsConfig(p), func(o *ses.Options) {
		o.BaseEndpoint = aws.String(p.endpoint())
	}), nil
}
