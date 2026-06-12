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
}

// SchemaInfo lists the tables a console user can query.
type SchemaInfo struct {
	Tables []TableInfo `json:"tables"`
}

// TableInfo names one table and its columns within a schema (e.g. "public").
type TableInfo struct {
	Schema  string   `json:"schema"`
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
}
