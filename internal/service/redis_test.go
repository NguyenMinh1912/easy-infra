package service

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/redis/go-redis/v9"
)

// fakeRedis is an in-memory redisClient that dispatches each command to a
// handler keyed by the command name (args[0], upper-cased), so a test declares
// only the replies it cares about.
type fakeRedis struct {
	handlers map[string]func(args []any) (any, error)
	closed   bool
}

func (f *fakeRedis) Do(_ context.Context, args ...any) (any, error) {
	name := strings.ToUpper(fmt.Sprintf("%v", args[0]))
	h, ok := f.handlers[name]
	if !ok {
		return nil, fmt.Errorf("unexpected command %q", name)
	}
	return h(args)
}

func (f *fakeRedis) Close() error { f.closed = true; return nil }

// redisWith returns a Redis whose opener yields the given fake.
func redisWith(fake *fakeRedis) Redis {
	return Redis{open: func(Config, int) (redisClient, error) { return fake, nil }}
}

func TestRedisHealthPings(t *testing.T) {
	fake := &fakeRedis{handlers: map[string]func([]any) (any, error){
		"PING": func([]any) (any, error) { return "PONG", nil },
	}}
	if err := redisWith(fake).Health(context.Background(), Spec{Env: Config{"host": "localhost"}}); err != nil {
		t.Fatalf("Health: %v", err)
	}
	if !fake.closed {
		t.Error("Health did not close the client")
	}
}

func TestRedisHealthReportsUnreachable(t *testing.T) {
	fake := &fakeRedis{handlers: map[string]func([]any) (any, error){
		"PING": func([]any) (any, error) { return nil, fmt.Errorf("connection refused") },
	}}
	err := redisWith(fake).Health(context.Background(), Spec{Env: Config{"host": "localhost"}})
	if err == nil || !strings.Contains(err.Error(), "redis not ready") {
		t.Fatalf("Health err = %v, want it to wrap 'redis not ready'", err)
	}
}

func TestRedisQueryRendersScalar(t *testing.T) {
	fake := &fakeRedis{handlers: map[string]func([]any) (any, error){
		"GET": func([]any) (any, error) { return "bar", nil },
	}}
	res, err := redisWith(fake).Query(context.Background(), Spec{}, "GET foo")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if want := []string{"result"}; !reflect.DeepEqual(res.Columns, want) {
		t.Errorf("Columns = %v, want %v", res.Columns, want)
	}
	if !reflect.DeepEqual(res.Rows, [][]any{{"bar"}}) {
		t.Errorf("Rows = %v, want [[bar]]", res.Rows)
	}
	if res.Command != "GET" {
		t.Errorf("Command = %q, want GET", res.Command)
	}
}

func TestRedisQueryRendersArray(t *testing.T) {
	fake := &fakeRedis{handlers: map[string]func([]any) (any, error){
		"KEYS": func([]any) (any, error) { return []any{"a", "b"}, nil },
	}}
	res, err := redisWith(fake).Query(context.Background(), Spec{}, "KEYS *")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if !reflect.DeepEqual(res.Rows, [][]any{{"a"}, {"b"}}) {
		t.Errorf("Rows = %v, want one row per element", res.Rows)
	}
}

func TestRedisQueryNilKeyIsNotAnError(t *testing.T) {
	fake := &fakeRedis{handlers: map[string]func([]any) (any, error){
		"GET": func([]any) (any, error) { return nil, redis.Nil },
	}}
	res, err := redisWith(fake).Query(context.Background(), Spec{}, "GET missing")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if !reflect.DeepEqual(res.Rows, [][]any{{nil}}) {
		t.Errorf("Rows = %v, want a single nil row", res.Rows)
	}
}

func TestRedisQueryRejectsEmpty(t *testing.T) {
	if _, err := redisWith(&fakeRedis{}).Query(context.Background(), Spec{}, "   "); err == nil {
		t.Error("expected empty command to error")
	}
}

func TestTokenizeRedisCommand(t *testing.T) {
	got, err := tokenizeRedisCommand(`SET key "a b c"`)
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	if want := []string{"SET", "key", "a b c"}; !reflect.DeepEqual(got, want) {
		t.Errorf("tokens = %v, want %v", got, want)
	}
	if _, err := tokenizeRedisCommand(`GET "unterminated`); err == nil {
		t.Error("expected unterminated quote to error")
	}
}

func TestRedisDatabasesParsesConfig(t *testing.T) {
	fake := &fakeRedis{handlers: map[string]func([]any) (any, error){
		"CONFIG": func([]any) (any, error) { return []any{"databases", "16"}, nil },
	}}
	n, err := redisWith(fake).Databases(context.Background(), Spec{})
	if err != nil {
		t.Fatalf("Databases: %v", err)
	}
	if n != 16 {
		t.Errorf("Databases = %d, want 16", n)
	}
}

