package ui

import (
	"context"
	"errors"

	"github.com/tardanoir/seshat/internal/ai"
	"github.com/tardanoir/seshat/internal/config"
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
// Kept here so internal/ai stays free of any config dependency.
func aiConfigFrom(c config.AIConfig) ai.Config {
	out := ai.Config{
		DefaultProvider: c.DefaultProvider,
		Providers:       make(map[string]ai.ProviderConf, len(c.Providers)),
	}
	for name, p := range c.Providers {
		out.Providers[name] = ai.ProviderConf{
			Kind:     p.Kind,
			APIKey:   p.APIKey,
			Model:    p.Model,
			BaseURL:  p.BaseURL,
			Argv:     p.Argv,
			TimeoutS: p.TimeoutS,
		}
	}
	return out
}

// buildAIRequest assembles a Request from the App's cached schema.
func buildAIRequest(connName, dialect, intent string, tables []queryeditor.TableRef, columnsBy map[string][]queryeditor.ColumnRef) ai.Request {
	schema := make([]ai.SchemaTable, 0, len(tables))
	for _, t := range tables {
		key := t.Schema + "." + t.Name
		refs := columnsBy[key]
		cols := make([]ai.SchemaColumn, len(refs))
		for i, r := range refs {
			cols[i] = ai.SchemaColumn{Name: r.Name, DataType: r.DataType}
		}
		schema = append(schema, ai.SchemaTable{
			Schema:  t.Schema,
			Name:    t.Name,
			Columns: cols,
		})
	}
	return ai.Request{
		Intent:     intent,
		Dialect:    dialect,
		Connection: connName,
		Schema:     schema,
	}
}
