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
	// Relations are the foreign-key paths from the source table to other tables,
	// in both directions, so the UI can let a row link out to the data it refers
	// to and the rows that refer back to it. Empty when the table has no foreign
	// keys (or introspecting them failed).
	Relations []Relation `json:"relations,omitempty"`
}

// Relation is one foreign-key path between the result's source table and a
// related table. Direction is "references" when the source table's columns
// point at the related table (follow to the parent row it refers to) or
// "referencedBy" when the related table's columns point back at the source
// (follow to the child rows that refer to it).
type Relation struct {
	Constraint string           `json:"constraint"`
	Direction  string           `json:"direction"`
	Schema     string           `json:"schema"`
	Table      string           `json:"table"`
	Columns    []RelationColumn `json:"columns"`
}

// RelationColumn pairs a column on the source table (Local) with the column on
// the related table it joins to (Foreign). To follow the relation, the related
// table is filtered where each Foreign column equals the source row's Local
// value.
type RelationColumn struct {
	Local   string `json:"local"`
	Foreign string `json:"foreign"`
}

// RelationBrowser is an optional capability a Service implements when it can
// fetch the rows on the far side of a Relation. Callers type-assert for it and
// hide the affordance when a service does not, mirroring Querier and RowEditor.
type RelationBrowser interface {
	// RelatedRows returns the rows of q.Table matching q.Filters (ANDed
	// equality predicates), shaped like any other query result so they can be
	// edited and explored further.
	RelatedRows(ctx context.Context, spec Spec, q RelatedQuery) (*QueryResult, error)
}

// RelatedQuery selects the rows reachable through a Relation: the related table
// and the equality predicates that join it to the originating row. A nil Value
// matches NULL. Values travel as text so the database coerces them to each
// column's type, matching how RowMutation values are bound.
type RelatedQuery struct {
	Schema  string
	Table   string
	Filters []RelationFilter
}

// RelationFilter is one column-equals-value predicate of a RelatedQuery.
type RelationFilter struct {
	Column string
	Value  *string
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
