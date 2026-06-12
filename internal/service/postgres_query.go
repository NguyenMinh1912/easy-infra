package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// maxQueryRows caps how many rows a console query returns, keeping the JSON
// response bounded regardless of what the user selects.
const maxQueryRows = 500

// Query implements Querier: run one statement and collect its result. Row sets
// are read up to maxQueryRows; statements without a row set (INSERT/UPDATE/DDL)
// report their command tag and affected-row count.
func (p Postgres) Query(ctx context.Context, spec Spec, sql string) (*QueryResult, error) {
	conn, err := p.connect(ctx, spec.Env, "")
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)

	rows, err := conn.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("executing query: %w", err)
	}
	defer rows.Close()

	columns := make([]string, 0, len(rows.FieldDescriptions()))
	for _, fd := range rows.FieldDescriptions() {
		columns = append(columns, fd.Name)
	}

	out := make([][]any, 0)
	truncated := false
	for rows.Next() {
		if len(out) == maxQueryRows {
			truncated = true
			break
		}
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("reading row: %w", err)
		}
		rendered := make([]any, len(values))
		for i, v := range values {
			rendered[i] = renderValue(v)
		}
		out = append(out, rendered)
	}
	// Close before consulting Err/CommandTag so both reflect the finished (or
	// abandoned, when truncated) result. pgx reports most statement errors here
	// rather than from Query itself.
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("executing query: %w", err)
	}

	tag := rows.CommandTag()
	rowCount := len(out)
	if len(columns) == 0 {
		// No row set: the statement reports what it affected instead.
		rowCount = int(tag.RowsAffected())
	}
	return &QueryResult{
		Columns:   columns,
		Rows:      out,
		RowCount:  rowCount,
		Command:   tag.String(),
		Truncated: truncated,
	}, nil
}

// Schema implements Querier: one pass over information_schema.columns, grouped
// into tables. System schemas are excluded — the console suggests only the
// user's own objects.
func (p Postgres) Schema(ctx context.Context, spec Spec) (*SchemaInfo, error) {
	conn, err := p.connect(ctx, spec.Env, "")
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)

	// The schema the connection resolves unqualified names against follows the
	// profile's configured search_path, so the editor can default to it.
	current, err := currentSchema(ctx, conn)
	if err != nil {
		return nil, err
	}

	rows, err := conn.Query(ctx, `
		SELECT table_schema, table_name, column_name
		FROM information_schema.columns
		WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
		ORDER BY table_schema, table_name, ordinal_position`)
	if err != nil {
		return nil, fmt.Errorf("introspecting schema: %w", err)
	}
	defer rows.Close()

	info := &SchemaInfo{Tables: []TableInfo{}, CurrentSchema: current}
	for rows.Next() {
		var schema, table, column string
		if err := rows.Scan(&schema, &table, &column); err != nil {
			return nil, fmt.Errorf("reading schema row: %w", err)
		}
		n := len(info.Tables)
		if n == 0 || info.Tables[n-1].Schema != schema || info.Tables[n-1].Name != table {
			info.Tables = append(info.Tables, TableInfo{Schema: schema, Name: table})
			n++
		}
		info.Tables[n-1].Columns = append(info.Tables[n-1].Columns, column)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("introspecting schema: %w", err)
	}
	return info, nil
}

// renderValue maps a pgx-decoded value onto something that serializes cleanly
// to JSON: primitives pass through, well-known types get a readable string
// form, and anything else falls back to its own JSON marshalling or fmt.
func renderValue(v any) any {
	switch x := v.(type) {
	case nil, bool, string, int16, int32, int64, float32, float64:
		return v
	case time.Time:
		return x.Format("2006-01-02 15:04:05.999999999Z07:00")
	case [16]byte: // uuid
		return fmt.Sprintf("%x-%x-%x-%x-%x", x[0:4], x[4:6], x[6:8], x[8:10], x[10:16])
	case []byte:
		return fmt.Sprintf("\\x%x", x)
	case map[string]any: // json/jsonb objects are already JSON-shaped
		return x
	case []any: // json arrays and postgres arrays; elements may need rendering
		rendered := make([]any, len(x))
		for i, e := range x {
			rendered[i] = renderValue(e)
		}
		return rendered
	default:
		// pgtype values (numeric, intervals, …) know how to express themselves
		// as JSON; keep that representation rather than dumping struct fields.
		if m, ok := v.(json.Marshaler); ok {
			if b, err := m.MarshalJSON(); err == nil {
				return json.RawMessage(b)
			}
		}
		return fmt.Sprintf("%v", v)
	}
}
