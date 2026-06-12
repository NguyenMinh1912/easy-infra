package service

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// objectInfo is the per-object metadata the backup/restore path records and
// replays: enough to recreate the object faithfully (its size and content type).
type objectInfo struct {
	Size        int64
	ContentType string
}

// objectPage is one folder level of a bucket: the sub-folder prefixes and the
// objects directly at that level, as returned by a delimited listing.
type objectPage struct {
	prefixes []string
	objects  []objectListEntry
}

// objectListEntry is one object's key and metadata within a delimited listing.
type objectListEntry struct {
	key          string
	size         int64
	lastModified time.Time
	contentType  string
}

// s3Client is the subset of the S3 / MinIO API the minio lifecycle relies on.
// Depending on an interface (rather than *minio.Client directly) lets tests
// inject a fake and assert the exact bucket/object traffic without a live
// server — mirroring how postgres depends on pgConn.
type s3Client interface {
	// ListBuckets returns the names of all buckets on the server.
	ListBuckets(ctx context.Context) ([]string, error)
	// BucketExists reports whether the named bucket exists.
	BucketExists(ctx context.Context, bucket string) (bool, error)
	// MakeBucket creates the named bucket.
	MakeBucket(ctx context.Context, bucket string) error
	// RemoveBucket deletes the named (already-emptied) bucket.
	RemoveBucket(ctx context.Context, bucket string) error
	// ListObjects returns the keys of every object in bucket, recursively.
	ListObjects(ctx context.Context, bucket string) ([]string, error)
	// ListObjectsPage lists the immediate children under prefix in bucket,
	// folder-style: the object entries at that level plus the sub-folder
	// prefixes (each ending in "/"). An empty prefix lists the bucket root.
	ListObjectsPage(ctx context.Context, bucket, prefix string) (objectPage, error)
	// GetObject opens the object's contents for reading and returns its metadata.
	// The caller closes the reader.
	GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, objectInfo, error)
	// PutObject writes size bytes from r into bucket under key, tagging it with
	// contentType when non-empty.
	PutObject(ctx context.Context, bucket, key string, r io.Reader, size int64, contentType string) error
	// RemoveObject deletes a single object.
	RemoveObject(ctx context.Context, bucket, key string) error
}

// s3Opener establishes a client to the MinIO server described by p. It is a
// seam: the zero-value MinIO uses realS3Opener, while tests supply a fake.
type s3Opener func(ctx context.Context, p minioParams) (s3Client, error)

// realS3Opener dials a real MinIO/S3 server with the minio-go client. The client
// is lazy — it does not connect until the first request — so reachability is
// confirmed by Health's ListBuckets rather than here.
func realS3Opener(_ context.Context, p minioParams) (s3Client, error) {
	cl, err := minio.New(p.endpoint(), &minio.Options{
		Creds:  credentials.NewStaticV4(p.user, p.password, ""),
		Secure: p.secure,
	})
	if err != nil {
		return nil, fmt.Errorf("connecting to minio: %w", err)
	}
	return &realS3Client{cl: cl}, nil
}

// realS3Client adapts *minio.Client to s3Client, translating the channel-based
// listing and option structs into the small surface the lifecycle needs.
type realS3Client struct{ cl *minio.Client }

func (c *realS3Client) ListBuckets(ctx context.Context) ([]string, error) {
	buckets, err := c.cl.ListBuckets(ctx)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(buckets))
	for _, b := range buckets {
		names = append(names, b.Name)
	}
	return names, nil
}

func (c *realS3Client) BucketExists(ctx context.Context, bucket string) (bool, error) {
	return c.cl.BucketExists(ctx, bucket)
}

func (c *realS3Client) MakeBucket(ctx context.Context, bucket string) error {
	return c.cl.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
}

func (c *realS3Client) RemoveBucket(ctx context.Context, bucket string) error {
	return c.cl.RemoveBucket(ctx, bucket)
}

func (c *realS3Client) ListObjects(ctx context.Context, bucket string) ([]string, error) {
	var keys []string
	for obj := range c.cl.ListObjects(ctx, bucket, minio.ListObjectsOptions{Recursive: true}) {
		if obj.Err != nil {
			return nil, obj.Err
		}
		keys = append(keys, obj.Key)
	}
	return keys, nil
}

func (c *realS3Client) ListObjectsPage(ctx context.Context, bucket, prefix string) (objectPage, error) {
	var page objectPage
	// A non-recursive listing applies the "/" delimiter: sub-folders come back
	// as zero-size keys ending in "/", objects as keys at this level.
	for obj := range c.cl.ListObjects(ctx, bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: false}) {
		if obj.Err != nil {
			return objectPage{}, obj.Err
		}
		if strings.HasSuffix(obj.Key, "/") {
			page.prefixes = append(page.prefixes, obj.Key)
			continue
		}
		page.objects = append(page.objects, objectListEntry{
			key:          obj.Key,
			size:         obj.Size,
			lastModified: obj.LastModified,
			contentType:  obj.ContentType,
		})
	}
	return page, nil
}

func (c *realS3Client) GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, objectInfo, error) {
	obj, err := c.cl.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, objectInfo{}, err
	}
	// Stat dials the server, so it surfaces a missing object here (rather than at
	// first read) and gives the authoritative size/content-type for the manifest.
	st, err := obj.Stat()
	if err != nil {
		_ = obj.Close()
		return nil, objectInfo{}, err
	}
	return obj, objectInfo{Size: st.Size, ContentType: st.ContentType}, nil
}

func (c *realS3Client) PutObject(ctx context.Context, bucket, key string, r io.Reader, size int64, contentType string) error {
	_, err := c.cl.PutObject(ctx, bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (c *realS3Client) RemoveObject(ctx context.Context, bucket, key string) error {
	return c.cl.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
}

// minioParams is a profile's MinIO connection settings, normalised out of the
// discrete host/port/... fields so the rest of the lifecycle treats them
// uniformly.
type minioParams struct {
	host        string
	port        int
	consolePort int
	user        string
	password    string
	secure      bool
}

// endpoint is the host:port the S3 API is reached on (no scheme; secure selects
// http vs https in the client options).
func (p minioParams) endpoint() string {
	return fmt.Sprintf("%s:%d", p.host, p.port)
}

// minioParamsFrom extracts the connection settings from a profile's env.
func minioParamsFrom(env Config) (minioParams, error) {
	host, err := requireString(env, "host")
	if err != nil {
		return minioParams{}, err
	}
	port, err := optionalPort(env, "port", 9000)
	if err != nil {
		return minioParams{}, err
	}
	consolePort, err := optionalPort(env, "consolePort", 9001)
	if err != nil {
		return minioParams{}, err
	}
	user, err := requireString(env, "user")
	if err != nil {
		return minioParams{}, err
	}
	password, err := requireString(env, "password")
	if err != nil {
		return minioParams{}, err
	}
	secure, err := optionalBool(env, "secure", false)
	if err != nil {
		return minioParams{}, err
	}
	return minioParams{
		host:        host,
		port:        port,
		consolePort: consolePort,
		user:        user,
		password:    password,
		secure:      secure,
	}, nil
}
