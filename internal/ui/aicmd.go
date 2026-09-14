package ui

import (
	"context"
	"errors"
	"strings"

	"github.com/tardanoir/seshat/internal/ai"
	"github.com/tardanoir/seshat/internal/config"
	"github.com/tardanoir/seshat/internal/secret"
	"github.com/tardanoir/seshat/internal/ui/queryeditor"

	tea "charm.land/bubbletea/v2"
)

// AIResultMsg carries a successful generation back to the App.
type AIResultMsg struct {
	SQL      string
	Provider string
	Block    ai.CommentBlock
}

// AIErrorMsg surfaces a generation failure (or context cancellation).
type AIErrorMsg struct {
	Err error
}

// generateAICmd kicks off provider.Generate in a goroutine and emits the
// appropriate result message. Cancellation is honored via ctx.
func generateAICmd(ctx context.Context, provider ai.Provider, req ai.Request, block ai.CommentBlock) tea.Cmd {
	return func() tea.Msg {
		resp, err := provider.Generate(ctx, req)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return AIErrorMsg{Err: err}
			}
			return AIErrorMsg{Err: err}
		}
		return AIResultMsg{
			SQL:      resp.SQL,
			Provider: resp.Provider,
			Block:    block,
		}
	}
}

// aiConfigFrom converts the on-disk config into the ai package's config shape.
// Kept here so internal/ai stays free of any config dependency. "keyring:<name>"
// API keys are resolved against the OS secret store here; "$ENV" and literals
// pass through for ai.ResolveAPIKey / direct use.
func aiConfigFrom(c config.AIConfig) ai.Config {
	out := ai.Config{
		DefaultProvider: c.DefaultProvider,
		Providers:       make(map[string]ai.ProviderConf, len(c.Providers)),
	}
	for name, p := range c.Providers {
		out.Providers[name] = ai.ProviderConf{
			Kind:     p.Kind,
			APIKey:   resolveProviderKey(p.APIKey),
			Model:    p.Model,
			BaseURL:  p.BaseURL,
			Argv:     p.Argv,
			TimeoutS: p.TimeoutS,
		}
	}
	return out
}

// resolveProviderKey expands a "keyring:<name>" reference via the OS secret
// store. Other forms (e.g. "$ENV_VAR", literals) are returned unchanged for
// later resolution by ai.ResolveAPIKey. A keyring lookup failure yields "" so
// the provider reports a clean "api_key not set" rather than sending garbage.
func resolveProviderKey(s string) string {
	if name, ok := strings.CutPrefix(s, "keyring:"); ok {
		if v, err := secret.Get(name); err == nil {
			return v
		}
		return ""
	}
	return s
}

// schemaFrom maps the App's cached schema into the ai package's shape.
func schemaFrom(tables []queryeditor.TableRef, columnsBy map[string][]queryeditor.ColumnRef) []ai.SchemaTable {
	schema := make([]ai.SchemaTable, 0, len(tables))
	for _, t := range tables {
		key := t.Schema + "." + t.Name
		refs := columnsBy[key]
		cols := make([]ai.SchemaColumn, len(refs))
		for i, r := range refs {
			cols[i] = ai.SchemaColumn{Name: r.Name, DataType: r.DataType}
		}
		schema = append(schema, ai.SchemaTable{Schema: t.Schema, Name: t.Name, Columns: cols})
	}
	return schema
}

// buildAIRequest assembles a one-shot Request from the App's cached schema.
func buildAIRequest(connName, dialect, intent string, tables []queryeditor.TableRef, columnsBy map[string][]queryeditor.ColumnRef) ai.Request {
	return ai.Request{
		Intent:     intent,
		Dialect:    dialect,
		Connection: connName,
		Schema:     schemaFrom(tables, columnsBy),
	}
}

// buildChatRequest assembles a multi-turn ChatRequest from the conversation and
// the App's cached schema.
func buildChatRequest(connName, dialect string, messages []ai.ChatMessage, tables []queryeditor.TableRef, columnsBy map[string][]queryeditor.ColumnRef) ai.ChatRequest {
	return ai.ChatRequest{
		Messages:   messages,
		Dialect:    dialect,
		Connection: connName,
		Schema:     schemaFrom(tables, columnsBy),
	}
}

// AIChatChunkMsg carries one streamed delta back to the App's update loop.
type AIChatChunkMsg struct {
	Delta string
	Done  bool
	Err   error
}

// aiChatStreamMsg hands the open chunk channel to the App after ChatStream
// succeeds, so the App can drain it one chunk per tea.Cmd.
type aiChatStreamMsg struct{ ch <-chan ai.ChatChunk }

// startChatCmd opens the stream; on success it returns aiChatStreamMsg, else an
// error chunk.
func startChatCmd(ctx context.Context, provider ai.ChatProvider, req ai.ChatRequest) tea.Cmd {
	return func() tea.Msg {
		ch, err := provider.ChatStream(ctx, req)
		if err != nil {
			return AIChatChunkMsg{Done: true, Err: err}
		}
		return aiChatStreamMsg{ch: ch}
	}
}

// recvChatCmd reads the next chunk from the stream channel.
func recvChatCmd(ch <-chan ai.ChatChunk) tea.Cmd {
	return func() tea.Msg {
		c, ok := <-ch
		if !ok {
			return AIChatChunkMsg{Done: true}
		}
		return AIChatChunkMsg{Delta: c.Delta, Done: c.Done, Err: c.Err}
	}
}
