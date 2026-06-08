package ai

import "context"

// FakeProvider is a Provider implementation for tests and manual smoke checks.
type FakeProvider struct {
	NameValue string
	Reply     string
	Err       error
	Got       Request
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
