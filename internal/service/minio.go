package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// minioDir is the sub-folder, within a shared snapshot, that holds minio's
// artifact: a manifest plus the object bytes laid out as <bucket>/<key>.
const minioDir = "minio"

// minioManifestFile records the buckets and per-object metadata of a snapshot,
// so empty buckets and content types survive a backup/restore round-trip.
const minioManifestFile = "manifest.json"

// MinIO provisions a MinIO S3-compatible object storage service.
//
// open and docker are seams for testing: when nil the lifecycle dials a real
// server via minio-go (realS3Opener) and drives the real `docker` CLI
// (realDocker); tests set them to inject a fake client and container runner.
type MinIO struct {
	open   s3Opener
	docker dockerRunner
}

// opener returns the client opener to use, defaulting to a real minio-go dial.
func (m MinIO) opener() s3Opener {
	if m.open != nil {
		return m.open
	}
	return realS3Opener
}

// dockerClient returns the container runner to use, defaulting to the real
// docker CLI.
func (m MinIO) dockerClient() dockerRunner {
	if m.docker != nil {
		return m.docker
	}
	return realDocker{}
}

// Name implements Service.
func (MinIO) Name() string { return "minio" }

// DefaultDefinition implements Service.
func (MinIO) DefaultDefinition() Config {
	return Config{"version": "latest", cleanableKey: true}
}

// ValidateDefinition implements Service.
func (MinIO) ValidateDefinition(cfg Config) error {
	if _, err := optionalString(cfg, "version", "latest"); err != nil {
		return err
	}
	return validateCleanable(cfg)
}

// DefaultEnv implements Service.
func (MinIO) DefaultEnv() Config {
	return Config{
		"host":        "localhost",
		"port":        9000,
		"consolePort": 9001,
		"user":        "minioadmin",
		"password":    "minioadmin",
	}
}

// ValidateEnv implements Service.
func (MinIO) ValidateEnv(cfg Config) error {
	if _, err := requireString(cfg, "host"); err != nil {
		return err
	}
	if _, err := optionalPort(cfg, "port", 9000); err != nil {
		return err
	}
	if _, err := optionalPort(cfg, "consolePort", 9001); err != nil {
		return err
	}
	if _, err := requireString(cfg, "user"); err != nil {
		return err
	}
	if _, err := requireString(cfg, "password"); err != nil {
		return err
	}
	// secure is optional; when true the client speaks TLS to the endpoint.
	if _, err := optionalBool(cfg, "secure", false); err != nil {
		return err
	}
	return nil
}

// Health implements Service: connect to the configured endpoint and confirm it
// is reachable and the credentials are accepted by listing buckets.
func (m MinIO) Health(ctx context.Context, spec Spec) error {
	client, err := m.connect(ctx, spec.Env)
	if err != nil {
		return err
	}
	if _, err := client.ListBuckets(ctx); err != nil {
		return fmt.Errorf("minio not ready: %w", err)
	}
	return nil
}

// Apply implements Service: if a snapshot exists for the active profile, recreate
// its buckets and re-upload its objects. spec.Snapshot selects which version to
// restore; when empty the latest snapshot is used. With no snapshot yet (or one
// without a minio artifact), Apply is a no-op, leaving the store as-is.
func (m MinIO) Apply(ctx context.Context, spec Spec) error {
	var dir string
	if spec.Snapshot != "" {
		dir = SnapshotDir(spec.Profile, spec.Snapshot)
	} else {
		latest, err := latestSnapshotDir(spec.Profile)
		if err != nil {
			return err
		}
		dir = latest
	}
	if dir == "" {
		spec.logf("no snapshot found for profile %q; leaving minio untouched\n", spec.Profile)
		return nil
	}

	base := filepath.Join(dir, minioDir)
	manifest, err := readMinioManifest(base)
	if err != nil {
		return err
	}
	if manifest == nil {
		spec.logf("snapshot %s has no minio artifact; nothing to restore\n", filepath.Base(dir))
		return nil
	}

	client, err := m.connect(ctx, spec.Env)
	if err != nil {
		return err
	}

	for _, bucket := range manifest.Buckets {
		spec.logf("ensuring bucket %q\n", bucket)
		if err := ensureBucket(ctx, client, bucket); err != nil {
			return err
		}
	}
	for _, obj := range manifest.Objects {
		path, err := safeJoin(base, obj.Bucket, filepath.FromSlash(obj.Key))
		if err != nil {
			return err
		}
		if err := uploadObject(ctx, client, obj, path); err != nil {
			return err
		}
	}
	spec.logf("restored %d object(s) across %d bucket(s)\n", len(manifest.Objects), len(manifest.Buckets))
	return nil
}

