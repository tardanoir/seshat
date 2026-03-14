package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type postgresDriver struct {
	conn *pgx.Conn
}

func newPostgres(ctx context.Context, connString string) (*postgresDriver, error) {
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return nil, err
	}
	return &postgresDriver{conn: conn}, nil
}

func (p *postgresDriver) Close(ctx context.Context) error {
	if p.conn != nil {
		return p.conn.Close(ctx)
	}
	return nil
}

func (p *postgresDriver) Execute(ctx context.Context, sql string) (*QueryResult, error) {
	start := time.Now()
	result, err := execMultiple(ctx, p, sql)
	if err != nil {
		return nil, err
	}
	result.Duration = time.Since(start)
	return result, nil
}

func (p *postgresDriver) execSingle(ctx context.Context, sql string) (*QueryResult, error) {
	rows, err := p.conn.Query(ctx, sql)
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
		RowsAffected: rows.CommandTag().RowsAffected(),
		IsSelect:     len(columns) > 0,
	}, nil
}

func (p *postgresDriver) ListTables(ctx context.Context) ([]TableInfo, error) {
	rows, err := p.conn.Query(ctx, `
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

func (p *postgresDriver) ListColumns(ctx context.Context, schema, tableName string) ([]ColumnInfo, error) {
	rows, err := p.conn.Query(ctx, `
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
