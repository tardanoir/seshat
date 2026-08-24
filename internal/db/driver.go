package db

import (
	"context"
	"sync"
	"time"
)

type QueryResult struct {
	Columns      []string
	Rows         [][]string
	Duration     time.Duration
	RowsAffected int64
	IsSelect     bool
	Truncated    bool
	TotalRows    int
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
	Dialect() string
	Close(ctx context.Context) error
}

// DB wraps a Driver with a connection name. Drivers hold a single underlying
// connection that is not safe for concurrent use, and Bubble Tea runs every
// tea.Cmd in its own goroutine, so mu serializes all driver access.
type DB struct {
	mu      sync.Mutex
	Driver  Driver
	Name    string
	MaxRows int
}

func (d *DB) Execute(ctx context.Context, sql string) (*QueryResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	result, err := d.Driver.Execute(ctx, sql)
	if err != nil {
		return nil, err
	}
	if d.MaxRows > 0 && result.IsSelect && len(result.Rows) > d.MaxRows {
		result.TotalRows = len(result.Rows)
		result.Rows = result.Rows[:d.MaxRows]
		result.Truncated = true
	}
	return result, nil
}

type maxRowsSetter interface {
	SetMaxRows(int)
}

// this is used only in the export func, so the user get's all the results
func (d *DB) ExecuteUnlimited(ctx context.Context, sql string) (*QueryResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if setter, ok := d.Driver.(maxRowsSetter); ok {
		setter.SetMaxRows(0)
		defer setter.SetMaxRows(d.MaxRows)
	}
	return d.Driver.Execute(ctx, sql)
}

func (d *DB) ListTables(ctx context.Context) ([]TableInfo, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.Driver.ListTables(ctx)
}

func (d *DB) ListColumns(ctx context.Context, schema, tableName string) ([]ColumnInfo, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.Driver.ListColumns(ctx, schema, tableName)
}

func (d *DB) Close(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.Driver.Close(ctx)
}

func (d *DB) Dialect() string {
	return d.Driver.Dialect()
}
