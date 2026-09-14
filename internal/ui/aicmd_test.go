package ui

import (
	"context"
	"testing"

	"github.com/tardanoir/seshat/internal/ai"
)

func TestResolveProviderKey(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"sk-literal", "sk-literal"},     // literal passes through
		{"$MY_VAR", "$MY_VAR"},           // env ref left for ai.ResolveAPIKey
		{"keyring:not-present-xyz", ""},  // missing/unavailable keyring → empty
	}
	for _, c := range cases {
		if got := resolveProviderKey(c.in); got != c.want {
			t.Errorf("resolveProviderKey(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestFakeChatStreamDrain verifies the ChatProvider streaming contract end to
// end against the fake provider and the recvChatCmd drain helper.
func TestFakeChatStreamDrain(t *testing.T) {
	fake := &ai.FakeProvider{Reply: "```sql\nSELECT 1\n```"}
	ch, err := fake.ChatStream(context.Background(), ai.ChatRequest{
		Messages: []ai.ChatMessage{{Role: "user", Content: "give me one"}},
	})
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}

	var got string
	for {
		msg := recvChatCmd(ch)()
		chunk, ok := msg.(AIChatChunkMsg)
		if !ok {
			t.Fatalf("expected AIChatChunkMsg, got %T", msg)
		}
		if chunk.Err != nil {
			t.Fatalf("chunk error: %v", chunk.Err)
		}
		got += chunk.Delta
		if chunk.Done {
			break
		}
	}
	if got != "```sql\nSELECT 1\n```" {
		t.Errorf("assembled reply = %q", got)
	}
	if fake.GotChat.Messages[0].Content != "give me one" {
		t.Errorf("provider did not receive the request: %+v", fake.GotChat)
	}
}
