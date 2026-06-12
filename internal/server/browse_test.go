package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/minhnc/easy-infra/internal/service"
)

// stubBrowser is a browse-capable stubService serving canned bucket and object
// listings, recording the bucket/prefix it was asked to list.
type stubBrowser struct {
	stubService
	buckets    []string
	bucketsErr error
	listing    *service.ObjectListing
	listErr    error
	gotBucket  string
	gotPrefix  string
	gotSpec    service.Spec
}

func (s *stubBrowser) Buckets(_ context.Context, spec service.Spec) ([]string, error) {
	s.gotSpec = spec
	return s.buckets, s.bucketsErr
}

func (s *stubBrowser) Objects(_ context.Context, spec service.Spec, bucket, prefix string) (*service.ObjectListing, error) {
	s.gotSpec = spec
	s.gotBucket = bucket
	s.gotPrefix = prefix
	return s.listing, s.listErr
}

func TestBrowseBucketsHappyPath(t *testing.T) {
	stub := &stubBrowser{
		stubService: stubService{name: "stub"},
		buckets:     []string{"assets", "uploads"},
	}
	srv := newConsoleServer(t, stub)
	rec := doJSON(t, srv, http.MethodGet, "/api/profiles/default/services/stub/buckets", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	var got bucketsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode buckets response: %v", err)
	}
	if got.Error != "" || len(got.Buckets) != 2 || got.Buckets[0] != "assets" {
		t.Fatalf("response = %+v", got)
	}
	if stub.gotSpec.Profile != "default" || stub.gotSpec.Env == nil {
		t.Errorf("spec = %+v, want profile default with the saved env", stub.gotSpec)
	}
}

func TestBrowseBucketsError(t *testing.T) {
	stub := &stubBrowser{
		stubService: stubService{name: "stub"},
		bucketsErr:  errors.New("connection refused"),
	}
	srv := newConsoleServer(t, stub)
	rec := doJSON(t, srv, http.MethodGet, "/api/profiles/default/services/stub/buckets", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	var got bucketsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode buckets response: %v", err)
	}
	if got.Error == "" || len(got.Buckets) != 0 {
		t.Fatalf("response = %+v, want error envelope with no buckets", got)
	}
}

func TestBrowseObjectsHappyPath(t *testing.T) {
	stub := &stubBrowser{
		stubService: stubService{name: "stub"},
		listing: &service.ObjectListing{
			Prefixes: []string{"docs/"},
			Objects:  []service.ObjectEntry{{Key: "logo.png", Size: 7, ContentType: "image/png"}},
		},
	}
	srv := newConsoleServer(t, stub)
	rec := doJSON(t, srv, http.MethodGet, "/api/profiles/default/services/stub/objects?bucket=assets&prefix=", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	var got objectsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode objects response: %v", err)
	}
	if got.Error != "" || len(got.Prefixes) != 1 || len(got.Objects) != 1 || got.Objects[0].Key != "logo.png" {
		t.Fatalf("response = %+v", got)
	}
	if stub.gotBucket != "assets" {
		t.Errorf("listed bucket = %q, want assets", stub.gotBucket)
	}
}

func TestBrowseObjectsMissingBucket(t *testing.T) {
	srv := newConsoleServer(t, &stubBrowser{stubService: stubService{name: "stub"}})
	rec := doJSON(t, srv, http.MethodGet, "/api/profiles/default/services/stub/objects", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body %q)", rec.Code, rec.Body.String())
	}
}

func TestBrowseUnsupportedService(t *testing.T) {
	// A plain stubService does not implement service.Browser.
	srv := newConsoleServer(t, stubService{name: "stub"})
	rec := doJSON(t, srv, http.MethodGet, "/api/profiles/default/services/stub/buckets", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body %q)", rec.Code, rec.Body.String())
	}
}

func TestBrowseUnknownProfile(t *testing.T) {
	srv := newConsoleServer(t, &stubBrowser{stubService: stubService{name: "stub"}})
	rec := doJSON(t, srv, http.MethodGet, "/api/profiles/nope/services/stub/buckets", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body %q)", rec.Code, rec.Body.String())
	}
}
