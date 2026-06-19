package service

import (
	"context"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

// scanCount is the COUNT hint passed to SCAN: how many keys one page aims to
// return. It is a hint, not a guarantee — the UI pages with the returned cursor.
const scanCount = 100

// maxValueItems caps how many elements of a collection value (list, set, hash,
// zset) Value returns, keeping the response bounded for very large values.
const maxValueItems = 500

// defaultDatabases is the logical-database count assumed when the server will
// not report one (managed Redis often disables CONFIG GET).
const defaultDatabases = 16

// Databases implements KeyBrowser: report the server's logical database count
// via CONFIG GET, falling back to the Redis default when the server declines to
// answer.
func (r Redis) Databases(ctx context.Context, spec Spec) (int, error) {
	client, err := r.connect(ctx, spec.Env, 0)
	if err != nil {
		return 0, err
	}
	defer client.Close()

	reply, err := client.Do(ctx, "CONFIG", "GET", "databases")
	if err != nil {
		return defaultDatabases, nil
	}
	pair, ok := reply.([]any)
	if !ok || len(pair) < 2 {
		return defaultDatabases, nil
	}
	n, err := replyToInt(pair[1])
	if err != nil || n <= 0 {
		return defaultDatabases, nil
	}
	return int(n), nil
}

// Keys implements KeyBrowser: one SCAN page in database db, annotating each key
// with its type and TTL.
func (r Redis) Keys(ctx context.Context, spec Spec, db int, pattern string, cursor uint64) (*KeyPage, error) {
	if pattern == "" {
		pattern = "*"
	}
	client, err := r.connect(ctx, spec.Env, db)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	reply, err := client.Do(ctx, "SCAN", cursor, "MATCH", pattern, "COUNT", scanCount)
	if err != nil {
		return nil, fmt.Errorf("scanning keys: %w", err)
	}
	parts, ok := reply.([]any)
	if !ok || len(parts) != 2 {
		return nil, fmt.Errorf("unexpected SCAN reply")
	}
	next, err := replyToUint(parts[0])
	if err != nil {
		return nil, fmt.Errorf("reading scan cursor: %w", err)
	}
	names, err := replyToStringSlice(parts[1])
	if err != nil {
		return nil, fmt.Errorf("reading scan keys: %w", err)
	}

	page := &KeyPage{Keys: make([]KeyEntry, 0, len(names)), Cursor: next}
	for _, name := range names {
		typ, err := r.keyType(ctx, client, name)
		if err != nil {
			return nil, err
		}
		ttl, err := r.keyTTL(ctx, client, name)
		if err != nil {
			return nil, err
		}
		page.Keys = append(page.Keys, KeyEntry{Name: name, Type: typ, TTL: ttl})
	}
	return page, nil
}

// Value implements KeyBrowser: read one key's value, shaped by its type.
func (r Redis) Value(ctx context.Context, spec Spec, db int, key string) (*KeyValue, error) {
	client, err := r.connect(ctx, spec.Env, db)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	typ, err := r.keyType(ctx, client, key)
	if err != nil {
		return nil, err
	}
	ttl, err := r.keyTTL(ctx, client, key)
	if err != nil {
		return nil, err
	}
	val := &KeyValue{Key: key, Type: typ, TTL: ttl}
	if typ == "none" {
		return val, nil
	}

	switch typ {
	case "string":
		s, err := r.readString(ctx, client, key)
		if err != nil {
			return nil, err
		}
		val.String = s
		val.Length = int64(len(s))
	case "list":
		if err := r.readList(ctx, client, key, val); err != nil {
			return nil, err
		}
	case "set":
		if err := r.readSet(ctx, client, key, val); err != nil {
			return nil, err
		}
	case "hash":
		if err := r.readHash(ctx, client, key, val); err != nil {
			return nil, err
		}
	case "zset":
		if err := r.readZSet(ctx, client, key, val); err != nil {
			return nil, err
		}
	default:
		// Unsupported types (e.g. stream) carry metadata only; the UI shows the
		// type and TTL without a value body.
	}
	return val, nil
}

// keyType returns a key's type ("string", "list", …) or "none" when absent.
func (r Redis) keyType(ctx context.Context, client redisClient, key string) (string, error) {
	reply, err := client.Do(ctx, "TYPE", key)
	if err != nil {
		return "", fmt.Errorf("reading key type: %w", err)
	}
	return replyToString(reply)
}

// keyTTL returns a key's TTL in seconds (-1 no expiry, -2 missing).
func (r Redis) keyTTL(ctx context.Context, client redisClient, key string) (int64, error) {
	reply, err := client.Do(ctx, "TTL", key)
	if err != nil {
		return 0, fmt.Errorf("reading key ttl: %w", err)
	}
	return replyToInt(reply)
}

func (r Redis) readString(ctx context.Context, client redisClient, key string) (string, error) {
	reply, err := client.Do(ctx, "GET", key)
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("reading string value: %w", err)
	}
	return replyToString(reply)
}

