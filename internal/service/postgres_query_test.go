package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// fakeRows is an in-memory pgx.Rows serving canned columns and values, with an
// optional error reported after iteration (the way pgx surfaces most statement
// failures).
type fakeRows struct {
	columns []string
	data    [][]any
	tag     pgconn.CommandTag
	err     error
	idx     int
}

func (r *fakeRows) Close()     {}
func (r *fakeRows) Err() error { return r.err }

func (r *fakeRows) CommandTag() pgconn.CommandTag { return r.tag }

func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription {
	fds := make([]pgconn.FieldDescription, len(r.columns))
	for i, name := range r.columns {
		fds[i] = pgconn.FieldDescription{Name: name}
	}
	return fds
}

func (r *fakeRows) Next() bool {
	if r.err != nil || r.idx >= len(r.data) {
		return false
	}
	r.idx++
	return true
}

func (r *fakeRows) Scan(dest ...any) error {
	row := r.data[r.idx-1]
	for i, d := range dest {
		switch p := d.(type) {
		case *string:
			*p = row[i].(string)
		case *int:
			*p = row[i].(int)
		case *bool:
			*p = row[i].(bool)
		}
	}
	return nil
}

func (r *fakeRows) Values() ([]any, error) { return r.data[r.idx-1], nil }

func (r *fakeRows) RawValues() [][]byte { return nil }

func (r *fakeRows) Conn() *pgx.Conn { return nil }

// queryConn serves a canned result for Query, recording the SQL it ran.
type queryConn struct {
	fakeConn
	rows     *fakeRows
	queryErr error
	queries  []string
}

func (c *queryConn) Query(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
	c.queries = append(c.queries, sql)
	if c.queryErr != nil {
		return nil, c.queryErr
	}
	return c.rows, nil
}

// withQueryConn returns a Postgres whose opener always yields qc.
func withQueryConn(qc *queryConn) Postgres {
	return Postgres{open: func(context.Context, string) (pgConn, error) { return qc, nil }}
}

func TestQueryRowsResult(t *testing.T) {
	qc := &queryConn{rows: &fakeRows{
		columns: []string{"id", "email"},
		data: [][]any{
			{int64(1), "ada@example.com"},
			{int64(2), nil},
		},
		tag: pgconn.NewCommandTag("SELECT 2"),
	}}
	res, err := withQueryConn(qc).Query(context.Background(), Spec{Env: Postgres{}.DefaultEnv()}, "SELECT id, email FROM users")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if !reflect.DeepEqual(res.Columns, []string{"id", "email"}) {
		t.Errorf("Columns = %v", res.Columns)
	}
	want := [][]any{{int64(1), "ada@example.com"}, {int64(2), nil}}
	if !reflect.DeepEqual(res.Rows, want) {
		t.Errorf("Rows = %v, want %v", res.Rows, want)
	}
	if res.RowCount != 2 || res.Command != "SELECT 2" || res.Truncated {
		t.Errorf("result = %+v", res)
	}
	if len(qc.queries) != 1 || qc.queries[0] != "SELECT id, email FROM users" {
		t.Errorf("queries = %v", qc.queries)
	}
}

func TestQueryTruncatesAtCap(t *testing.T) {
	data := make([][]any, maxQueryRows+10)
	for i := range data {
		data[i] = []any{int64(i)}
	}
	qc := &queryConn{rows: &fakeRows{columns: []string{"n"}, data: data}}
	res, err := withQueryConn(qc).Query(context.Background(), Spec{Env: Postgres{}.DefaultEnv()}, "SELECT n FROM big")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(res.Rows) != maxQueryRows || !res.Truncated {
		t.Errorf("rows = %d truncated = %v, want %d rows truncated", len(res.Rows), res.Truncated, maxQueryRows)
	}
	if res.RowCount != maxQueryRows {
		t.Errorf("RowCount = %d, want %d", res.RowCount, maxQueryRows)
	}
}

func TestQueryNoRowSet(t *testing.T) {
	qc := &queryConn{rows: &fakeRows{tag: pgconn.NewCommandTag("UPDATE 7")}}
	res, err := withQueryConn(qc).Query(context.Background(), Spec{Env: Postgres{}.DefaultEnv()}, "UPDATE t SET x = 1")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if res.RowCount != 7 || res.Command != "UPDATE 7" || len(res.Rows) != 0 || len(res.Columns) != 0 {
		t.Errorf("result = %+v, want UPDATE 7 with no rows", res)
	}
}

func TestQueryStatementError(t *testing.T) {
	qc := &queryConn{rows: &fakeRows{err: errors.New(`relation "usrs" does not exist`)}}
	if _, err := withQueryConn(qc).Query(context.Background(), Spec{Env: Postgres{}.DefaultEnv()}, "SELECT * FROM usrs"); err == nil {
		t.Fatal("Query returned nil error for a failing statement")
	}
	qc = &queryConn{queryErr: errors.New("connection refused")}
	if _, err := withQueryConn(qc).Query(context.Background(), Spec{Env: Postgres{}.DefaultEnv()}, "SELECT 1"); err == nil {
		t.Fatal("Query returned nil error for a failing dial")
	}
}

func TestSchemaGroupsTables(t *testing.T) {
	qc := &queryConn{rows: &fakeRows{
		columns: []string{"table_schema", "table_name", "column_name"},
		data: [][]any{
			{"public", "users", "id"},
			{"public", "users", "email"},
			{"public", "orders", "id"},
			{"audit", "events", "at"},
		},
	}}
	info, err := withQueryConn(qc).Schema(context.Background(), Spec{Env: Postgres{}.DefaultEnv()})
	if err != nil {
		t.Fatalf("Schema: %v", err)
	}
	want := []TableInfo{
		{Schema: "public", Name: "users", Columns: []string{"id", "email"}},
		{Schema: "public", Name: "orders", Columns: []string{"id"}},
		{Schema: "audit", Name: "events", Columns: []string{"at"}},
	}
	if !reflect.DeepEqual(info.Tables, want) {
		t.Errorf("Tables = %+v, want %+v", info.Tables, want)
	}
	// Schema reports the connection's current schema so the editor can default
	// completion to it; here current_schema() is unset, so it falls back.
	if info.CurrentSchema != "public" {
		t.Errorf("CurrentSchema = %q, want %q", info.CurrentSchema, "public")
	}
}

func TestRenderValue(t *testing.T) {
	ts := time.Date(2026, 5, 1, 9, 12, 44, 0, time.UTC)
	uuid := [16]byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0}
	cases := []struct {
		in   any
		want any
	}{
		{nil, nil},
		{true, true},
		{int64(42), int64(42)},
		{"hi", "hi"},
		{ts, "2026-05-01 09:12:44Z"},
		{uuid, "12345678-9abc-def0-1234-56789abcdef0"},
		{[]byte{0xde, 0xad}, `\xdead`},
		{map[string]any{"a": 1}, map[string]any{"a": 1}},
		{[]any{ts, "x"}, []any{"2026-05-01 09:12:44Z", "x"}},
		{jsonValue{}, json.RawMessage(`"rendered"`)},
		{plainStruct{}, fmt.Sprintf("%v", plainStruct{})},
	}
	for _, tc := range cases {
		if got := renderValue(tc.in); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("renderValue(%v) = %#v, want %#v", tc.in, got, tc.want)
		}
	}
}

// jsonValue stands in for pgtype values that express themselves as JSON.
type jsonValue struct{}

func (jsonValue) MarshalJSON() ([]byte, error) { return []byte(`"rendered"`), nil }

type plainStruct struct{ X int }
