package queryeditor

import (
	"sort"
	"strings"
	"unicode"
)

type TableRef struct {
	Schema string
	Name   string
}

type ColumnRef struct {
	Schema string
	Table  string
	Name   string
}

type provider struct {
	keywords []string
	tables   []string
	columns  []string
}

var sqlKeywords = []string{
	"AND", "AS", "ASC", "ALL", "ALTER", "BEGIN", "BETWEEN", "BOOLEAN", "BY",
	"BIGINT", "CASCADE", "CASE", "CAST", "CHECK", "COMMIT", "CONSTRAINT",
	"COUNT", "COALESCE", "CREATE", "CROSS", "DATE", "DEFAULT", "DELETE",
	"DESC", "DISTINCT", "DROP", "ELSE", "END", "EXISTS", "FALSE", "FLOAT",
	"FOREIGN", "FROM", "FULL", "GRANT", "GROUP", "HAVING", "ILIKE", "IN",
	"INDEX", "INNER", "INSERT", "INTEGER", "INTERVAL", "INTO", "IS", "JOIN",
	"KEY", "LEFT", "LIKE", "LIMIT", "NOT", "NULL", "NUMERIC", "OFFSET", "ON",
	"OR", "ORDER", "OUTER", "PRIMARY", "REFERENCES", "RESTRICT", "RETURNING",
	"REVOKE", "RIGHT", "ROLLBACK", "SCHEMA", "SELECT", "SERIAL", "SET",
	"SMALLINT", "SUM", "TABLE", "TEXT", "THEN", "TIME", "TIMESTAMP", "TRUE",
	"UNION", "UNIQUE", "UPDATE", "VALUES", "VARCHAR", "VIEW", "WHEN", "WHERE",
	"WITH",
}

func newProvider() *provider {
	return &provider{keywords: sqlKeywords}
}

func (p *provider) setSchema(tables []TableRef, columns []ColumnRef) {
	tSet := map[string]struct{}{}
	for _, t := range tables {
		tSet[t.Name] = struct{}{}
		if t.Schema != "" && t.Schema != "public" {
			tSet[t.Schema+"."+t.Name] = struct{}{}
		}
	}
	p.tables = keys(tSet)

	cSet := map[string]struct{}{}
	for _, c := range columns {
		cSet[c.Name] = struct{}{}
	}
	p.columns = keys(cSet)
}

func keys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

type completionContext int

const (
	ctxAny completionContext = iota
	ctxTable
	ctxColumn
)

type SuggestionKind int

const (
	KindKeyword SuggestionKind = iota
	KindTable
	KindColumn
)

func (k SuggestionKind) Label() string {
	switch k {
	case KindTable:
		return "tbl"
	case KindColumn:
		return "col"
	default:
		return "kw"
	}
}

type Suggestion struct {
	Text string
	Kind SuggestionKind
}

// suggest returns a ranked list of completions for prefix. Ordering reflects
// ctx: tables-first after FROM/JOIN, columns-first after WHERE/SELECT/etc.
// Keywords only appear when ctx is ambiguous.
func (p *provider) suggest(prefix string, ctx completionContext, limit int) []Suggestion {
	if prefix == "" || limit <= 0 {
		return nil
	}
	upper := hasUpper(prefix)
	lp := strings.ToLower(prefix)

	match := func(pool []string, kind SuggestionKind) []Suggestion {
		var out []Suggestion
		for _, w := range pool {
			lw := strings.ToLower(w)
			if lw == lp {
				continue
			}
			if strings.HasPrefix(lw, lp) {
				out = append(out, Suggestion{Text: w, Kind: kind})
			}
		}
		return out
	}

	cols := match(p.columns, KindColumn)
	tbls := match(p.tables, KindTable)
	kws := match(p.keywords, KindKeyword)
	if !upper {
		for i := range kws {
			kws[i].Text = strings.ToLower(kws[i].Text)
		}
	}

	var order [][]Suggestion
	switch ctx {
	case ctxTable:
		order = [][]Suggestion{tbls, cols}
	case ctxColumn:
		order = [][]Suggestion{cols, tbls}
	default:
		order = [][]Suggestion{cols, tbls, kws}
	}

	var out []Suggestion
	for _, group := range order {
		for _, s := range group {
			out = append(out, s)
			if len(out) >= limit {
				return out
			}
		}
	}
	return out
}

// sqlContextAt scans the text before the cursor and returns the clause
// context we're in. Recognizes the nearest "driving" keyword.
func sqlContextAt(text string) completionContext {
	// Strip string literals + line comments to avoid false matches.
	cleaned := stripStringsAndComments(text)
	upper := strings.ToUpper(cleaned)
	tokens := tokenize(upper)

	tableKW := map[string]bool{
		"FROM": true, "JOIN": true, "INTO": true, "UPDATE": true,
		"TABLE": true, "TRUNCATE": true,
	}
	columnKW := map[string]bool{
		"WHERE": true, "AND": true, "OR": true, "ON": true,
		"HAVING": true, "SELECT": true, "SET": true,
		"BY": true, "USING": true,
	}

	for i := len(tokens) - 1; i >= 0; i-- {
		t := tokens[i]
		if tableKW[t] {
			return ctxTable
		}
		if columnKW[t] {
			return ctxColumn
		}
	}
	return ctxAny
}

func stripStringsAndComments(s string) string {
	var b strings.Builder
	inSingle, inDouble, inLine := false, false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inLine {
			if c == '\n' {
				inLine = false
				b.WriteByte(c)
			}
			continue
		}
		if inSingle {
			if c == '\'' {
				inSingle = false
			}
			continue
		}
		if inDouble {
			if c == '"' {
				inDouble = false
			}
			continue
		}
		if c == '-' && i+1 < len(s) && s[i+1] == '-' {
			inLine = true
			i++
			continue
		}
		if c == '\'' {
			inSingle = true
			continue
		}
		if c == '"' {
			inDouble = true
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

func tokenize(s string) []string {
	var toks []string
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			toks = append(toks, string(cur))
			cur = cur[:0]
		}
	}
	for _, r := range s {
		if isWordRune(r) && r != '.' {
			cur = append(cur, r)
		} else {
			flush()
		}
	}
	flush()
	return toks
}

func hasUpper(s string) bool {
	for _, r := range s {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

// wordBeforeCursor returns the identifier-like run of runes ending at col on
// line. Returns the word and the rune index where it starts. Only considers
// [A-Za-z0-9_.] runes.
func wordBeforeCursor(line string, col int) (word string, start int) {
	runes := []rune(line)
	if col > len(runes) {
		col = len(runes)
	}
	end := col
	i := end
	for i > 0 && isWordRune(runes[i-1]) {
		i--
	}
	return string(runes[i:end]), i
}

func isWordRune(r rune) bool {
	return r == '_' || r == '.' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// cursorAtWordEnd reports whether the character at col sits outside a word,
// so the cursor is at the boundary of the word that ends at col.
func cursorAtWordEnd(line string, col int) bool {
	runes := []rune(line)
	if col >= len(runes) {
		return true
	}
	return !isWordRune(runes[col])
}
