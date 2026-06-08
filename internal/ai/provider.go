// Package ai provides natural-language-to-SQL generation backed by either
// HTTP API providers (Anthropic, OpenAI, Gemini) or local CLI subprocesses
// (claude, opencode). The Provider interface is intentionally narrow so the
// UI layer can stay provider-agnostic.
package ai

import "context"

type SchemaColumn struct {
	Name     string
	DataType string
}

type SchemaTable struct {
	Schema  string
	Name    string
	Columns []SchemaColumn
}

type Request struct {
	Intent     string
	Dialect    string
	Connection string
	Schema     []SchemaTable
}

type Response struct {
	SQL      string
	Raw      string
	Provider string
}

type Provider interface {
	Name() string
	Generate(ctx context.Context, req Request) (Response, error)
}
