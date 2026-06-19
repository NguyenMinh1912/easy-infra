package service

import "context"

// KeyBrowser is an optional capability a Service implements when it exposes a
// browsable key/value keyspace — the backend of the UI's Redis key browser.
// Callers type-assert for it and degrade gracefully when a service does not
// provide it, mirroring how Querier models the console and Browser the object
// store.
type KeyBrowser interface {
	// Databases reports how many logical databases the server exposes, so the UI
	// can offer a database selector.
	Databases(ctx context.Context, spec Spec) (int, error)

	// Keys lists a page of keys in logical database db matching pattern (a glob
	// like "*" or "user:*"), continuing from cursor. The returned cursor is zero
	// when the scan is complete.
	Keys(ctx context.Context, spec Spec, db int, pattern string, cursor uint64) (*KeyPage, error)

	// Value reads one key's value in logical database db, shaped by its type.
	Value(ctx context.Context, spec Spec, db int, key string) (*KeyValue, error)
}

// KeyPage is one page of a keyspace scan.
type KeyPage struct {
	Keys []KeyEntry `json:"keys"`
	// Cursor continues the scan; zero means the keyspace has been fully walked.
	Cursor uint64 `json:"cursor"`
}

// KeyEntry names one key and its summary metadata.
type KeyEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
	// TTL is the key's time-to-live in seconds: -1 when the key has no expiry,
	// -2 when it no longer exists.
	TTL int64 `json:"ttl"`
}

// HashField is one field/value pair of a hash value.
type HashField struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

// ZSetMember is one member of a sorted set with its score.
type ZSetMember struct {
	Member string  `json:"member"`
	Score  float64 `json:"score"`
}

// KeyValue is one key's value, shaped by its Redis type. Exactly one of the
// type-specific fields is populated for a known type; an unsupported type (e.g.
// stream) carries only the metadata. Collection values are capped, with
// Truncated set when the full value was longer than the page returned.
type KeyValue struct {
	Key  string `json:"key"`
	Type string `json:"type"`
	// TTL is the time-to-live in seconds: -1 no expiry, -2 the key is missing.
	TTL    int64        `json:"ttl"`
	String string       `json:"string,omitempty"`
	List   []string     `json:"list,omitempty"`
	Set    []string     `json:"set,omitempty"`
	Hash   []HashField  `json:"hash,omitempty"`
	ZSet   []ZSetMember `json:"zset,omitempty"`
	// Length is the full element count of a collection value (or the string's
	// byte length), regardless of how many elements were returned.
	Length    int64 `json:"length"`
	Truncated bool  `json:"truncated"`
}
