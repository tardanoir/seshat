package db

import (
	"context"
	"fmt"
	"strings"
)

func Connect(ctx context.Context, driverName, connString, name string, maxRows int) (*DB, error) {
	var drv Driver
	var err error

	switch driverName {
	case "postgres", "postgresql", "":
		drv, err = newPostgres(ctx, connString)
	case "sqlite", "sqlite3":
		drv, err = newSQLite(connString)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", driverName)
	}
	if err != nil {
		return nil, fmt.Errorf("connecting to %s: %w", name, err)
	}
	return &DB{Driver: drv, Name: name, MaxRows: maxRows}, nil
}

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

func execMultiple(ctx context.Context, drv singleExecutor, sql string) (*QueryResult, error) {
	stmts := splitStatements(sql)
	if len(stmts) == 0 {
		return &QueryResult{IsSelect: false}, nil
	}

	if len(stmts) == 1 {
		return drv.execSingle(ctx, stmts[0])
	}

	var lastResult *QueryResult
	for _, stmt := range stmts {
		r, err := drv.execSingle(ctx, stmt)
		if err != nil {
			return nil, err
		}
		if r.IsSelect || lastResult == nil {
			lastResult = r
		}
	}
	return lastResult, nil
}

type singleExecutor interface {
	execSingle(ctx context.Context, sql string) (*QueryResult, error)
}
