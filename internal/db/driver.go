package db

import (
	"context"
	"time"
)

type QueryResult struct {
	Columns      []string
	Rows         [][]string
	Duration     time.Duration
	RowsAffected int64
	IsSelect     bool
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

// Driver is the interface that database backends must implement.
type Driver interface {
	Execute(ctx context.Context, sql string) (*QueryResult, error)
	ListTables(ctx context.Context) ([]TableInfo, error)
	ListColumns(ctx context.Context, schema, tableName string) ([]ColumnInfo, error)
	Close(ctx context.Context) error
}

// DB wraps a Driver with a connection name.
type DB struct {
	Driver Driver
	Name   string
}

func (d *DB) Execute(ctx context.Context, sql string) (*QueryResult, error) {
	return d.Driver.Execute(ctx, sql)
}

func (d *DB) ListTables(ctx context.Context) ([]TableInfo, error) {
	return d.Driver.ListTables(ctx)
}

func (d *DB) ListColumns(ctx context.Context, schema, tableName string) ([]ColumnInfo, error) {
	return d.Driver.ListColumns(ctx, schema, tableName)
}

func (d *DB) Close(ctx context.Context) error {
	return d.Driver.Close(ctx)
}
