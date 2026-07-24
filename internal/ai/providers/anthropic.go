package providers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/tardanoir/seshat/internal/ai"
)

const anthropicDefaultModel = "claude-sonnet-4-5"
const anthropicDefaultURL = "https://api.anthropic.com/v1/messages"

type Anthropic struct {
	APIKey  string
	Model   string
	BaseURL string
	client  *http.Client
}

func NewAnthropic(apiKey, model, baseURL string, timeout time.Duration) *Anthropic {
	return &Anthropic{
		APIKey:  apiKey,
		Model:   pickStr(model, anthropicDefaultModel),
		BaseURL: pickStr(baseURL, anthropicDefaultURL),
		client:  newClient(timeout),
	}
}

func (a *Anthropic) Name() string { return "anthropic" }

func (a *Anthropic) Generate(ctx context.Context, req ai.Request) (ai.Response, error) {
	if a.APIKey == "" {
		return ai.Response{}, errors.New("anthropic: api_key not set")
	}
	body := map[string]any{
		"model":      a.Model,
		"max_tokens": 1024,
		"system":     ai.SystemPrompt,
		"messages": []map[string]any{
			{"role": "user", "content": ai.BuildPrompt(req)},
		},
	}
	headers := map[string]string{
		"x-api-key":         a.APIKey,
		"anthropic-version": "2023-06-01",
	}
	var resp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := postJSON(ctx, a.client, a.BaseURL, headers, body, &resp); err != nil {
		return ai.Response{}, err
	}
	raw := ""
	for _, c := range resp.Content {
		if c.Type == "text" {
			raw += c.Text
		}
	}
	return ai.Response{
		SQL:      ai.ExtractSQL(raw),
		Raw:      raw,
		Provider: a.Name(),
	}, nil
}

// ChatStream implements ai.ChatProvider with the Anthropic Messages streaming API.
func (a *Anthropic) ChatStream(ctx context.Context, req ai.ChatRequest) (<-chan ai.ChatChunk, error) {
	if a.APIKey == "" {
		return nil, errors.New("anthropic: api_key not set")
	}
	system := ai.ChatSystemPrompt + "\n\n" + ai.BuildChatContext(req)
	msgs := make([]map[string]any, 0, len(req.Messages))
	for _, m := range req.Messages {
		if m.Role == "system" {
			system += "\n\n" + m.Content
			continue
		}
		msgs = append(msgs, map[string]any{"role": m.Role, "content": m.Content})
	}
	body := map[string]any{
		"model":      a.Model,
		"max_tokens": 2048,
		"system":     system,
		"stream":     true,
		"messages":   msgs,
	}
	headers := map[string]string{
		"x-api-key":         a.APIKey,
		"anthropic-version": "2023-06-01",
	}
	return postSSE(ctx, a.BaseURL, headers, body, func(data string) (string, bool, error) {
		var ev struct {
			Type  string `json:"type"`
			Delta struct {
				Text string `json:"text"`
			} `json:"delta"`
		}
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			return "", false, nil // ignore keepalives / unparseable events
		}
		switch ev.Type {
		case "content_block_delta":
			return ev.Delta.Text, false, nil
		case "message_stop":
			return "", true, nil
		}
		return "", false, nil
	})
}

func pickStr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
