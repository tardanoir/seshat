package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type QueryResult struct {
	Columns      []string
	Rows         [][]string
	Duration     time.Duration
	RowsAffected int64
	IsSelect     bool
}

type DB struct {
	conn *pgx.Conn
	Name string
}

func Connect(ctx context.Context, connString, name string) (*DB, error) {
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("connecting to %s: %w", name, err)
	}
	return &DB{conn: conn, Name: name}, nil
}

func (d *DB) Close(ctx context.Context) error {
	if d.conn != nil {
		return d.conn.Close(ctx)
	}
	return nil
}

func (d *DB) Execute(ctx context.Context, sql string) (*QueryResult, error) {
	stmts := splitStatements(sql)
	if len(stmts) == 0 {
		return &QueryResult{IsSelect: false}, nil
	}

	// Single statement — execute directly
	if len(stmts) == 1 {
		return d.execSingle(ctx, stmts[0])
	}

	// Multiple statements — execute sequentially, return last result with columns
	// (or the very last result if none have columns)
	start := time.Now()
	var lastResult *QueryResult
	for _, stmt := range stmts {
		r, err := d.execSingle(ctx, stmt)
		if err != nil {
			return nil, err
		}
		if r.IsSelect || lastResult == nil {
			lastResult = r
		}
	}
	lastResult.Duration = time.Since(start)
	return lastResult, nil
}

func (d *DB) execSingle(ctx context.Context, sql string) (*QueryResult, error) {
	start := time.Now()
	rows, err := d.conn.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	descs := rows.FieldDescriptions()
	columns := make([]string, len(descs))
	for i, fd := range descs {
		columns[i] = fd.Name
	}

	var resultRows [][]string
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, err
		}
		row := make([]string, len(vals))
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
		Columns:      columns,
		Rows:         resultRows,
		Duration:     time.Since(start),
		RowsAffected: rows.CommandTag().RowsAffected(),
		IsSelect:     len(columns) > 0,
	}, nil
}

// splitStatements splits SQL text on semicolons, respecting single-quoted
// strings and double-quoted identifiers. Empty statements are skipped.
func splitStatements(sql string) []string {
	var stmts []string
	var buf []byte
	inSingle := false
	inDouble := false

	for i := 0; i < len(sql); i++ {
		ch := sql[i]
		switch {
		case ch == '\'' && !inDouble:
			inSingle = !inSingle
			buf = append(buf, ch)
		case ch == '"' && !inSingle:
			inDouble = !inDouble
			buf = append(buf, ch)
		case ch == ';' && !inSingle && !inDouble:
			s := strings.TrimSpace(string(buf))
			if s != "" {
				stmts = append(stmts, s)
			}
			buf = buf[:0]
		default:
			buf = append(buf, ch)
		}
	}

	// Trailing statement without semicolon
	s := strings.TrimSpace(string(buf))
	if s != "" {
		stmts = append(stmts, s)
	}
	return stmts
}

type TableInfo struct {
	Schema string
	Name   string
}

type ColumnInfo struct {
	Name     string
	DataType string
	Nullable bool
}

func (d *DB) ListTables(ctx context.Context) ([]TableInfo, error) {
	rows, err := d.conn.Query(ctx, `
		SELECT table_schema, table_name
		FROM information_schema.tables
		WHERE table_schema NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
		  AND table_schema NOT LIKE 'pg_temp_%'
		  AND left(table_schema, 1) != '_'
		  AND table_schema NOT LIKE 'timescaledb_%'
		  AND table_type = 'BASE TABLE'
		ORDER BY table_schema, table_name`)
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

func (d *DB) ListColumns(ctx context.Context, schema, tableName string) ([]ColumnInfo, error) {
	rows, err := d.conn.Query(ctx, `
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position`, schema, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []ColumnInfo
	for rows.Next() {
		var c ColumnInfo
		var nullable string
		if err := rows.Scan(&c.Name, &c.DataType, &nullable); err != nil {
			return nil, err
		}
		c.Nullable = nullable == "YES"
		cols = append(cols, c)
	}
	return cols, rows.Err()
}
