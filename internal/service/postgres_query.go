package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
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
	return collectResult(ctx, conn, rows)
}

// collectResult reads a pgx row set into a QueryResult, capping at maxQueryRows
// and resolving how (if at all) the rows map back to a single editable table.
// It is shared by Query and RelatedRows so a result reached by following a
// relation behaves exactly like one typed into the console.
func collectResult(ctx context.Context, conn pgConn, rows pgx.Rows) (*QueryResult, error) {
	// Capture the field descriptions' source-table metadata now: it drives
	// editability detection below and may be reset once rows is closed.
	fds := rows.FieldDescriptions()
	columns := make([]string, len(fds))
	metas := make([]colMeta, len(fds))
	for i, fd := range fds {
		columns[i] = fd.Name
		metas[i] = colMeta{name: fd.Name, tableOID: fd.TableOID, attnum: fd.TableAttributeNumber}
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
	result := &QueryResult{
		Columns:   columns,
		Rows:      out,
		RowCount:  rowCount,
		Command:   tag.String(),
		Truncated: truncated,
	}
	if len(columns) > 0 {
		// Best-effort: a result that isn't a simple single-table selection (or a
		// failed catalog lookup) just stays read-only.
		result.Editable = editableInfo(ctx, conn, metas)
	}
	return result, nil
}

// RelatedRows implements RelationBrowser: select the rows of q.Table that match
// q.Filters, joined by value to the originating row. Identifiers are quoted and
// values bound via the simple protocol so each text value is coerced to its
// column's type, mirroring UpdateRow.
func (p Postgres) RelatedRows(ctx context.Context, spec Spec, q RelatedQuery) (*QueryResult, error) {
	conn, err := p.connect(ctx, spec.Env, "")
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)

	conds := make([]string, len(q.Filters))
	args := []any{pgx.QueryExecModeSimpleProtocol}
	for i, f := range q.Filters {
		if f.Value == nil {
			conds[i] = fmt.Sprintf("%s IS NULL", quoteIdent(f.Column))
			continue
		}
		conds[i] = fmt.Sprintf("%s = $%d", quoteIdent(f.Column), len(args))
		args = append(args, *f.Value)
	}
	sql := fmt.Sprintf("SELECT * FROM %s.%s", quoteIdent(q.Schema), quoteIdent(q.Table))
	if len(conds) > 0 {
		sql += " WHERE " + strings.Join(conds, " AND ")
	}

	rows, err := conn.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("executing query: %w", err)
	}
	defer rows.Close()
	return collectResult(ctx, conn, rows)
}

// colMeta is the source-table provenance of one result column, taken from its
// pgx field description: the table it came from and its attribute number there,
// both zero for expressions and other non-column results.
type colMeta struct {
	name     string
	tableOID uint32
	attnum   uint16
}

// editableInfo reports how a result maps back to a single editable table, or
// nil when it cannot be edited safely. A result qualifies only when every
// column that traces to a table traces to the same one (so joins and pure
// expressions are read-only), that table has a primary key, and all of the
// key's columns appear in the result so each row is uniquely addressable.
func editableInfo(ctx context.Context, conn pgConn, metas []colMeta) *EditableInfo {
	var oid uint32
	for _, m := range metas {
		if m.tableOID == 0 {
			continue
		}
		if oid == 0 {
			oid = m.tableOID
		} else if oid != m.tableOID {
			return nil
		}
	}
	if oid == 0 {
		return nil
	}

	schema, table, ok := tableIdentity(ctx, conn, oid)
	if !ok {
		return nil
	}
	names, pk, ok := tableColumns(ctx, conn, oid)
	if !ok || len(pk) == 0 {
		return nil
	}

	cols := make([]string, len(metas))
	present := make(map[string]bool, len(metas))
	for i, m := range metas {
		if m.tableOID == oid && m.attnum > 0 {
			if name, ok := names[int(m.attnum)]; ok {
				cols[i] = name
				present[name] = true
			}
		}
	}
	for _, k := range pk {
		if !present[k] {
			return nil
		}
	}
	return &EditableInfo{
		Schema:     schema,
		Table:      table,
		PrimaryKey: pk,
		Columns:    cols,
		Relations:  foreignKeys(ctx, conn, oid),
	}
}

