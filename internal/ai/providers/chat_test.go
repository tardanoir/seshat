package providers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	"github.com/tardanoir/seshat/internal/ai"
)

// drain collects all deltas from a chat stream until Done or error.
func drain(ch <-chan ai.ChatChunk) (string, error) {
	var b strings.Builder
	for c := range ch {
		if c.Err != nil {
			return b.String(), c.Err
		}
		b.WriteString(c.Delta)
		if c.Done {
			break
		}
	}
	return b.String(), nil
}

func sseServer(t *testing.T, lines ...string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for _, l := range lines {
			fmt.Fprintf(w, "%s\n\n", l)
		}
	}))
}

func TestOpenAI_ChatStream(t *testing.T) {
	srv := sseServer(t,
		`data: {"choices":[{"delta":{"content":"SELECT "}}]}`,
		`data: {"choices":[{"delta":{"content":"1"}}]}`,
		`data: [DONE]`,
	)
	defer srv.Close()

	p := NewOpenAI("k", "", srv.URL, 0)
	ch, err := p.ChatStream(context.Background(), ai.ChatRequest{Messages: []ai.ChatMessage{{Role: "user", Content: "hi"}}})
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	got, err := drain(ch)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}
	if got != "SELECT 1" {
		t.Errorf("got %q, want %q", got, "SELECT 1")
	}
}

func TestAnthropic_ChatStream(t *testing.T) {
	srv := sseServer(t,
		`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"SELECT "}}`,
		`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"1"}}`,
		`data: {"type":"message_stop"}`,
	)
	defer srv.Close()

	p := NewAnthropic("k", "", srv.URL, 0)
	ch, err := p.ChatStream(context.Background(), ai.ChatRequest{Messages: []ai.ChatMessage{{Role: "user", Content: "hi"}}})
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	got, err := drain(ch)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}
	if got != "SELECT 1" {
		t.Errorf("got %q, want %q", got, "SELECT 1")
	}
}

func TestGemini_ChatStream(t *testing.T) {
	// Gemini has no explicit terminator event; the stream ends at body EOF.
	srv := sseServer(t,
		`data: {"candidates":[{"content":{"parts":[{"text":"SELECT "}]}}]}`,
		`data: {"candidates":[{"content":{"parts":[{"text":"1"}]}}]}`,
	)
	defer srv.Close()

	p := NewGemini("k", "", srv.URL, 0)
	ch, err := p.ChatStream(context.Background(), ai.ChatRequest{Messages: []ai.ChatMessage{{Role: "user", Content: "hi"}}})
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	got, err := drain(ch)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}
	if got != "SELECT 1" {
		t.Errorf("got %q, want %q", got, "SELECT 1")
	}
}

func TestChatStream_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusUnauthorized)
	}))
	defer srv.Close()

	p := NewOpenAI("k", "", srv.URL, 0)
	if _, err := p.ChatStream(context.Background(), ai.ChatRequest{Messages: []ai.ChatMessage{{Role: "user", Content: "hi"}}}); err == nil {
		t.Fatal("expected error on non-2xx response")
	}
}

func TestCLI_ChatStream(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("relies on /bin/sh")
	}
	// Echo the prompt back; the assembled prompt must contain the user's turn.
	p := NewCLI("test", []string{"sh", "-c", "cat"}, 0)
	ch, err := p.ChatStream(context.Background(), ai.ChatRequest{Messages: []ai.ChatMessage{{Role: "user", Content: "count the rows"}}})
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}
	got, err := drain(ch)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}
	if !strings.Contains(got, "count the rows") {
		t.Errorf("expected user turn in echoed prompt; got %q", got)
	}
}
