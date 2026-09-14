package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/tardanoir/seshat/internal/ai"
)

const defaultTimeout = 60 * time.Second

func newClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &http.Client{Timeout: timeout}
}

// streamClient has no overall timeout — a streamed response can run far longer
// than a single request, so cancellation is driven entirely by ctx.
var streamClient = &http.Client{}

// postSSE opens a streaming POST and returns a channel of ai.ChatChunk. Each
// SSE "data:" payload is handed to parse, which returns the text delta and
// whether the stream is finished. The request and body scan run in a goroutine;
// the channel closes when the stream ends, ctx is cancelled, or an error occurs.
func postSSE(ctx context.Context, url string, headers map[string]string, body any, parse func(data string) (delta string, done bool, err error)) (<-chan ai.ChatChunk, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := streamClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, truncate(string(b), 512))
	}

	out := make(chan ai.ChatChunk)
	go func() {
		defer close(out)
		defer resp.Body.Close()
		send := func(c ai.ChatChunk) bool {
			select {
			case out <- c:
				return true
			case <-ctx.Done():
				return false
			}
		}
		sc := bufio.NewScanner(resp.Body)
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || !strings.HasPrefix(line, "data:") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			delta, done, perr := parse(data)
			if perr != nil {
				send(ai.ChatChunk{Err: perr})
				return
			}
			if delta != "" && !send(ai.ChatChunk{Delta: delta}) {
				return
			}
			if done {
				send(ai.ChatChunk{Done: true})
				return
			}
		}
		if err := sc.Err(); err != nil && ctx.Err() == nil {
			send(ai.ChatChunk{Err: err})
			return
		}
		send(ai.ChatChunk{Done: true})
	}()
	return out, nil
}

// postJSON marshals body, sends a POST with the given headers, and decodes the
// response into out. Non-2xx responses are returned as errors with the body
// included for diagnostics.
func postJSON(ctx context.Context, client *http.Client, url string, headers map[string]string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http %d: %s", resp.StatusCode, truncate(string(respBody), 512))
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