func (r Redis) readList(ctx context.Context, client redisClient, key string, val *KeyValue) error {
	length, err := r.collectionLen(ctx, client, "LLEN", key)
	if err != nil {
		return err
	}
	reply, err := client.Do(ctx, "LRANGE", key, 0, maxValueItems-1)
	if err != nil {
		return fmt.Errorf("reading list value: %w", err)
	}
	items, err := replyToStringSlice(reply)
	if err != nil {
		return err
	}
	val.List = items
	val.Length = length
	val.Truncated = length > int64(len(items))
	return nil
}

func (r Redis) readSet(ctx context.Context, client redisClient, key string, val *KeyValue) error {
	length, err := r.collectionLen(ctx, client, "SCARD", key)
	if err != nil {
		return err
	}
	// SSCAN bounds the reply for very large sets; one page suffices for the
	// viewer and Truncated flags when more remain.
	reply, err := client.Do(ctx, "SSCAN", key, 0, "COUNT", maxValueItems)
	if err != nil {
		return fmt.Errorf("reading set value: %w", err)
	}
	parts, ok := reply.([]any)
	if !ok || len(parts) != 2 {
		return fmt.Errorf("unexpected SSCAN reply")
	}
	members, err := replyToStringSlice(parts[1])
	if err != nil {
		return err
	}
	val.Set = members
	val.Length = length
	val.Truncated = length > int64(len(members))
	return nil
}

func (r Redis) readHash(ctx context.Context, client redisClient, key string, val *KeyValue) error {
	length, err := r.collectionLen(ctx, client, "HLEN", key)
	if err != nil {
		return err
	}
	reply, err := client.Do(ctx, "HGETALL", key)
	if err != nil {
		return fmt.Errorf("reading hash value: %w", err)
	}
	pairs, err := replyToStringSlice(reply)
	if err != nil {
		return err
	}
	for i := 0; i+1 < len(pairs) && len(val.Hash) < maxValueItems; i += 2 {
		val.Hash = append(val.Hash, HashField{Field: pairs[i], Value: pairs[i+1]})
	}
	val.Length = length
	val.Truncated = length > int64(len(val.Hash))
	return nil
}

func (r Redis) readZSet(ctx context.Context, client redisClient, key string, val *KeyValue) error {
	length, err := r.collectionLen(ctx, client, "ZCARD", key)
	if err != nil {
		return err
	}
	reply, err := client.Do(ctx, "ZRANGE", key, 0, maxValueItems-1, "WITHSCORES")
	if err != nil {
		return fmt.Errorf("reading zset value: %w", err)
	}
	pairs, err := replyToStringSlice(reply)
	if err != nil {
		return err
	}
	for i := 0; i+1 < len(pairs); i += 2 {
		score, err := strconv.ParseFloat(pairs[i+1], 64)
		if err != nil {
			return fmt.Errorf("reading zset score: %w", err)
		}
		val.ZSet = append(val.ZSet, ZSetMember{Member: pairs[i], Score: score})
	}
	val.Length = length
	val.Truncated = length > int64(len(val.ZSet))
	return nil
}

// collectionLen runs a length command (LLEN/SCARD/HLEN/ZCARD) for a key.
func (r Redis) collectionLen(ctx context.Context, client redisClient, command, key string) (int64, error) {
	reply, err := client.Do(ctx, command, key)
	if err != nil {
		return 0, fmt.Errorf("reading value length: %w", err)
	}
	return replyToInt(reply)
}

// replyToString coerces a bulk-string reply to a string.
func replyToString(reply any) (string, error) {
	switch v := reply.(type) {
	case nil:
		return "", nil
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	default:
		return "", fmt.Errorf("expected a string reply, got %T", reply)
	}
}

// replyToInt coerces an integer reply (or a numeric string) to int64.
func replyToInt(reply any) (int64, error) {
	switch v := reply.(type) {
	case int64:
		return v, nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	case []byte:
		return strconv.ParseInt(string(v), 10, 64)
	default:
		return 0, fmt.Errorf("expected an integer reply, got %T", reply)
	}
}

// replyToUint coerces a SCAN cursor reply to uint64.
func replyToUint(reply any) (uint64, error) {
	switch v := reply.(type) {
	case int64:
		return uint64(v), nil
	case string:
		return strconv.ParseUint(v, 10, 64)
	case []byte:
		return strconv.ParseUint(string(v), 10, 64)
	default:
		return 0, fmt.Errorf("expected a cursor reply, got %T", reply)
	}
}

// replyToStringSlice coerces an array reply to a slice of strings.
func replyToStringSlice(reply any) ([]string, error) {
	if reply == nil {
		return []string{}, nil
	}
	arr, ok := reply.([]any)
	if !ok {
		return nil, fmt.Errorf("expected an array reply, got %T", reply)
	}
	out := make([]string, 0, len(arr))
	for _, e := range arr {
		s, err := replyToString(e)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}
