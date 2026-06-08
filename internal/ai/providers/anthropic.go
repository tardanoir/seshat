package providers

import (
	"context"
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

func pickStr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
