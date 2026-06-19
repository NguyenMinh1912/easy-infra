package service

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// sqsAPI is the subset of the SQS client the LocalStack cloud browser relies on.
// Depending on an interface (rather than *sqs.Client directly) lets tests inject
// a fake that returns canned replies without a live LocalStack — mirroring how
// minio depends on s3Client and redis on redisClient. *sqs.Client satisfies it.
type sqsAPI interface {
	ListQueues(ctx context.Context, params *sqs.ListQueuesInput, optFns ...func(*sqs.Options)) (*sqs.ListQueuesOutput, error)
	GetQueueAttributes(ctx context.Context, params *sqs.GetQueueAttributesInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error)
}

// sesAPI is the subset of the SESv2 client the LocalStack cloud browser relies
// on. *sesv2.Client satisfies it.
type sesAPI interface {
	ListEmailIdentities(ctx context.Context, params *sesv2.ListEmailIdentitiesInput, optFns ...func(*sesv2.Options)) (*sesv2.ListEmailIdentitiesOutput, error)
}

// sqsOpener / sesOpener establish a client for the LocalStack endpoint described
// by env. They are seams: the zero-value LocalStack dials a real endpoint via
// the AWS SDK (realSQSOpener / realSESOpener), while tests supply a fake.
type sqsOpener func(env Config) (sqsAPI, error)
type sesOpener func(env Config) (sesAPI, error)

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

// realSESOpener builds an SESv2 client from the profile env, pinned to the
// LocalStack endpoint.
func realSESOpener(env Config) (sesAPI, error) {
	p, err := localstackParamsFrom(env)
	if err != nil {
		return nil, err
	}
	return sesv2.NewFromConfig(awsConfig(p), func(o *sesv2.Options) {
		o.BaseEndpoint = aws.String(p.endpoint())
	}), nil
}
