package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sort"
	"strings"
	"testing"
)

// fakeS3 is an in-memory s3Client standing in for a real MinIO server, so the
// minio lifecycle can be tested without a daemon. Objects are keyed by bucket
// then object key; an empty bucket is a present-but-empty inner map.
type fakeS3 struct {
	buckets  map[string]map[string][]byte
	types    map[string]map[string]string
	listErr  error // when set, ListBuckets fails (used to exercise Health)
	bucketed bool  // tracks whether the store has been seeded
}

func newFakeS3() *fakeS3 {
	return &fakeS3{
		buckets: map[string]map[string][]byte{},
		types:   map[string]map[string]string{},
	}
}

func (f *fakeS3) put(bucket, key string, data []byte, contentType string) {
	if f.buckets[bucket] == nil {
		f.buckets[bucket] = map[string][]byte{}
		f.types[bucket] = map[string]string{}
	}
	if key != "" {
		f.buckets[bucket][key] = data
		f.types[bucket][key] = contentType
	}
	f.bucketed = true
}

func (f *fakeS3) ListBuckets(context.Context) ([]string, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	names := make([]string, 0, len(f.buckets))
	for b := range f.buckets {
		names = append(names, b)
	}
	sort.Strings(names)
	return names, nil
}

func (f *fakeS3) BucketExists(_ context.Context, bucket string) (bool, error) {
	_, ok := f.buckets[bucket]
	return ok, nil
}

func (f *fakeS3) MakeBucket(_ context.Context, bucket string) error {
	if _, ok := f.buckets[bucket]; !ok {
		f.buckets[bucket] = map[string][]byte{}
		f.types[bucket] = map[string]string{}
	}
	return nil
}

func (f *fakeS3) RemoveBucket(_ context.Context, bucket string) error {
	delete(f.buckets, bucket)
	delete(f.types, bucket)
	return nil
}

func (f *fakeS3) ListObjects(_ context.Context, bucket string) ([]string, error) {
	objs := f.buckets[bucket]
	keys := make([]string, 0, len(objs))
	for k := range objs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, nil
}

func (f *fakeS3) ListObjectsPage(_ context.Context, bucket, prefix string) (objectPage, error) {
	var page objectPage
	seen := map[string]bool{}
	for k := range f.buckets[bucket] {
		if k == "" || !strings.HasPrefix(k, prefix) {
			continue
		}
		rest := k[len(prefix):]
		if i := strings.IndexByte(rest, '/'); i >= 0 {
			// A nested key: surface its immediate sub-folder once.
			folder := prefix + rest[:i+1]
			if !seen[folder] {
				seen[folder] = true
				page.prefixes = append(page.prefixes, folder)
			}
			continue
		}
		page.objects = append(page.objects, objectListEntry{
			key:         k,
			size:        int64(len(f.buckets[bucket][k])),
			contentType: f.types[bucket][k],
		})
	}
	sort.Strings(page.prefixes)
	sort.Slice(page.objects, func(i, j int) bool { return page.objects[i].key < page.objects[j].key })
	return page, nil
}

func (f *fakeS3) GetObject(_ context.Context, bucket, key string) (io.ReadCloser, objectInfo, error) {
	data, ok := f.buckets[bucket][key]
	if !ok {
		return nil, objectInfo{}, errors.New("no such object")
	}
	return io.NopCloser(bytes.NewReader(data)), objectInfo{Size: int64(len(data)), ContentType: f.types[bucket][key]}, nil
}

func (f *fakeS3) PutObject(_ context.Context, bucket, key string, r io.Reader, _ int64, contentType string) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	f.put(bucket, key, data, contentType)
	return nil
}

func (f *fakeS3) RemoveObject(_ context.Context, bucket, key string) error {
	delete(f.buckets[bucket], key)
	delete(f.types[bucket], key)
	return nil
}

// withS3 builds a MinIO whose opener always returns the given fake client.
func withS3(f *fakeS3) MinIO {
	return MinIO{open: func(context.Context, minioParams) (s3Client, error) { return f, nil }}
}

