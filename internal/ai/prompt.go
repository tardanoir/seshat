package ai

import (
	"fmt"
	"regexp"
	"strings"
)

const SystemPrompt = "You are a SQL generation assistant. Reply with a single SQL statement and nothing else. No prose, no explanations, no markdown fences."

// ChatSystemPrompt seeds the conversational chat mode. Unlike SystemPrompt it
// allows brief prose, but asks for SQL inside a fenced block so ExtractSQL can
// pull a runnable statement back out.
const ChatSystemPrompt = "You are a SQL assistant embedded in a terminal SQL client. " +
	"Help the user write, fix, and refine SQL for their database. Keep explanations brief. " +
	"Whenever you provide SQL, put it in a single ```sql fenced code block so it can be extracted and run."

// BuildChatContext returns a system-context string describing the active
// connection, dialect, and schema, used to seed a chat conversation. It reuses
// formatSchema and the maxPromptBytes budget from BuildPrompt.
func BuildChatContext(req ChatRequest) string {
	dialect := req.Dialect
	if dialect == "" {
		dialect = "sql"
	}
	header := fmt.Sprintf("Connection: %s (%s)\nDialect: %s\n", safe(req.Connection), dialect, dialect)
	for _, withCols := range []bool{true, false} {
		ctx := header + "Schema:\n" + formatSchema(req.Schema, withCols)
		if len(ctx) <= maxPromptBytes {
			return ctx
		}
	}
	// Last resort: truncate the table list itself.
	for n := len(req.Schema); n > 0; n-- {
		ctx := header + "Schema:\n" + formatSchema(req.Schema[:n], false) + "(schema truncated)\n"
		if len(ctx) <= maxPromptBytes {
			return ctx
		}
	}
	return header
}

// maxPromptBytes caps the user-prompt size to keep token costs predictable.
// When the schema is too large we drop columns first and then whole tables.
const maxPromptBytes = 80 * 1024

// BuildPrompt formats the user prompt sent to the LLM.
func BuildPrompt(req Request) string {
	dialect := req.Dialect
	if dialect == "" {
		dialect = "sql"
	}

	header := fmt.Sprintf(
		"-- Connection: %s (%s)\n-- Dialect: %s\n",
		safe(req.Connection), dialect, dialect,
	)
	footer := "-- Request:\n" + req.Intent + "\n"

	schemaText := formatSchema(req.Schema, true)
	prompt := header + "-- Schema:\n" + schemaText + footer
	if len(prompt) <= maxPromptBytes {
		return prompt
	}

	// Drop column lists first.
	schemaText = formatSchema(req.Schema, false)
	prompt = header + "-- Schema:\n" + schemaText + footer
	if len(prompt) <= maxPromptBytes {
		return prompt
	}

	// Last resort: truncate the table list itself.
	for n := len(req.Schema); n > 0; n-- {
		schemaText = formatSchema(req.Schema[:n], false)
		prompt = header + "-- Schema:\n" + schemaText + "-- (schema truncated)\n" + footer
		if len(prompt) <= maxPromptBytes {
			return prompt
		}
	}
	return header + footer
}

func formatSchema(tables []SchemaTable, withColumns bool) string {
	var b strings.Builder
	for _, t := range tables {
		schema := t.Schema
		if schema == "" {
			schema = "public"
		}
		if withColumns && len(t.Columns) > 0 {
			cols := make([]string, len(t.Columns))
			for i, c := range t.Columns {
				if c.DataType != "" {
					cols[i] = c.Name + " " + c.DataType
				} else {
					cols[i] = c.Name
				}
			}
			fmt.Fprintf(&b, "%s.%s(%s)\n", schema, t.Name, strings.Join(cols, ", "))
		} else {
			fmt.Fprintf(&b, "%s.%s\n", schema, t.Name)
		}
	}
	return b.String()
}

var fenceRe = regexp.MustCompile("(?s)```(?:sql|SQL)?\\s*(.*?)```")

// ExtractSQL strips markdown fences and surrounding prose from raw LLM output.
// If a fenced block is present, the longest one wins. Otherwise the trimmed
// raw text is returned.
func ExtractSQL(raw string) string {
	matches := fenceRe.FindAllStringSubmatch(raw, -1)
	if len(matches) > 0 {
		best := ""
		for _, m := range matches {
			if len(m[1]) > len(best) {
				best = m[1]
			}
		}
		return strings.TrimSpace(best)
	}
	return strings.TrimSpace(raw)
}

func safe(s string) string {
	if s == "" {
		return "(unknown)"
	}
	return s
}
