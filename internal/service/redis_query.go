package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

// Query implements Querier: run one Redis command (redis-cli style) and render
// its reply into the tabular QueryResult the console UI expects. The command is
// tokenised respecting single/double quotes so arguments with spaces work. A
// command that misses a key (redis.Nil) is a normal outcome rendered as an
// empty result, not an error.
func (r Redis) Query(ctx context.Context, spec Spec, command string) (*QueryResult, error) {
	tokens, err := tokenizeRedisCommand(command)
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("command must not be empty")
	}

	db, err := redisDB(spec.Env)
	if err != nil {
		return nil, err
	}
	client, err := r.connect(ctx, spec.Env, db)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	args := make([]any, len(tokens))
	for i, t := range tokens {
		args[i] = t
	}
	reply, err := client.Do(ctx, args...)
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("executing command: %w", err)
	}

	rows, truncated := redisRows(reply)
	return &QueryResult{
		Columns:   []string{"result"},
		Rows:      rows,
		RowCount:  len(rows),
		Command:   strings.ToUpper(tokens[0]),
		Truncated: truncated,
	}, nil
}

// Schema implements Querier. Redis has no table schema, so introspection
// returns an empty namespace and the console editor degrades to keyword-only
// completion — the same graceful fallback the editor uses when introspection
// fails for postgres.
func (r Redis) Schema(context.Context, Spec) (*SchemaInfo, error) {
	return &SchemaInfo{Tables: []TableInfo{}}, nil
}

// redisRows renders a command reply into console rows: an array reply becomes
// one row per element, anything else a single row. Rows are capped at
// maxQueryRows, reporting truncation like the postgres console.
func redisRows(reply any) ([][]any, bool) {
	if arr, ok := reply.([]any); ok {
		truncated := false
		if len(arr) > maxQueryRows {
			arr = arr[:maxQueryRows]
			truncated = true
		}
		rows := make([][]any, 0, len(arr))
		for _, e := range arr {
			rows = append(rows, []any{renderRedisScalar(e)})
		}
		return rows, truncated
	}
	return [][]any{{renderRedisScalar(reply)}}, false
}

// renderRedisScalar maps one reply element onto something that serialises
// cleanly to JSON. Nested arrays are flattened to a space-joined string, the
// way redis-cli renders them inline.
func renderRedisScalar(v any) any {
	switch x := v.(type) {
	case nil:
		return nil
	case string, int64, float64, bool:
		return v
	case []byte:
		return string(x)
	case error:
		return x.Error()
	case []any:
		parts := make([]string, len(x))
		for i, e := range x {
			parts[i] = fmt.Sprintf("%v", renderRedisScalar(e))
		}
		return strings.Join(parts, " ")
	default:
		return fmt.Sprintf("%v", x)
	}
}

// tokenizeRedisCommand splits a command line into arguments, honouring single
// and double quotes so a quoted argument may contain spaces. An unterminated
// quote is an actionable error.
func tokenizeRedisCommand(line string) ([]string, error) {
	var tokens []string
	var cur strings.Builder
	inToken := false
	quote := rune(0)

	flush := func() {
		if inToken {
			tokens = append(tokens, cur.String())
			cur.Reset()
			inToken = false
		}
	}

	for _, c := range line {
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			} else {
				cur.WriteRune(c)
			}
		case c == '\'' || c == '"':
			inToken = true
			quote = c
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			flush()
		default:
			inToken = true
			cur.WriteRune(c)
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote in command")
	}
	flush()
	return tokens, nil
}
