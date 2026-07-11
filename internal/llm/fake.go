package llm

import (
	"context"
	"fmt"
	"strings"
)

// FakeClient is a test double for LLMClient. It returns a canned response
// based on the last user message: if a registered handler matches the prompt,
// its response is returned; otherwise DefaultContent. If Fail is set, every
// call returns ErrLLMUnavailable — useful for testing fallback paths.
type FakeClient struct {
	// Handler inspects the request's user message and returns the canned
	// completion content plus an error. If nil, DefaultContent is returned.
	Handler       func(userMessage string) (string, error)
	DefaultContent string
	Fail          bool
	// Calls records the requests made, for assertions.
	Calls []CompleteRequest
}

// Complete satisfies LLMClient.
func (f *FakeClient) Complete(ctx context.Context, req CompleteRequest) (CompleteResponse, error) {
	f.Calls = append(f.Calls, req)
	if f.Fail {
		return CompleteResponse{}, ErrLLMUnavailable
	}
	if f.Handler != nil {
		// Find the last user message.
		user := ""
		for _, m := range req.Messages {
			if m.Role == "user" {
				user = m.Content
			}
		}
		content, err := f.Handler(user)
		if err != nil {
			return CompleteResponse{}, err
		}
		return CompleteResponse{Content: content, FinishReason: "stop"}, nil
	}
	return CompleteResponse{Content: f.DefaultContent, FinishReason: "stop"}, nil
}

// MustParseFormulaID is a test helper that extracts a formula_id JSON field
// from a string, failing the test expectation simply by returning the raw
// value if parsing fails. Used to build FakeClient handlers.
func MustParseFormulaID(s, id string) string {
	return fmt.Sprintf(`{"formula_id":"%s","reason":"%s"}`, id, strings.TrimSpace(s))
}
