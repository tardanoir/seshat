package providers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/tardanoir/seshat/internal/ai"
)

// promptPlaceholder marks where the constructed prompt is substituted into the
// user-supplied argv template. Example: ["claude", "-p", "{prompt}"].
const promptPlaceholder = "{prompt}"

// CLI runs an external command, passing the prompt either via a placeholder in
// argv or, if no placeholder is present, on stdin. The first form fits CLIs
// that take a positional prompt; the second works for tools that read from
// stdin.
type CLI struct {
	NameValue string
	Argv      []string
	Timeout   time.Duration
}

func NewCLI(name string, argv []string, timeout time.Duration) *CLI {
	return &CLI{NameValue: name, Argv: argv, Timeout: timeout}
}

func (c *CLI) Name() string {
	if c.NameValue == "" {
		return "cli"
	}
	return c.NameValue
}

func (c *CLI) Generate(ctx context.Context, req ai.Request) (ai.Response, error) {
	if len(c.Argv) == 0 {
		return ai.Response{}, errors.New("cli: argv not configured")
	}
	raw, err := c.runPrompt(ctx, ai.SystemPrompt+"\n\n"+ai.BuildPrompt(req))
	if err != nil {
		return ai.Response{}, err
	}
	return ai.Response{
		SQL:      ai.ExtractSQL(raw),
		Raw:      raw,
		Provider: c.Name(),
	}, nil
}

// runPrompt runs the configured subprocess with prompt and returns its stdout.
func (c *CLI) runPrompt(ctx context.Context, prompt string) (string, error) {
	args, useStdin := substitutePrompt(c.Argv, prompt)

	if c.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	if useStdin {
		cmd.Stdin = strings.NewReader(prompt)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		// Some CLIs (e.g. claude when not logged in) print errors to stdout
		// rather than stderr. Fall back to stdout when stderr is empty so the
		// user sees something actionable instead of just "exit status N".
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s: %s", c.Name(), msg)
	}
	return stdout.String(), nil
}

// ChatStream implements ai.ChatProvider for CLI tools by flattening the
// conversation into one prompt and emitting the full reply as a single chunk
// (no token-level streaming).
func (c *CLI) ChatStream(ctx context.Context, req ai.ChatRequest) (<-chan ai.ChatChunk, error) {
	if len(c.Argv) == 0 {
		return nil, errors.New("cli: argv not configured")
	}
	var b strings.Builder
	b.WriteString(ai.ChatSystemPrompt + "\n\n")
	b.WriteString(ai.BuildChatContext(req) + "\n\n")
	for _, m := range req.Messages {
		b.WriteString(strings.ToUpper(m.Role) + ": " + m.Content + "\n")
	}
	b.WriteString("ASSISTANT: ")
	prompt := b.String()

	ch := make(chan ai.ChatChunk, 2)
	go func() {
		defer close(ch)
		out, err := c.runPrompt(ctx, prompt)
		if err != nil {
			ch <- ai.ChatChunk{Err: err}
			return
		}
		ch <- ai.ChatChunk{Delta: strings.TrimSpace(out)}
		ch <- ai.ChatChunk{Done: true}
	}()
	return ch, nil
}

func substitutePrompt(argv []string, prompt string) (out []string, stdin bool) {
	out = make([]string, len(argv))
	found := false
	for i, a := range argv {
		if strings.Contains(a, promptPlaceholder) {
			out[i] = strings.ReplaceAll(a, promptPlaceholder, prompt)
			found = true
		} else {
			out[i] = a
		}
	}
	return out, !found
}
