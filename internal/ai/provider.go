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

// ChatMessage is one turn in a multi-turn conversation.
type ChatMessage struct {
	Role    string // "system" | "user" | "assistant"
	Content string
}

// ChatRequest is a multi-turn chat completion request. Dialect/Connection/Schema
// let providers ground replies in the live database, mirroring Request.
type ChatRequest struct {
	Messages   []ChatMessage
	Dialect    string
	Connection string
	Schema     []SchemaTable
}

// ChatChunk is a single streamed delta. The stream ends with a chunk where
// Done is true (Delta empty) or where Err is non-nil.
type ChatChunk struct {
	Delta string
	Done  bool
	Err   error
}

// ChatProvider is the optional capability for multi-turn chat with token
// streaming. Callers type-assert Provider to ChatProvider and degrade to a
// friendly message when a provider doesn't implement it.
type ChatProvider interface {
	Provider
	ChatStream(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error)
}