// TableRelations implements SchemaGrapher: resolve the named table to its OID
// and return its foreign-key relations in both directions. An unknown table is
// an error; introspection that finds no relations is simply an empty result.
func (p Postgres) TableRelations(ctx context.Context, spec Spec, schema, table string) ([]Relation, error) {
	conn, err := p.connect(ctx, spec.Env, "")
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)

	var oid uint32
	err = conn.QueryRow(ctx, `
		SELECT c.oid
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1 AND c.relname = $2`, schema, table).Scan(&oid)
	if err != nil {
		return nil, fmt.Errorf("resolving table %s.%s: %w", schema, table, err)
	}
	return foreignKeys(ctx, conn, oid), nil
}

// foreignKeys returns the foreign-key relations touching the table identified
// by oid, in both directions: constraints it declares (it references another
// table) and constraints that target it (another table references it). Each
// relation's column pairs are oriented from the table's own column (Local) to
// the related table's column (Foreign). Best-effort: a failed lookup yields no
// relations and the result is simply not linkable.
func foreignKeys(ctx context.Context, conn pgConn, oid uint32) []Relation {
	rows, err := conn.Query(ctx, `
		SELECT con.conname,
		       con.conrelid = $1 AS outgoing,
		       CASE WHEN con.conrelid = $1 THEN tn.nspname ELSE sn.nspname END AS other_schema,
		       CASE WHEN con.conrelid = $1 THEN t.relname  ELSE s.relname  END AS other_table,
		       CASE WHEN con.conrelid = $1 THEN sa.attname ELSE ta.attname END AS local_column,
		       CASE WHEN con.conrelid = $1 THEN ta.attname ELSE sa.attname END AS foreign_column
		FROM pg_constraint con
		JOIN pg_class s ON s.oid = con.conrelid
		JOIN pg_namespace sn ON sn.oid = s.relnamespace
		JOIN pg_class t ON t.oid = con.confrelid
		JOIN pg_namespace tn ON tn.oid = t.relnamespace
		JOIN LATERAL unnest(con.conkey, con.confkey) WITH ORDINALITY AS k(src, tgt, ord) ON true
		JOIN pg_attribute sa ON sa.attrelid = con.conrelid AND sa.attnum = k.src
		JOIN pg_attribute ta ON ta.attrelid = con.confrelid AND ta.attnum = k.tgt
		WHERE con.contype = 'f' AND (con.conrelid = $1 OR con.confrelid = $1)
		ORDER BY con.conname, k.ord`, oid)
	if err != nil {
		return nil
	}
	defer rows.Close()

	// A self-referential constraint shows up once per direction, so group by the
	// constraint name and its direction rather than the name alone.
	type relKey struct {
		name     string
		outgoing bool
	}
	byKey := make(map[relKey]*Relation)
	order := make([]relKey, 0)
	for rows.Next() {
		var name, otherSchema, otherTable, local, foreign string
		var outgoing bool
		if err := rows.Scan(&name, &outgoing, &otherSchema, &otherTable, &local, &foreign); err != nil {
			return nil
		}
		key := relKey{name: name, outgoing: outgoing}
		rel, ok := byKey[key]
		if !ok {
			direction := "referencedBy"
			if outgoing {
				direction = "references"
			}
			rel = &Relation{Constraint: name, Direction: direction, Schema: otherSchema, Table: otherTable}
			byKey[key] = rel
			order = append(order, key)
		}
		rel.Columns = append(rel.Columns, RelationColumn{Local: local, Foreign: foreign})
	}
	if rows.Err() != nil || len(order) == 0 {
		return nil
	}

	relations := make([]Relation, 0, len(order))
	for _, key := range order {
		relations = append(relations, *byKey[key])
	}
	return relations
}

