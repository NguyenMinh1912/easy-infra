package service

import (
	"context"
	"fmt"
	"io"
)

// dump writes a logical SQL backup of the public schema to w by introspecting
// the catalogs. The script is ordered so it reloads cleanly into an empty
// schema: sequences, then tables (without constraints), then data as COPY
// blocks, then constraints and indexes, then sequence values. Foreign keys are
// added only after data loads, so a consistent dump restores without deferring
// referential checks.
//
// v1 fidelity covers tables, columns (type/NOT NULL/default), primary/unique/
// foreign-key/check constraints, indexes, and sequences. Out of scope (and
// documented): extensions, custom types/domains, functions/triggers, views,
// row-level security, and partitioning.
func dump(ctx context.Context, conn pgConn, w io.Writer) error {
	out := &sqlWriter{w: w}
	out.printf("-- easy-infra postgres backup\n")
	out.printf("SET client_encoding = 'UTF8';\n\n")

	// Dump the schema the connection actually resolves unqualified names
	// against, not a hardcoded "public": a profile may point the connection at
	// another schema via a JDBC currentSchema (translated to search_path), and
	// that is where its objects live.
	schema, err := currentSchema(ctx, conn)
	if err != nil {
		return err
	}
	qSchema := quoteIdent(schema)
	if schema != "public" {
		// So a restore into a freshly-created database lands the objects in the
		// right place even when the schema does not exist yet.
		out.printf("CREATE SCHEMA IF NOT EXISTS %s;\n\n", qSchema)
	}

	seqs, err := listSequences(ctx, conn, schema)
	if err != nil {
		return err
	}
	tables, err := listTables(ctx, conn, schema)
	if err != nil {
		return err
	}

	for _, s := range seqs {
		out.printf("%s\n", s.createSQL(qSchema))
	}
	if len(seqs) > 0 {
		out.printf("\n")
	}

	for _, t := range tables {
		ddl, err := tableDDL(ctx, conn, qSchema, t)
		if err != nil {
			return err
		}
		out.printf("%s\n\n", ddl)
	}

	for _, t := range tables {
		if err := dumpTableData(ctx, conn, out, qSchema, t); err != nil {
			return err
		}
	}

	for _, t := range tables {
		cons, err := tableConstraints(ctx, conn, t)
		if err != nil {
			return err
		}
		for _, c := range cons {
			out.printf("ALTER TABLE %s.%s ADD CONSTRAINT %s %s;\n", qSchema, quoteIdent(t.name), quoteIdent(c.name), c.def)
		}
	}

	for _, t := range tables {
		idxs, err := tableIndexes(ctx, conn, t)
		if err != nil {
			return err
		}
		for _, def := range idxs {
			out.printf("%s;\n", def)
		}
	}

	for _, s := range seqs {
		if s.lastValue != nil {
			out.printf("SELECT pg_catalog.setval('%s.%s', %d, true);\n", qSchema, quoteIdent(s.name), *s.lastValue)
		}
	}

	return out.err
}

// currentSchema returns the schema the connection resolves unqualified names
// against. current_schema() honors the search_path set from the profile's
// connection string (e.g. a JDBC currentSchema), so this follows the profile to
// wherever its objects actually live. It falls back to "public" when the
// search_path resolves to no existing schema.
func currentSchema(ctx context.Context, conn pgConn) (string, error) {
	var schema *string
	if err := conn.QueryRow(ctx, "SELECT current_schema()").Scan(&schema); err != nil {
		return "", fmt.Errorf("resolving current schema: %w", err)
	}
	if schema == nil || *schema == "" {
		return "public", nil
	}
	return *schema, nil
}

// sqlWriter accumulates writes and remembers the first error, so dump can emit
// many fragments without checking each call.
type sqlWriter struct {
	w   io.Writer
	err error
}

func (s *sqlWriter) printf(format string, a ...any) {
	if s.err != nil {
		return
	}
	_, s.err = fmt.Fprintf(s.w, format, a...)
}

// table identifies a base table in the public schema.
type table struct {
	oid  uint32
	name string
}

func listTables(ctx context.Context, conn pgConn, schema string) ([]table, error) {
	rows, err := conn.Query(ctx, `
		SELECT c.oid, c.relname
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = $1 AND c.relkind = 'r'
		ORDER BY c.relname`, schema)
	if err != nil {
		return nil, fmt.Errorf("listing tables: %w", err)
	}
	defer rows.Close()
	var tables []table
	for rows.Next() {
		var t table
		if err := rows.Scan(&t.oid, &t.name); err != nil {
			return nil, err
		}
		tables = append(tables, t)
	}
	return tables, rows.Err()
}

