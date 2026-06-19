package service

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// redisClient is the subset of a Redis connection the Redis console and key
// browser rely on. Depending on a tiny interface (rather than *redis.Client
// directly) lets tests inject a fake that returns canned replies without a live
// server — mirroring how pgConn models the postgres connection. Every command
// goes through Do, so the fake only has to understand the RESP reply shapes the
// callers parse.
type redisClient interface {
	// Do runs one command and returns its decoded reply. A missing key surfaces
	// as redis.Nil, which callers handle rather than treating as an error.
	Do(ctx context.Context, args ...any) (any, error)
	Close() error
}

// redisOpener establishes a client for the environment described by env,
// selecting logical database db. It is a seam: the zero-value Redis dials a real
// server via go-redis (realRedisOpener), while tests supply a fake.
type redisOpener func(env Config, db int) (redisClient, error)

// realRedisOpener builds a go-redis client from the profile env. go-redis dials
// lazily, so option errors surface here while connection errors surface on the
// first command (e.g. PING).
func realRedisOpener(env Config, db int) (redisClient, error) {
	opt, err := redisOptions(env, db)
	if err != nil {
		return nil, err
	}
	return goRedisClient{redis.NewClient(opt)}, nil
}

// goRedisClient adapts *redis.Client to redisClient.
type goRedisClient struct{ c *redis.Client }

func (g goRedisClient) Do(ctx context.Context, args ...any) (any, error) {
	return g.c.Do(ctx, args...).Result()
}

func (g goRedisClient) Close() error { return g.c.Close() }

// redisOptions builds go-redis connection options from a profile's environment
// config, selecting logical database db. A profile may instead supply the whole
// DSN as a single "url" field (e.g. redis://:pass@host:6379/0); when present it
// is parsed and db overrides its database when non-zero.
func redisOptions(env Config, db int) (*redis.Options, error) {
	if raw, ok := env["url"]; ok {
		s, err := urlString(raw)
		if err != nil {
			return nil, err
		}
		opt, err := redis.ParseURL(s)
		if err != nil {
			return nil, fmt.Errorf("%q is not a valid connection URL: %w", "url", err)
		}
		if db != 0 {
			opt.DB = db
		}
		// Pin RESP2 so multi-value replies (HGETALL, CONFIG GET, ZRANGE
		// WITHSCORES) arrive as flat arrays rather than RESP3 maps, matching how
		// the keyspace reader parses them.
		opt.Protocol = 2
		return opt, nil
	}

	host, err := requireString(env, "host")
	if err != nil {
		return nil, err
	}
	port, err := optionalPort(env, "port", 6379)
	if err != nil {
		return nil, err
	}
	password, err := optionalString(env, "password", "")
	if err != nil {
		return nil, err
	}
	return &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: password,
		DB:       db,
		// Pin RESP2 so multi-value replies arrive as flat arrays rather than
		// RESP3 maps, matching how the keyspace reader parses them.
		Protocol: 2,
	}, nil
}