func TestRedisDatabasesFallsBack(t *testing.T) {
	fake := &fakeRedis{handlers: map[string]func([]any) (any, error){
		"CONFIG": func([]any) (any, error) { return nil, fmt.Errorf("CONFIG disabled") },
	}}
	n, err := redisWith(fake).Databases(context.Background(), Spec{})
	if err != nil {
		t.Fatalf("Databases: %v", err)
	}
	if n != defaultDatabases {
		t.Errorf("Databases = %d, want fallback %d", n, defaultDatabases)
	}
}

func TestRedisKeysAnnotatesTypeAndTTL(t *testing.T) {
	fake := &fakeRedis{handlers: map[string]func([]any) (any, error){
		"SCAN": func([]any) (any, error) { return []any{"0", []any{"k1", "k2"}}, nil },
		"TYPE": func(args []any) (any, error) {
			if args[1] == "k1" {
				return "string", nil
			}
			return "list", nil
		},
		"TTL": func(args []any) (any, error) {
			if args[1] == "k1" {
				return int64(-1), nil
			}
			return int64(60), nil
		},
	}}
	page, err := redisWith(fake).Keys(context.Background(), Spec{}, 0, "*", 0)
	if err != nil {
		t.Fatalf("Keys: %v", err)
	}
	if page.Cursor != 0 {
		t.Errorf("Cursor = %d, want 0", page.Cursor)
	}
	want := []KeyEntry{{Name: "k1", Type: "string", TTL: -1}, {Name: "k2", Type: "list", TTL: 60}}
	if !reflect.DeepEqual(page.Keys, want) {
		t.Errorf("Keys = %+v, want %+v", page.Keys, want)
	}
}

func TestRedisValueString(t *testing.T) {
	fake := &fakeRedis{handlers: map[string]func([]any) (any, error){
		"TYPE": func([]any) (any, error) { return "string", nil },
		"TTL":  func([]any) (any, error) { return int64(100), nil },
		"GET":  func([]any) (any, error) { return "hello", nil },
	}}
	val, err := redisWith(fake).Value(context.Background(), Spec{}, 0, "greeting")
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	if val.Type != "string" || val.String != "hello" || val.Length != 5 || val.TTL != 100 {
		t.Errorf("Value = %+v, want string hello len 5 ttl 100", val)
	}
}

func TestRedisValueHash(t *testing.T) {
	fake := &fakeRedis{handlers: map[string]func([]any) (any, error){
		"TYPE":    func([]any) (any, error) { return "hash", nil },
		"TTL":     func([]any) (any, error) { return int64(-1), nil },
		"HLEN":    func([]any) (any, error) { return int64(2), nil },
		"HGETALL": func([]any) (any, error) { return []any{"f1", "v1", "f2", "v2"}, nil },
	}}
	val, err := redisWith(fake).Value(context.Background(), Spec{}, 0, "h")
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	want := []HashField{{Field: "f1", Value: "v1"}, {Field: "f2", Value: "v2"}}
	if !reflect.DeepEqual(val.Hash, want) {
		t.Errorf("Hash = %+v, want %+v", val.Hash, want)
	}
	if val.Length != 2 || val.Truncated {
		t.Errorf("Length/Truncated = %d/%v, want 2/false", val.Length, val.Truncated)
	}
}

func TestRedisValueZSet(t *testing.T) {
	fake := &fakeRedis{handlers: map[string]func([]any) (any, error){
		"TYPE":   func([]any) (any, error) { return "zset", nil },
		"TTL":    func([]any) (any, error) { return int64(-1), nil },
		"ZCARD":  func([]any) (any, error) { return int64(2), nil },
		"ZRANGE": func([]any) (any, error) { return []any{"m1", "1.5", "m2", "2"}, nil },
	}}
	val, err := redisWith(fake).Value(context.Background(), Spec{}, 0, "z")
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	want := []ZSetMember{{Member: "m1", Score: 1.5}, {Member: "m2", Score: 2}}
	if !reflect.DeepEqual(val.ZSet, want) {
		t.Errorf("ZSet = %+v, want %+v", val.ZSet, want)
	}
}

func TestRedisValueMissingKey(t *testing.T) {
	fake := &fakeRedis{handlers: map[string]func([]any) (any, error){
		"TYPE": func([]any) (any, error) { return "none", nil },
		"TTL":  func([]any) (any, error) { return int64(-2), nil },
	}}
	val, err := redisWith(fake).Value(context.Background(), Spec{}, 0, "ghost")
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	if val.Type != "none" || val.String != "" || val.List != nil {
		t.Errorf("Value = %+v, want a bare none entry", val)
	}
}