// Backup implements Service: download every bucket's objects into the snapshot
// folder, laid out as <bucket>/<key>, alongside a manifest describing the
// buckets and object metadata. The command layer sets spec.BackupDir so every
// service in a snapshot shares one folder; when it is empty Backup creates its
// own fresh snapshot.
func (m MinIO) Backup(ctx context.Context, spec Spec) error {
	client, err := m.connect(ctx, spec.Env)
	if err != nil {
		return err
	}

	dir := spec.BackupDir
	if dir == "" {
		dir = NewSnapshotDir(spec.Profile)
	}
	base := filepath.Join(dir, minioDir)
	if err := os.MkdirAll(base, 0o755); err != nil {
		return fmt.Errorf("creating backup dir %s: %w", base, err)
	}

	buckets, err := client.ListBuckets(ctx)
	if err != nil {
		return fmt.Errorf("listing buckets: %w", err)
	}
	manifest := &minioManifest{Buckets: buckets}
	for _, bucket := range buckets {
		keys, err := client.ListObjects(ctx, bucket)
		if err != nil {
			return fmt.Errorf("listing objects in %q: %w", bucket, err)
		}
		for _, key := range keys {
			info, err := downloadObject(ctx, client, bucket, key, base)
			if err != nil {
				return err
			}
			manifest.Objects = append(manifest.Objects, minioObject{
				Bucket:      bucket,
				Key:         key,
				Size:        info.Size,
				ContentType: info.ContentType,
			})
		}
		spec.logf("backed up %d object(s) from bucket %q\n", len(keys), bucket)
	}
	if err := writeMinioManifest(base, manifest); err != nil {
		return err
	}
	spec.logf("wrote manifest with %d bucket(s), %d object(s)\n", len(manifest.Buckets), len(manifest.Objects))
	return nil
}

// Clean implements Service: empty and remove every bucket, returning the store
// to an empty state. Destructive — callers confirm before invoking.
func (m MinIO) Clean(ctx context.Context, spec Spec) error {
	if err := spec.ensureCleanable(); err != nil {
		return err
	}
	client, err := m.connect(ctx, spec.Env)
	if err != nil {
		return err
	}
	buckets, err := client.ListBuckets(ctx)
	if err != nil {
		return fmt.Errorf("listing buckets: %w", err)
	}
	for _, bucket := range buckets {
		keys, err := client.ListObjects(ctx, bucket)
		if err != nil {
			return fmt.Errorf("listing objects in %q: %w", bucket, err)
		}
		for _, key := range keys {
			if err := client.RemoveObject(ctx, bucket, key); err != nil {
				return fmt.Errorf("removing %s/%s: %w", bucket, key, err)
			}
		}
		if err := client.RemoveBucket(ctx, bucket); err != nil {
			return fmt.Errorf("removing bucket %q: %w", bucket, err)
		}
	}
	return nil
}

// connect opens a client to the MinIO endpoint described by env.
func (m MinIO) connect(ctx context.Context, env Config) (s3Client, error) {
	params, err := minioParamsFrom(env)
	if err != nil {
		return nil, err
	}
	client, err := m.opener()(ctx, params)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// ensureBucket creates bucket if it does not already exist.
func ensureBucket(ctx context.Context, client s3Client, bucket string) error {
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("checking bucket %q: %w", bucket, err)
	}
	if exists {
		return nil
	}
	if err := client.MakeBucket(ctx, bucket); err != nil {
		return fmt.Errorf("creating bucket %q: %w", bucket, err)
	}
	return nil
}

// downloadObject streams one object to <base>/<bucket>/<key> and returns its
// metadata for the manifest.
func downloadObject(ctx context.Context, client s3Client, bucket, key, base string) (objectInfo, error) {
	r, info, err := client.GetObject(ctx, bucket, key)
	if err != nil {
		return objectInfo{}, fmt.Errorf("reading %s/%s: %w", bucket, key, err)
	}
	defer r.Close()

	path, err := safeJoin(base, bucket, filepath.FromSlash(key))
	if err != nil {
		return objectInfo{}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return objectInfo{}, fmt.Errorf("creating dir for %s: %w", path, err)
	}
	f, err := os.Create(path)
	if err != nil {
		return objectInfo{}, fmt.Errorf("creating %s: %w", path, err)
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return objectInfo{}, fmt.Errorf("writing %s: %w", path, err)
	}
	return info, nil
}

// uploadObject streams the file at path back into bucket under obj.Key.
func uploadObject(ctx context.Context, client s3Client, obj minioObject, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return fmt.Errorf("statting %s: %w", path, err)
	}
	if err := client.PutObject(ctx, obj.Bucket, obj.Key, f, fi.Size(), obj.ContentType); err != nil {
		return fmt.Errorf("uploading %s/%s: %w", obj.Bucket, obj.Key, err)
	}
	return nil
}

// minioManifest records the buckets and objects captured in a snapshot.
type minioManifest struct {
	Buckets []string      `json:"buckets"`
	Objects []minioObject `json:"objects"`
}

// minioObject is one object's location and metadata within a snapshot.
type minioObject struct {
	Bucket      string `json:"bucket"`
	Key         string `json:"key"`
	Size        int64  `json:"size"`
	ContentType string `json:"contentType,omitempty"`
}

// readMinioManifest loads the manifest under base. A missing manifest yields a
// nil manifest (and no error), signalling "no minio artifact in this snapshot".
func readMinioManifest(base string) (*minioManifest, error) {
	path := filepath.Join(base, minioManifestFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading manifest %s: %w", path, err)
	}
	var m minioManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest %s: %w", path, err)
	}
	return &m, nil
}

// writeMinioManifest serialises the manifest into base.
func writeMinioManifest(base string, m *minioManifest) error {
	path := filepath.Join(base, minioManifestFile)
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding manifest: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing manifest %s: %w", path, err)
	}
	return nil
}

// safeJoin joins parts onto base and verifies the result stays within base, so a
// crafted bucket or object key cannot escape the snapshot folder via "..".
func safeJoin(base string, parts ...string) (string, error) {
	joined := filepath.Join(append([]string{base}, parts...)...)
	rel, err := filepath.Rel(base, joined)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes %q", filepath.Join(parts...), base)
	}
	return joined, nil
}
