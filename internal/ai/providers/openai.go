package providers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/tardanoir/seshat/internal/ai"
)

const openAIDefaultModel = "gpt-4.1-mini"
const openAIDefaultURL = "https://api.openai.com/v1/chat/completions"

type OpenAI struct {
	APIKey  string
	Model   string
	BaseURL string
	client  *http.Client
}

func NewOpenAI(apiKey, model, baseURL string, timeout time.Duration) *OpenAI {
	return &OpenAI{
		APIKey:  apiKey,
		Model:   pickStr(model, openAIDefaultModel),
		BaseURL: pickStr(baseURL, openAIDefaultURL),
		client:  newClient(timeout),
	}
}

func (o *OpenAI) Name() string { return "openai" }

func (o *OpenAI) Generate(ctx context.Context, req ai.Request) (ai.Response, error) {
	if o.APIKey == "" {
		return ai.Response{}, errors.New("openai: api_key not set")
	}
	body := map[string]any{
		"model":       o.Model,
		"temperature": 0,
		"messages": []map[string]any{
			{"role": "system", "content": ai.SystemPrompt},
			{"role": "user", "content": ai.BuildPrompt(req)},
		},
	}
	headers := map[string]string{
		"Authorization": "Bearer " + o.APIKey,
	}
	var resp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := postJSON(ctx, o.client, o.BaseURL, headers, body, &resp); err != nil {
		return ai.Response{}, err
	}
	raw := ""
	if len(resp.Choices) > 0 {
		raw = resp.Choices[0].Message.Content
	}
	return ai.Response{
		SQL:      ai.ExtractSQL(raw),
		Raw:      raw,
		Provider: o.Name(),
	}, nil
}

// ChatStream implements ai.ChatProvider with the OpenAI chat completions streaming API.
func (o *OpenAI) ChatStream(ctx context.Context, req ai.ChatRequest) (<-chan ai.ChatChunk, error) {
	if o.APIKey == "" {
		return nil, errors.New("openai: api_key not set")
	}
	msgs := []map[string]any{
		{"role": "system", "content": ai.ChatSystemPrompt + "\n\n" + ai.BuildChatContext(req)},
	}
	for _, m := range req.Messages {
		msgs = append(msgs, map[string]any{"role": m.Role, "content": m.Content})
	}
	body := map[string]any{
		"model":       o.Model,
		"temperature": 0,
		"stream":      true,
		"messages":    msgs,
	}
	headers := map[string]string{"Authorization": "Bearer " + o.APIKey}
	return postSSE(ctx, o.BaseURL, headers, body, func(data string) (string, bool, error) {
		if data == "[DONE]" {
			return "", true, nil
		}
		var ev struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			return "", false, nil
		}
		if len(ev.Choices) > 0 {
			return ev.Choices[0].Delta.Content, false, nil
		}
		return "", false, nil
	})
}
