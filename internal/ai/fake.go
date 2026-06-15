package ai

import "context"

// FakeProvider is a Provider (and ChatProvider) implementation for tests and
// manual smoke checks.
type FakeProvider struct {
	NameValue string
	Reply     string
	Err       error
	Got       Request
	GotChat   ChatRequest
}

func (f *FakeProvider) Name() string {
	if f.NameValue == "" {
		return "fake"
	}
	return f.NameValue
}

func (f *FakeProvider) Generate(_ context.Context, req Request) (Response, error) {
	f.Got = req
	if f.Err != nil {
		return Response{}, f.Err
	}
	return Response{
		SQL:      ExtractSQL(f.Reply),
		Raw:      f.Reply,
		Provider: f.Name(),
	}, nil
}

// ChatStream emits Reply across two deltas to exercise the streaming path.
func (f *FakeProvider) ChatStream(_ context.Context, req ChatRequest) (<-chan ChatChunk, error) {
	f.GotChat = req
	ch := make(chan ChatChunk, 4)
	go func() {
		defer close(ch)
		if f.Err != nil {
			ch <- ChatChunk{Err: f.Err}
			return
		}
		reply := f.Reply
		if reply == "" {
			reply = "ok"
		}
		mid := len(reply) / 2
		ch <- ChatChunk{Delta: reply[:mid]}
		ch <- ChatChunk{Delta: reply[mid:]}
		ch <- ChatChunk{Done: true}
	}()
	return ch, nil
}
