package providers

import (
	"context"
	"runtime"
	"strings"
	"testing"

	"github.com/tardanoir/seshat/internal/ai"
)

func TestCLI_PromptPlaceholderSubstituted(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("relies on /bin/sh")
	}
	// Use sh to echo back the substituted argv so we can verify the placeholder
	// was replaced.
	p := NewCLI("test", []string{"sh", "-c", "printf '%s' \"$1\"", "_", "{prompt}"}, 0)
	resp, err := p.Generate(context.Background(), ai.Request{Intent: "show me sales"})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.Contains(resp.Raw, "show me sales") {
		t.Errorf("expected prompt in stdout; got %q", resp.Raw)
	}
}

func TestCLI_StdinFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("relies on /bin/cat")
	}
	// No placeholder => prompt should be sent on stdin.
	p := NewCLI("test", []string{"cat"}, 0)
	resp, err := p.Generate(context.Background(), ai.Request{Intent: "stdin path"})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !strings.Contains(resp.Raw, "stdin path") {
		t.Errorf("expected prompt echoed via stdin; got %q", resp.Raw)
	}
}

func TestCLI_NonZeroExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("relies on /bin/sh")
	}
	p := NewCLI("test", []string{"sh", "-c", "echo boom >&2; exit 7"}, 0)
	if _, err := p.Generate(context.Background(), ai.Request{}); err == nil {
		t.Fatal("expected error")
	} else if !strings.Contains(err.Error(), "boom") {
		t.Errorf("expected stderr in error; got %v", err)
	}
}