// tableDDL builds the CREATE TABLE statement (columns only; constraints and
// indexes are emitted separately).
func tableDDL(ctx context.Context, conn pgConn, qSchema string, t table) (string, error) {
	rows, err := conn.Query(ctx, `
		SELECT a.attname,
		       format_type(a.atttypid, a.atttypmod) AS type,
		       a.attnotnull,
		       pg_get_expr(d.adbin, d.adrelid) AS default
		FROM pg_attribute a
		LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
		WHERE a.attrelid = $1 AND a.attnum > 0 AND NOT a.attisdropped
		ORDER BY a.attnum`, t.oid)
	if err != nil {
		return "", fmt.Errorf("reading columns of %s: %w", t.name, err)
	}
	defer rows.Close()

	var cols []string
	for rows.Next() {
		var name, typ string
		var notNull bool
		var def *string
		if err := rows.Scan(&name, &typ, &notNull, &def); err != nil {
			return "", err
		}
		col := fmt.Sprintf("    %s %s", quoteIdent(name), typ)
		if notNull {
			col += " NOT NULL"
		}
		if def != nil {
			col += " DEFAULT " + *def
		}
		cols = append(cols, col)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	out := fmt.Sprintf("CREATE TABLE %s.%s (\n", qSchema, quoteIdent(t.name))
	for i, c := range cols {
		out += c
		if i < len(cols)-1 {
			out += ","
		}
		out += "\n"
	}
	out += ");"
	return out, nil
}

// constraint is a named constraint and its reconstructed definition.
type constraint struct {
	name string
	def  string
}

func tableConstraints(ctx context.Context, conn pgConn, t table) ([]constraint, error) {
	rows, err := conn.Query(ctx, `
		SELECT conname, pg_get_constraintdef(oid)
		FROM pg_constraint
		WHERE conrelid = $1
		ORDER BY conname`, t.oid)
	if err != nil {
		return nil, fmt.Errorf("reading constraints of %s: %w", t.name, err)
	}
	defer rows.Close()
	var cons []constraint
	for rows.Next() {
		var c constraint
		if err := rows.Scan(&c.name, &c.def); err != nil {
			return nil, err
		}
		cons = append(cons, c)
	}
	return cons, rows.Err()
}

// tableIndexes returns CREATE INDEX statements for indexes not backing a
// constraint (those come from tableConstraints to avoid duplicates).
func tableIndexes(ctx context.Context, conn pgConn, t table) ([]string, error) {
	rows, err := conn.Query(ctx, `
		SELECT pg_get_indexdef(i.indexrelid)
		FROM pg_index i
		WHERE i.indrelid = $1
		  AND NOT EXISTS (SELECT 1 FROM pg_constraint c WHERE c.conindid = i.indexrelid)
		ORDER BY i.indexrelid`, t.oid)
	if err != nil {
		return nil, fmt.Errorf("reading indexes of %s: %w", t.name, err)
	}
	defer rows.Close()
	var defs []string
	for rows.Next() {
		var def string
		if err := rows.Scan(&def); err != nil {
			return nil, err
		}
		defs = append(defs, def)
	}
	return defs, rows.Err()
}

// dumpTableData emits a COPY block for one table: the COPY ... FROM stdin
// header, the table's rows in PostgreSQL text COPY format (as produced by COPY
// ... TO stdout), and the terminating \. marker.
func dumpTableData(ctx context.Context, conn pgConn, out *sqlWriter, qSchema string, t table) error {
	qname := qSchema + "." + quoteIdent(t.name)
	out.printf("COPY %s FROM stdin;\n", qname)
	if out.err != nil {
		return out.err
	}
	if _, err := conn.CopyTo(ctx, out.w, fmt.Sprintf("COPY %s TO stdout", qname)); err != nil {
		return fmt.Errorf("copying data of %s: %w", t.name, err)
	}
	out.printf("\\.\n\n")
	return out.err
}

// sequence describes a sequence in the public schema.
type sequence struct {
	name        string
	dataType    string
	startValue  int64
	minValue    int64
	maxValue    int64
	incrementBy int64
	cycle       bool
	lastValue   *int64
}

func (s sequence) createSQL(qSchema string) string {
	cycle := "NO CYCLE"
	if s.cycle {
		cycle = "CYCLE"
	}
	return fmt.Sprintf(
		"CREATE SEQUENCE IF NOT EXISTS %s.%s AS %s INCREMENT BY %d MINVALUE %d MAXVALUE %d START WITH %d %s;",
		qSchema, quoteIdent(s.name), s.dataType, s.incrementBy, s.minValue, s.maxValue, s.startValue, cycle)
}

func listSequences(ctx context.Context, conn pgConn, schema string) ([]sequence, error) {
	rows, err := conn.Query(ctx, `
		SELECT sequencename, data_type::text, start_value, min_value, max_value, increment_by, cycle, last_value
		FROM pg_sequences
		WHERE schemaname = $1
		ORDER BY sequencename`, schema)
	if err != nil {
		return nil, fmt.Errorf("listing sequences: %w", err)
	}
	defer rows.Close()
	var seqs []sequence
	for rows.Next() {
		var s sequence
		if err := rows.Scan(&s.name, &s.dataType, &s.startValue, &s.minValue, &s.maxValue, &s.incrementBy, &s.cycle, &s.lastValue); err != nil {
			return nil, err
		}
		seqs = append(seqs, s)
	}
	return seqs, rows.Err()
}
