package service

import (
	"context"
	"io"
)

// Browser is an optional capability a Service implements when its store can be
// explored as buckets of folder-organised objects — the backend of the UI's
// object browser. Callers type-assert for it and degrade gracefully when a
// service does not provide it, mirroring how Querier models console support.
type Browser interface {
	// Buckets lists the top-level containers (buckets) on the service.
	Buckets(ctx context.Context, spec Spec) ([]string, error)

	// Objects lists the immediate children under prefix within bucket: the
	// sub-folders (common prefixes) and the objects directly at that level. An
	// empty prefix lists the bucket's root.
	Objects(ctx context.Context, spec Spec, bucket, prefix string) (*ObjectListing, error)

	// Object opens one object for download: a reader the caller must close
	// along with the object's size and content type.
	Object(ctx context.Context, spec Spec, bucket, key string) (io.ReadCloser, ObjectContent, error)

	// Put writes an object into bucket under key, streaming size bytes from r.
	// contentType tags the object when non-empty; size may be -1 when unknown,
	// in which case the implementation streams the body without it.
	Put(ctx context.Context, spec Spec, bucket, key string, r io.Reader, size int64, contentType string) error
}

// ObjectContent is the metadata of an object opened for download.
type ObjectContent struct {
	// Size is the object's length in bytes, or 0 when unknown.
	Size int64
	// ContentType is the object's MIME type, empty when unknown.
	ContentType string
}

// ObjectListing is one folder level within a bucket, shaped for JSON.
type ObjectListing struct {
	// Prefixes are the immediate sub-folders, each a full key ending in "/".
	Prefixes []string `json:"prefixes"`
	// Objects are the objects directly under the listed prefix.
	Objects []ObjectEntry `json:"objects"`
}

// ObjectEntry is one object's key and metadata within a listing.
type ObjectEntry struct {
	Key  string `json:"key"`
	Size int64  `json:"size"`
	// LastModified is the object's modification time, RFC3339, when known.
	LastModified string `json:"lastModified,omitempty"`
	ContentType  string `json:"contentType,omitempty"`
}
