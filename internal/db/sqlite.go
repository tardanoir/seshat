package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type sqliteDriver struct {
	db      *sql.DB
	maxRows int
}

func newSQLite(path string, maxRows int) (*sqliteDriver, error) {
	d, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := d.Ping(); err != nil {
		d.Close()
		return nil, err
	}
	return &sqliteDriver{db: d, maxRows: maxRows}, nil
}

func (s *sqliteDriver) SetMaxRows(n int) { s.maxRows = n }

func (s *sqliteDriver) Close(_ context.Context) error {
	return s.db.Close()
}

func (s *sqliteDriver) Execute(ctx context.Context, sqlText string) (*QueryResult, error) {
	start := time.Now()
	result, err := execMultiple(ctx, s, sqlText)
	if err != nil {
		return nil, err
	}
	result.Duration = time.Since(start)
	return result, nil
}

func (s *sqliteDriver) execSingle(ctx context.Context, sqlText string) (*QueryResult, error) {
	trimmed := strings.TrimSpace(strings.ToUpper(sqlText))
	isQuery := strings.HasPrefix(trimmed, "SELECT") ||
		strings.HasPrefix(trimmed, "PRAGMA") ||
		strings.HasPrefix(trimmed, "EXPLAIN") ||
		strings.HasPrefix(trimmed, "WITH")

	if !isQuery {
		result, err := s.db.ExecContext(ctx, sqlText)
		if err != nil {
			return nil, err
		}
		affected, _ := result.RowsAffected()
		return &QueryResult{
			RowsAffected: affected,
			IsSelect:     false,
		}, nil
	}

	rows, err := s.db.QueryContext(ctx, sqlText)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	cap := s.maxRows
	if cap <= 0 {
		cap = 256
	}
	resultRows := make([][]string, 0, cap)
	truncated := false
	nCols := len(columns)
	vals := make([]any, nCols)
	ptrs := make([]any, nCols)
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	for rows.Next() {
		if s.maxRows > 0 && len(resultRows) >= s.maxRows {
			truncated = true
			break
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make([]string, nCols)
		for i, v := range vals {
			if v == nil {
				row[i] = "NULL"
			} else {
				row[i] = fmt.Sprintf("%v", v)
			}
		}
		resultRows = append(resultRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &QueryResult{
		Columns:   columns,
		Rows:      resultRows,
		IsSelect:  len(columns) > 0,
		Truncated: truncated,
	}, nil
}

func (s *sqliteDriver) ListTables(ctx context.Context) ([]TableInfo, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT '' AS table_schema, name AS table_name
		FROM sqlite_master
		WHERE type = 'table'
		  AND name NOT LIKE 'sqlite_%'
		ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []TableInfo
	for rows.Next() {
		var t TableInfo
		if err := rows.Scan(&t.Schema, &t.Name); err != nil {
			return nil, err
		}
		tables = append(tables, t)
	}
	return tables, rows.Err()
}

func (s *sqliteDriver) ListColumns(ctx context.Context, _, tableName string) ([]ColumnInfo, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%q)", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []ColumnInfo
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull int
		var dfltValue any
		var pk int
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err != nil {
			return nil, err
		}
		cols = append(cols, ColumnInfo{
			Name:     name,
			DataType: dataType,
			Nullable: notNull == 0,
		})
	}
	return cols, rows.Err()
}
