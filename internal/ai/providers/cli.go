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
	prompt := ai.SystemPrompt + "\n\n" + ai.BuildPrompt(req)

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
			return ai.Response{}, ctx.Err()
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
		return ai.Response{}, fmt.Errorf("%s: %s", c.Name(), msg)
	}
	raw := stdout.String()
	return ai.Response{
		SQL:      ai.ExtractSQL(raw),
		Raw:      raw,
		Provider: c.Name(),
	}, nil
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