func minioEnv() Config {
	return Config{"host": "localhost", "port": 9000, "consolePort": 9001, "user": "minioadmin", "password": "minioadmin"}
}

// TestMinIOHealth confirms Health passes when the server lists buckets and fails
// (without panicking) when the listing errors.
func TestMinIOHealth(t *testing.T) {
	f := newFakeS3()
	if err := withS3(f).Health(context.Background(), Spec{Env: minioEnv()}); err != nil {
		t.Fatalf("Health (healthy): %v", err)
	}
	f.listErr = errors.New("connection refused")
	if err := withS3(f).Health(context.Background(), Spec{Env: minioEnv()}); err == nil {
		t.Error("Health: expected error when ListBuckets fails")
	}
}

// TestMinIOBackupRestoreRoundTrip drives the fork's data path: back up the store
// to a snapshot, wipe it with Clean, then Apply the snapshot back and confirm the
// buckets and objects (including an empty bucket and content types) return.
func TestMinIOBackupRestoreRoundTrip(t *testing.T) {
	t.Chdir(t.TempDir()) // backups land under a throwaway cwd

	f := newFakeS3()
	f.put("assets", "logo.png", []byte("PNGDATA"), "image/png")
	f.put("assets", "docs/readme.txt", []byte("hello"), "text/plain")
	f.put("empty", "", nil, "") // an empty bucket must survive the round-trip

	m := withS3(f)
	const profile = "staging"
	spec := Spec{Profile: profile, Env: minioEnv()}

	if err := m.Backup(context.Background(), spec); err != nil {
		t.Fatalf("Backup: %v", err)
	}
	if err := m.Clean(context.Background(), Spec{Profile: profile, Env: minioEnv(), Definition: Config{cleanableKey: true}}); err != nil {
		t.Fatalf("Clean: %v", err)
	}
	if len(f.buckets) != 0 {
		t.Fatalf("after Clean, store has %d buckets, want 0", len(f.buckets))
	}
	if err := m.Apply(context.Background(), spec); err != nil {
		t.Fatalf("Apply (restore): %v", err)
	}

	if _, ok := f.buckets["empty"]; !ok {
		t.Error("empty bucket was not restored")
	}
	if got := string(f.buckets["assets"]["logo.png"]); got != "PNGDATA" {
		t.Errorf("logo.png = %q, want PNGDATA", got)
	}
	if got := string(f.buckets["assets"]["docs/readme.txt"]); got != "hello" {
		t.Errorf("docs/readme.txt = %q, want hello", got)
	}
	if ct := f.types["assets"]["logo.png"]; ct != "image/png" {
		t.Errorf("logo.png content type = %q, want image/png", ct)
	}
}

// TestMinIOApplyNoSnapshot confirms Apply is a no-op (not an error) when the
// profile has no snapshot yet.
func TestMinIOApplyNoSnapshot(t *testing.T) {
	t.Chdir(t.TempDir())
	f := newFakeS3()
	if err := withS3(f).Apply(context.Background(), Spec{Profile: "fresh", Env: minioEnv()}); err != nil {
		t.Fatalf("Apply with no snapshot: %v", err)
	}
}

// TestMinIOApplyCreatesConfiguredBuckets confirms Apply creates the buckets
// declared in the profile definition even when no snapshot exists, and that it
// is idempotent for buckets that already exist.
func TestMinIOApplyCreatesConfiguredBuckets(t *testing.T) {
	t.Chdir(t.TempDir())
	f := newFakeS3()
	f.put("assets", "", nil, "") // already present; must be left intact

	spec := Spec{
		Profile:    "fresh",
		Env:        minioEnv(),
		Definition: Config{"buckets": []string{"assets", "uploads"}},
	}
	if err := withS3(f).Apply(context.Background(), spec); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	for _, want := range []string{"assets", "uploads"} {
		if _, ok := f.buckets[want]; !ok {
			t.Errorf("bucket %q was not created", want)
		}
	}
}

