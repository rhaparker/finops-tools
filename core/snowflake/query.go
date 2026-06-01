package snowflake

import (
	"context"
	"database/sql"
	"fmt"
)

// Row holds one result row as column name → value strings.
type Row map[string]string

// QueryResult is the outcome of a SQL query.
type QueryResult struct {
	Columns []string
	Rows    []Row
}

// Query executes SQL and returns all rows as strings.
func Query(ctx context.Context, db *sql.DB, sqlText string) (QueryResult, error) {
	rows, err := db.QueryContext(ctx, sqlText)
	if err != nil {
		return QueryResult{}, fmt.Errorf("snowflake query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return QueryResult{}, err
	}
	out := QueryResult{Columns: cols}

	for rows.Next() {
		dest := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range dest {
			var s sql.NullString
			dest[i] = &s
			ptrs[i] = &s
		}
		if err := rows.Scan(ptrs...); err != nil {
			return QueryResult{}, err
		}
		row := make(Row, len(cols))
		for i, col := range cols {
			ns := dest[i].(*sql.NullString)
			if ns.Valid {
				row[col] = ns.String
			}
		}
		out.Rows = append(out.Rows, row)
	}
	if err := rows.Err(); err != nil {
		return QueryResult{}, err
	}
	return out, nil
}
