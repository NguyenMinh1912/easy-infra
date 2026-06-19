package service

import "context"

// Querier is an optional capability a Service implements when it can execute
// ad-hoc statements and return tabular results — the backend of the UI's
// console. Callers type-assert for it and degrade gracefully when a service
// does not provide it, mirroring how Provisioner models fork support.
type Querier interface {
	// Query runs one statement against the environment described by spec and
	// returns its result: rows for statements that produce them, otherwise the
	// command tag and affected-row count.
	Query(ctx context.Context, spec Spec, sql string) (*QueryResult, error)

	// Schema describes the queryable namespace (tables and their columns),
	// feeding the console editor's autocomplete.
	Schema(ctx context.Context, spec Spec) (*SchemaInfo, error)
}

// QueryResult is the outcome of one console statement, shaped for JSON.
type QueryResult struct {
	Columns []string `json:"columns"`
	Rows    [][]any  `json:"rows"`
	// RowCount is the number of rows returned (row-producing statements) or
	// affected (DML without a row set).
	RowCount int `json:"rowCount"`
	// Command is the server's command tag, e.g. "SELECT 3" or "UPDATE 1".
	Command string `json:"command"`
	// Truncated is true when the row cap was hit and Rows is a prefix of the
	// full result.
	Truncated bool `json:"truncated"`
	// Editable, when non-nil, describes how the result maps back to a single
	// source table so the UI can offer inline cell edits and row deletes. It is
	// nil unless every result column comes from one table, that table has a
	// primary key, and all of the key's columns are present in the result.
	Editable *EditableInfo `json:"editable,omitempty"`
}

// EditableInfo describes a single-table result the console can edit in place.
// Columns is parallel to QueryResult.Columns: each entry is the source table
// column a result column maps to, or "" when the result column is an expression
// or otherwise not directly updatable.
type EditableInfo struct {
	Schema     string   `json:"schema"`
	Table      string   `json:"table"`
	PrimaryKey []string `json:"primaryKey"`
	Columns    []string `json:"columns"`
}

// RowEditor is an optional capability a Service implements when the rows a query
// returned can be edited in place — updating a single cell or deleting a row,
// each addressed by the result's primary key. Postgres implements it; callers
// type-assert for it and hide the affordance when a service does not, mirroring
// how Querier and Provisioner are optional capabilities.
type RowEditor interface {
	// UpdateRow sets one column of the row identified by m.Key to m.Value and
	// returns the resulting command tag (e.g. "UPDATE 1").
	UpdateRow(ctx context.Context, spec Spec, m RowMutation) (string, error)
	// DeleteRow removes the row identified by m.Key, returning its command tag.
	DeleteRow(ctx context.Context, spec Spec, m RowMutation) (string, error)
}

// RowMutation identifies a single row to edit and, for updates, the column and
// value to set. Key maps primary-key column names to the row's values; Column
// and Value are used only by UpdateRow, with a nil Value setting the column to
// NULL. Values travel as the text the user sees so the database coerces them to
// each column's actual type.
type RowMutation struct {
	Schema string
	Table  string
	Key    map[string]string
	Column string
	Value  *string
}

// SchemaInfo lists the tables a console user can query.
type SchemaInfo struct {
	Tables []TableInfo `json:"tables"`
	// CurrentSchema is the schema the profile's connection resolves unqualified
	// names against (its search_path), so the editor can suggest that schema's
	// tables without a prefix — matching where statements actually execute.
	CurrentSchema string `json:"currentSchema"`
}

// TableInfo names one table and its columns within a schema (e.g. "public").
type TableInfo struct {
	Schema  string   `json:"schema"`
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
}