// TestMinIOValidateDefinitionBuckets confirms the buckets field is validated:
// well-formed names (as a list or a comma-separated string) pass, and an
// invalid name is rejected.
func TestMinIOValidateDefinitionBuckets(t *testing.T) {
	valid := []Config{
		{"buckets": []string{"assets", "user-uploads"}},
		{"buckets": []any{"assets", "logs.archive"}},
		{"buckets": "assets, uploads"}, // web UI submits one comma-separated string
		{"buckets": ""},                // empty is allowed (no buckets)
	}
	for _, cfg := range valid {
		if err := (MinIO{}).ValidateDefinition(cfg); err != nil {
			t.Errorf("ValidateDefinition(%v): unexpected error %v", cfg, err)
		}
	}

	invalid := []Config{
		{"buckets": []string{"AB"}},       // too short and uppercase
		{"buckets": []string{"Bad_Name"}}, // illegal characters
		{"buckets": []string{"-leading"}}, // must start with a letter or digit
		{"buckets": []any{"ok", 7}},       // non-string element
	}
	for _, cfg := range invalid {
		if err := (MinIO{}).ValidateDefinition(cfg); err == nil {
			t.Errorf("ValidateDefinition(%v): expected an error, got nil", cfg)
		}
	}
}

// TestMinIOBrowse drives the Browser capability: Buckets lists the store's
// buckets, and Objects lists one folder level — the immediate sub-folders as
// prefixes and the objects directly at that level, not recursively.
func TestMinIOBrowse(t *testing.T) {
	f := newFakeS3()
	f.put("assets", "logo.png", []byte("PNGDATA"), "image/png")
	f.put("assets", "docs/readme.txt", []byte("hello"), "text/plain")
	f.put("assets", "docs/guide/intro.md", []byte("# intro"), "text/markdown")
	f.put("empty", "", nil, "")

	m := withS3(f)
	spec := Spec{Env: minioEnv()}

	buckets, err := m.Buckets(context.Background(), spec)
	if err != nil {
		t.Fatalf("Buckets: %v", err)
	}
	if len(buckets) != 2 || buckets[0] != "assets" || buckets[1] != "empty" {
		t.Fatalf("Buckets = %v, want [assets empty]", buckets)
	}

	// Root of "assets": one object (logo.png) and one sub-folder (docs/).
	root, err := m.Objects(context.Background(), spec, "assets", "")
	if err != nil {
		t.Fatalf("Objects(root): %v", err)
	}
	if len(root.Prefixes) != 1 || root.Prefixes[0] != "docs/" {
		t.Errorf("root prefixes = %v, want [docs/]", root.Prefixes)
	}
	if len(root.Objects) != 1 || root.Objects[0].Key != "logo.png" {
		t.Fatalf("root objects = %+v, want only logo.png", root.Objects)
	}
	if root.Objects[0].Size != 7 || root.Objects[0].ContentType != "image/png" {
		t.Errorf("logo.png entry = %+v, want size 7, image/png", root.Objects[0])
	}

	// Inside "docs/": one object (readme.txt) and one sub-folder (docs/guide/).
	docs, err := m.Objects(context.Background(), spec, "assets", "docs/")
	if err != nil {
		t.Fatalf("Objects(docs/): %v", err)
	}
	if len(docs.Prefixes) != 1 || docs.Prefixes[0] != "docs/guide/" {
		t.Errorf("docs/ prefixes = %v, want [docs/guide/]", docs.Prefixes)
	}
	if len(docs.Objects) != 1 || docs.Objects[0].Key != "docs/readme.txt" {
		t.Errorf("docs/ objects = %+v, want only docs/readme.txt", docs.Objects)
	}
}

// TestMinIOCleanProtected confirms a protected definition blocks Clean before any
// client work happens.
func TestMinIOCleanProtected(t *testing.T) {
	f := newFakeS3()
	f.put("assets", "x", []byte("y"), "")
	spec := Spec{Env: minioEnv(), Definition: Config{cleanableKey: false}}
	if err := withS3(f).Clean(context.Background(), spec); !errors.Is(err, ErrProtected) {
		t.Fatalf("Clean: got %v, want ErrProtected", err)
	}
	if _, ok := f.buckets["assets"]; !ok {
		t.Error("protected Clean removed data")
	}
}