// tableIdentity resolves a table OID to its schema and name.
func tableIdentity(ctx context.Context, conn pgConn, oid uint32) (schema, table string, ok bool) {
	err := conn.QueryRow(ctx, `
		SELECT n.nspname, c.relname
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.oid = $1`, oid).Scan(&schema, &table)
	if err != nil {
		return "", "", false
	}
	return schema, table, true
}

// tableColumns returns the table's live columns keyed by attribute number, plus
// the names of its primary-key columns.
func tableColumns(ctx context.Context, conn pgConn, oid uint32) (names map[int]string, pk []string, ok bool) {
	rows, err := conn.Query(ctx, `
		SELECT a.attnum, a.attname, COALESCE(i.indisprimary, false)
		FROM pg_attribute a
		LEFT JOIN pg_index i
			ON i.indrelid = a.attrelid AND i.indisprimary AND a.attnum = ANY(i.indkey)
		WHERE a.attrelid = $1 AND a.attnum > 0 AND NOT a.attisdropped
		ORDER BY a.attnum`, oid)
	if err != nil {
		return nil, nil, false
	}
	defer rows.Close()

	names = make(map[int]string)
	for rows.Next() {
		var attnum int
		var name string
		var isPK bool
		if err := rows.Scan(&attnum, &name, &isPK); err != nil {
			return nil, nil, false
		}
		names[attnum] = name
		if isPK {
			pk = append(pk, name)
		}
	}
	if rows.Err() != nil {
		return nil, nil, false
	}
	return names, pk, true
}

// UpdateRow implements RowEditor: set one column of the row addressed by its
// primary key. Identifiers are quoted and values bound as parameters, so the
// row is matched by value and the database coerces each text value to the
// column's actual type (via the simple protocol's unknown-type literals).
func (p Postgres) UpdateRow(ctx context.Context, spec Spec, m RowMutation) (string, error) {
	conn, err := p.connect(ctx, spec.Env, "")
	if err != nil {
		return "", err
	}
	defer conn.Close(ctx)

	var value any
	if m.Value != nil {
		value = *m.Value
	}
	where, keyArgs := whereClause(m.Key, 2) // $1 is the SET value
	sql := fmt.Sprintf("UPDATE %s.%s SET %s = $1 WHERE %s",
		quoteIdent(m.Schema), quoteIdent(m.Table), quoteIdent(m.Column), where)
	args := append([]any{pgx.QueryExecModeSimpleProtocol, value}, keyArgs...)
	tag, err := conn.Exec(ctx, sql, args...)
	if err != nil {
		return "", fmt.Errorf("updating row: %w", err)
	}
	return tag.String(), nil
}

// DeleteRow implements RowEditor: remove the row addressed by its primary key.
func (p Postgres) DeleteRow(ctx context.Context, spec Spec, m RowMutation) (string, error) {
	conn, err := p.connect(ctx, spec.Env, "")
	if err != nil {
		return "", err
	}
	defer conn.Close(ctx)

	where, keyArgs := whereClause(m.Key, 1)
	sql := fmt.Sprintf("DELETE FROM %s.%s WHERE %s",
		quoteIdent(m.Schema), quoteIdent(m.Table), where)
	args := append([]any{pgx.QueryExecModeSimpleProtocol}, keyArgs...)
	tag, err := conn.Exec(ctx, sql, args...)
	if err != nil {
		return "", fmt.Errorf("deleting row: %w", err)
	}
	return tag.String(), nil
}

// whereClause builds an "AND"-joined equality predicate over the primary-key
// columns, with placeholders numbered from start, and returns the matching
// argument values. Columns are sorted so placeholders and arguments stay
// aligned regardless of map iteration order.
func whereClause(key map[string]string, start int) (string, []any) {
	names := make([]string, 0, len(key))
	for name := range key {
		names = append(names, name)
	}
	sort.Strings(names)

	conds := make([]string, len(names))
	args := make([]any, len(names))
	for i, name := range names {
		conds[i] = fmt.Sprintf("%s = $%d", quoteIdent(name), start+i)
		args[i] = key[name]
	}
	return strings.Join(conds, " AND "), args
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
