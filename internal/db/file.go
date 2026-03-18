package db

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// fileDriver wraps an in-memory SQLite database with data imported from a
// CSV or JSON file. All SQL operations are delegated to the underlying
// sqliteDriver so the user gets full SQLite query support.
//
// Since file sources have exactly one table, SELECT statements that omit the
// FROM clause are rewritten to include it automatically. For example:
//
//	SELECT * WHERE price > 100   →   SELECT * FROM sales WHERE price > 100
//	SELECT name, age LIMIT 10    →   SELECT name, age FROM sales LIMIT 10

type fileDriver struct {
	sqlite    *sqliteDriver
	tableName string
}

func newCSV(path string) (*fileDriver, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening csv: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("reading csv: %w", err)
	}
	if len(records) < 1 {
		return nil, fmt.Errorf("csv file is empty")
	}

	headers := records[0]
	rows := records[1:]
	tableName := tableNameFromPath(path)

	return loadIntoSQLite(tableName, headers, rows)
}

func newJSON(path string) (*fileDriver, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("opening json: %w", err)
	}

	var objects []map[string]any
	if err := json.Unmarshal(data, &objects); err != nil {
		return nil, fmt.Errorf("parsing json (expected array of objects): %w", err)
	}
	if len(objects) == 0 {
		return nil, fmt.Errorf("json array is empty")
	}

	seen := make(map[string]bool)
	var headers []string
	for _, obj := range objects {
		for k := range obj {
			if !seen[k] {
				seen[k] = true
				headers = append(headers, k)
			}
		}
	}

	rows := make([][]string, len(objects))
	for i, obj := range objects {
		row := make([]string, len(headers))
		for j, h := range headers {
			v, ok := obj[h]
			if !ok || v == nil {
				row[j] = ""
			} else {
				row[j] = fmt.Sprintf("%v", v)
			}
		}
		rows[i] = row
	}

	tableName := tableNameFromPath(path)
	return loadIntoSQLite(tableName, headers, rows)
}

func loadIntoSQLite(tableName string, headers []string, rows [][]string) (*fileDriver, error) {
	d, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("creating in-memory db: %w", err)
	}

	// Build CREATE TABLE — all columns are TEXT... Don't have a good way to work arround this for now
	var cols []string
	for _, h := range headers {
		cols = append(cols, fmt.Sprintf("%q TEXT", sanitizeColumn(h)))
	}
	createSQL := fmt.Sprintf("CREATE TABLE %q (%s)", tableName, strings.Join(cols, ", "))
	if _, err := d.Exec(createSQL); err != nil {
		d.Close()
		return nil, fmt.Errorf("creating table: %w", err)
	}

	// Bulk insert using a transaction.
	tx, err := d.Begin()
	if err != nil {
		d.Close()
		return nil, err
	}

	placeholders := make([]string, len(headers))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	insertSQL := fmt.Sprintf("INSERT INTO %q VALUES (%s)", tableName, strings.Join(placeholders, ", "))
	stmt, err := tx.Prepare(insertSQL)
	if err != nil {
		tx.Rollback()
		d.Close()
		return nil, err
	}

	for _, row := range rows {
		vals := make([]any, len(headers))
		for i := range headers {
			if i < len(row) {
				vals[i] = row[i]
			} else {
				vals[i] = ""
			}
		}
		if _, err := stmt.Exec(vals...); err != nil {
			tx.Rollback()
			d.Close()
			return nil, fmt.Errorf("inserting row: %w", err)
		}
	}
	stmt.Close()
	if err := tx.Commit(); err != nil {
		d.Close()
		return nil, err
	}

	return &fileDriver{sqlite: &sqliteDriver{db: d, maxRows: 0}, tableName: tableName}, nil
}

func (f *fileDriver) Execute(ctx context.Context, sqlText string) (*QueryResult, error) {
	return f.sqlite.Execute(ctx, f.injectFrom(sqlText))
}

// injectFrom rewrites SQL statements that are missing a FROM clause by
// inserting FROM <tableName>. Only applies to SELECT statements.
func (f *fileDriver) injectFrom(sqlText string) string {
	stmts := splitStatements(sqlText)
	for i, stmt := range stmts {
		stmts[i] = f.injectFromSingle(stmt)
	}
	return strings.Join(stmts, ";\n")
}

func (f *fileDriver) injectFromSingle(stmt string) string {
	trimmed := strings.TrimSpace(stmt)
	upper := strings.ToUpper(trimmed)
	if !strings.HasPrefix(upper, "SELECT") {
		return stmt
	}
	// Already has FROM — leave it alone.
	if hasKeyword(upper, "FROM") {
		return stmt
	}
	// Find the best insertion point: before WHERE, GROUP BY, HAVING,
	// ORDER BY, LIMIT, or at the end of the statement.
	fromClause := " FROM " + fmt.Sprintf("%q", f.tableName)
	insertBefore := []string{"WHERE", "GROUP BY", "HAVING", "ORDER BY", "LIMIT", "UNION"}
	for _, kw := range insertBefore {
		idx := keywordIndex(upper, kw)
		if idx >= 0 {
			return trimmed[:idx] + fromClause + " " + trimmed[idx:]
		}
	}
	return trimmed + fromClause
}

func hasKeyword(upper, kw string) bool {
	return keywordIndex(upper, kw) >= 0
}

func keywordIndex(upper, kw string) int {
	masked := maskStringLiterals(upper)
	start := 0
	for {
		idx := strings.Index(masked[start:], kw)
		if idx < 0 {
			return -1
		}
		pos := start + idx
		if pos > 0 && isIdentChar(upper[pos-1]) {
			start = pos + len(kw)
			continue
		}
		end := pos + len(kw)
		if end < len(upper) && isIdentChar(upper[end]) {
			start = end
			continue
		}
		return pos
	}
}

func maskStringLiterals(s string) string {
	buf := []byte(s)
	inStr := false
	for i := 0; i < len(buf); i++ {
		if buf[i] == '\'' {
			if inStr {
				buf[i] = ' '
				inStr = false
			} else {
				buf[i] = ' '
				inStr = true
			}
		} else if inStr {
			buf[i] = ' '
		}
	}
	return string(buf)
}

func isIdentChar(ch byte) bool {
	return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') ||
		(ch >= '0' && ch <= '9') || ch == '_'
}

func (f *fileDriver) ListTables(ctx context.Context) ([]TableInfo, error) {
	return f.sqlite.ListTables(ctx)
}

func (f *fileDriver) ListColumns(ctx context.Context, schema, tableName string) ([]ColumnInfo, error) {
	return f.sqlite.ListColumns(ctx, schema, tableName)
}

func (f *fileDriver) Close(ctx context.Context) error {
	return f.sqlite.Close(ctx)
}

// tableNameFromPath derives a table name from the file path.
// e.g. "/home/user/test.csv" → "test"
func tableNameFromPath(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	// Replace non-alphanumeric chars with underscores.
	var sb strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('_')
		}
	}
	result := sb.String()
	if result == "" {
		return "data"
	}
	return result
}

func sanitizeColumn(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "column"
	}
	return name
}
