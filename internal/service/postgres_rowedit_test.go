package service

import (
	"context"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// execConn records the SQL and args passed to Exec, serving a canned tag — the
// surface RowEditor exercises.
type execConn struct {
	fakeConn
	sql  string
	args []any
	tag  pgconn.CommandTag
}

func (c *execConn) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	c.sql = sql
	c.args = args
	return c.tag, nil
}

func withExecConn(c *execConn) Postgres {
	return Postgres{open: func(context.Context, string) (pgConn, error) { return c, nil }}
}

func strptr(s string) *string { return &s }

func TestUpdateRow(t *testing.T) {
	c := &execConn{tag: pgconn.NewCommandTag("UPDATE 1")}
	cmd, err := withExecConn(c).UpdateRow(context.Background(), Spec{Env: Postgres{}.DefaultEnv()}, RowMutation{
		Schema: "public", Table: "users",
		Key:    map[string]string{"id": "7"},
		Column: "email", Value: strptr("ada@example.com"),
	})
	if err != nil {
		t.Fatalf("UpdateRow: %v", err)
	}
	if cmd != "UPDATE 1" {
		t.Errorf("command = %q, want %q", cmd, "UPDATE 1")
	}
	wantSQL := `UPDATE "public"."users" SET "email" = $1 WHERE "id" = $2`
	if c.sql != wantSQL {
		t.Errorf("sql = %q, want %q", c.sql, wantSQL)
	}
	// Simple-protocol mode first, then the SET value, then the key value — so
	// the database coerces each text value to the column's actual type.
	wantArgs := []any{pgx.QueryExecModeSimpleProtocol, "ada@example.com", "7"}
	if !reflect.DeepEqual(c.args, wantArgs) {
		t.Errorf("args = %#v, want %#v", c.args, wantArgs)
	}
}

func TestUpdateRowNullAndCompositeKey(t *testing.T) {
	c := &execConn{tag: pgconn.NewCommandTag("UPDATE 1")}
	_, err := withExecConn(c).UpdateRow(context.Background(), Spec{Env: Postgres{}.DefaultEnv()}, RowMutation{
		Schema: "app", Table: "memberships",
		// Out-of-order keys exercise the deterministic (sorted) placeholder order.
		Key:    map[string]string{"user_id": "3", "org_id": "9"},
		Column: "role", Value: nil,
	})
	if err != nil {
		t.Fatalf("UpdateRow: %v", err)
	}
	wantSQL := `UPDATE "app"."memberships" SET "role" = $1 WHERE "org_id" = $2 AND "user_id" = $3`
	if c.sql != wantSQL {
		t.Errorf("sql = %q, want %q", c.sql, wantSQL)
	}
	// A nil Value travels as a nil arg so the column is set to NULL.
	wantArgs := []any{pgx.QueryExecModeSimpleProtocol, nil, "9", "3"}
	if !reflect.DeepEqual(c.args, wantArgs) {
		t.Errorf("args = %#v, want %#v", c.args, wantArgs)
	}
}

func TestDeleteRow(t *testing.T) {
	c := &execConn{tag: pgconn.NewCommandTag("DELETE 1")}
	cmd, err := withExecConn(c).DeleteRow(context.Background(), Spec{Env: Postgres{}.DefaultEnv()}, RowMutation{
		Schema: "public", Table: "users",
		Key: map[string]string{"id": "7"},
	})
	if err != nil {
		t.Fatalf("DeleteRow: %v", err)
	}
	if cmd != "DELETE 1" {
		t.Errorf("command = %q, want %q", cmd, "DELETE 1")
	}
	wantSQL := `DELETE FROM "public"."users" WHERE "id" = $1`
	if c.sql != wantSQL {
		t.Errorf("sql = %q, want %q", c.sql, wantSQL)
	}
	wantArgs := []any{pgx.QueryExecModeSimpleProtocol, "7"}
	if !reflect.DeepEqual(c.args, wantArgs) {
		t.Errorf("args = %#v, want %#v", c.args, wantArgs)
	}
}

// catalogConn answers the catalog lookups editableInfo makes: QueryRow resolves
// the table identity, Query lists its columns and primary key.
type catalogConn struct {
	fakeConn
	schema, table string
	cols          [][]any // {attnum int, name string, isPK bool}
}

func (c *catalogConn) QueryRow(context.Context, string, ...any) pgx.Row {
	return identityRow{schema: c.schema, table: c.table}
}

func (c *catalogConn) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return &fakeRows{columns: []string{"attnum", "attname", "is_pk"}, data: c.cols}, nil
}

// identityRow scans the (schema, table) pair tableIdentity reads.
type identityRow struct{ schema, table string }

func (r identityRow) Scan(dest ...any) error {
	if len(dest) == 2 {
		*dest[0].(*string) = r.schema
		*dest[1].(*string) = r.table
	}
	return nil
}

func TestEditableInfo(t *testing.T) {
	users := &catalogConn{
		schema: "public", table: "users",
		cols: [][]any{{1, "id", true}, {2, "email", false}},
	}
	// A column subset (id, email) of a single table with the PK present is
	// editable; the alias maps back to its source column.
	got := editableInfo(context.Background(), users, []colMeta{
		{name: "id", tableOID: 100, attnum: 1},
		{name: "addr", tableOID: 100, attnum: 2},
	})
	want := &EditableInfo{
		Schema: "public", Table: "users",
		PrimaryKey: []string{"id"},
		Columns:    []string{"id", "email"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("editableInfo = %+v, want %+v", got, want)
	}
}

func TestEditableInfoNotEditable(t *testing.T) {
	noPK := &catalogConn{schema: "public", table: "logs", cols: [][]any{{1, "msg", false}}}
	withPK := &catalogConn{schema: "public", table: "users", cols: [][]any{{1, "id", true}, {2, "email", false}}}

	cases := []struct {
		name  string
		conn  pgConn
		metas []colMeta
	}{
		{"all expressions", withPK, []colMeta{{name: "now"}}},
		{"join across tables", withPK, []colMeta{
			{name: "id", tableOID: 100, attnum: 1},
			{name: "name", tableOID: 200, attnum: 1},
		}},
		{"no primary key", noPK, []colMeta{{name: "msg", tableOID: 100, attnum: 1}}},
		{"primary key not selected", withPK, []colMeta{{name: "email", tableOID: 100, attnum: 2}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := editableInfo(context.Background(), tc.conn, tc.metas); got != nil {
				t.Errorf("editableInfo = %+v, want nil", got)
			}
		})
	}
}
