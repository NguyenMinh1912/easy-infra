package service

import (
	"context"
	"errors"
	"testing"
)

func TestLocalStackHealthy(t *testing.T) {
	ls := LocalStack{openSQS: func(Config) (sqsAPI, error) { return &fakeSQS{}, nil }}

	if err := ls.Health(context.Background(), Spec{Env: Config{"host": "localhost"}}); err != nil {
		t.Fatalf("Health: %v", err)
	}
}

func TestLocalStackHealthUnreachable(t *testing.T) {
	ls := LocalStack{openSQS: func(Config) (sqsAPI, error) {
		return &fakeSQS{listErr: errors.New("connection refused")}, nil
	}}

	if err := ls.Health(context.Background(), Spec{Env: Config{"host": "localhost"}}); err == nil {
		t.Fatal("expected an error when LocalStack is unreachable")
	}
}
