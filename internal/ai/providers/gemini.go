package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/tardanoir/seshat/internal/ai"
)

const geminiDefaultModel = "gemini-2.5-flash"
const geminiDefaultURL = "https://generativelanguage.googleapis.com/v1beta"

type Gemini struct {
	APIKey  string
	Model   string
	BaseURL string
	client  *http.Client
}

func NewGemini(apiKey, model, baseURL string, timeout time.Duration) *Gemini {
	return &Gemini{
		APIKey:  apiKey,
		Model:   pickStr(model, geminiDefaultModel),
		BaseURL: pickStr(baseURL, geminiDefaultURL),
		client:  newClient(timeout),
	}
}

func (g *Gemini) Name() string { return "gemini" }

func (g *Gemini) Generate(ctx context.Context, req ai.Request) (ai.Response, error) {
	if g.APIKey == "" {
		return ai.Response{}, errors.New("gemini: api_key not set")
	}
	endpoint := fmt.Sprintf(
		"%s/models/%s:generateContent?key=%s",
		g.BaseURL, g.Model, url.QueryEscape(g.APIKey),
	)
	body := map[string]any{
		"systemInstruction": map[string]any{
			"parts": []map[string]any{{"text": ai.SystemPrompt}},
		},
		"contents": []map[string]any{
			{
				"role":  "user",
				"parts": []map[string]any{{"text": ai.BuildPrompt(req)}},
			},
		},
	}
	var resp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := postJSON(ctx, g.client, endpoint, nil, body, &resp); err != nil {
		return ai.Response{}, err
	}
	raw := ""
	if len(resp.Candidates) > 0 {
		for _, p := range resp.Candidates[0].Content.Parts {
			raw += p.Text
		}
	}
	return ai.Response{
		SQL:      ai.ExtractSQL(raw),
		Raw:      raw,
		Provider: g.Name(),
	}, nil
}

// ChatStream implements ai.ChatProvider with Gemini's streamGenerateContent (SSE).
func (g *Gemini) ChatStream(ctx context.Context, req ai.ChatRequest) (<-chan ai.ChatChunk, error) {
	if g.APIKey == "" {
		return nil, errors.New("gemini: api_key not set")
	}
	endpoint := fmt.Sprintf(
		"%s/models/%s:streamGenerateContent?alt=sse&key=%s",
		g.BaseURL, g.Model, url.QueryEscape(g.APIKey),
	)
	contents := make([]map[string]any, 0, len(req.Messages))
	for _, m := range req.Messages {
		role := m.Role
		switch role {
		case "assistant":
			role = "model"
		case "system":
			continue
		}
		contents = append(contents, map[string]any{
			"role":  role,
			"parts": []map[string]any{{"text": m.Content}},
		})
	}
	body := map[string]any{
		"systemInstruction": map[string]any{
			"parts": []map[string]any{{"text": ai.ChatSystemPrompt + "\n\n" + ai.BuildChatContext(req)}},
		},
		"contents": contents,
	}
	// Gemini's SSE stream has no explicit terminator event; the body simply
	// closes, which postSSE reports as Done.
	return postSSE(ctx, endpoint, nil, body, func(data string) (string, bool, error) {
		var ev struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			return "", false, nil
		}
		text := ""
		if len(ev.Candidates) > 0 {
			for _, p := range ev.Candidates[0].Content.Parts {
				text += p.Text
			}
		}
		return text, false, nil
	})
}
