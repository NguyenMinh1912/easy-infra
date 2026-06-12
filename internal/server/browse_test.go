package server

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
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
	listings   map[string]*service.ObjectListing
	listErr    error
	object     string
	objects    map[string]string
	objectInfo service.ObjectContent
	objectErr  error
	gotBucket  string
	gotPrefix  string
	gotKey     string
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
	// A per-prefix map (when set) serves the recursive archive walk; otherwise
	// the single canned listing answers any prefix.
	if s.listings != nil {
		return s.listings[prefix], s.listErr
	}
	return s.listing, s.listErr
}

func (s *stubBrowser) Object(_ context.Context, spec service.Spec, bucket, key string) (io.ReadCloser, service.ObjectContent, error) {
	s.gotSpec = spec
	s.gotBucket = bucket
	s.gotKey = key
	if s.objectErr != nil {
		return nil, service.ObjectContent{}, s.objectErr
	}
	// Per-key bodies (when set) let an archive test distinguish entries;
	// otherwise every object yields the single canned body.
	body := s.object
	if s.objects != nil {
		body = s.objects[key]
	}
	return io.NopCloser(strings.NewReader(body)), s.objectInfo, nil
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

func TestBrowseObjectDownload(t *testing.T) {
	stub := &stubBrowser{
		stubService: stubService{name: "stub"},
		object:      "PNGDATA",
		objectInfo:  service.ObjectContent{Size: 7, ContentType: "image/png"},
	}
	srv := newConsoleServer(t, stub)
	rec := doJSON(t, srv, http.MethodGet, "/api/profiles/default/services/stub/object?bucket=assets&key=docs/logo.png", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "PNGDATA" {
		t.Errorf("body = %q, want PNGDATA", rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("Content-Type = %q, want image/png", ct)
	}
	if cd := rec.Header().Get("Content-Disposition"); cd != `attachment; filename="logo.png"` {
		t.Errorf("Content-Disposition = %q, want attachment with base name", cd)
	}
	if stub.gotBucket != "assets" || stub.gotKey != "docs/logo.png" {
		t.Errorf("got bucket=%q key=%q, want assets/docs/logo.png", stub.gotBucket, stub.gotKey)
	}
}

func TestBrowseArchiveKeysAndPrefix(t *testing.T) {
	stub := &stubBrowser{
		stubService: stubService{name: "stub"},
		// "docs/" expands to its two objects; "top.txt" is taken as-is.
		listings: map[string]*service.ObjectListing{
			"docs/": {
				Prefixes: []string{"docs/img/"},
				Objects:  []service.ObjectEntry{{Key: "docs/readme.md"}},
			},
			"docs/img/": {
				Objects: []service.ObjectEntry{{Key: "docs/img/logo.png"}},
			},
		},
		objects: map[string]string{
			"top.txt":           "TOP",
			"docs/readme.md":    "README",
			"docs/img/logo.png": "PNG",
		},
	}
	srv := newConsoleServer(t, stub)
	rec := doJSON(t, srv, http.MethodGet,
		"/api/profiles/default/services/stub/objects/archive?bucket=assets&key=top.txt&prefix=docs/", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/zip" {
		t.Errorf("Content-Type = %q, want application/zip", ct)
	}
	if cd := rec.Header().Get("Content-Disposition"); cd != `attachment; filename="assets.zip"` {
		t.Errorf("Content-Disposition = %q, want assets.zip attachment", cd)
	}

	zr, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	got := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open entry %q: %v", f.Name, err)
		}
		data, _ := io.ReadAll(rc)
		rc.Close()
		got[f.Name] = string(data)
	}
	want := map[string]string{
		"top.txt":           "TOP",
		"docs/readme.md":    "README",
		"docs/img/logo.png": "PNG",
	}
	if len(got) != len(want) {
		t.Fatalf("archive entries = %v, want %v", got, want)
	}
	for name, body := range want {
		if got[name] != body {
			t.Errorf("entry %q = %q, want %q", name, got[name], body)
		}
	}
}

func TestBrowseArchiveNoSelection(t *testing.T) {
	srv := newConsoleServer(t, &stubBrowser{stubService: stubService{name: "stub"}})
	rec := doJSON(t, srv, http.MethodGet,
		"/api/profiles/default/services/stub/objects/archive?bucket=assets", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body %q)", rec.Code, rec.Body.String())
	}
}

func TestBrowseArchiveListError(t *testing.T) {
	stub := &stubBrowser{
		stubService: stubService{name: "stub"},
		listErr:     errors.New("store unreachable"),
	}
	srv := newConsoleServer(t, stub)
	rec := doJSON(t, srv, http.MethodGet,
		"/api/profiles/default/services/stub/objects/archive?bucket=assets&prefix=docs/", nil)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502 (body %q)", rec.Code, rec.Body.String())
	}
}

func TestBrowseObjectMissingKey(t *testing.T) {
	srv := newConsoleServer(t, &stubBrowser{stubService: stubService{name: "stub"}})
	rec := doJSON(t, srv, http.MethodGet, "/api/profiles/default/services/stub/object?bucket=assets", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body %q)", rec.Code, rec.Body.String())
	}
}

func TestBrowseObjectError(t *testing.T) {
	stub := &stubBrowser{
		stubService: stubService{name: "stub"},
		objectErr:   errors.New("store unreachable"),
	}
	srv := newConsoleServer(t, stub)
	rec := doJSON(t, srv, http.MethodGet, "/api/profiles/default/services/stub/object?bucket=assets&key=x", nil)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502 (body %q)", rec.Code, rec.Body.String())
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
