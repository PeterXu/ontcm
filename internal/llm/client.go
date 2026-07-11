// Package llm provides a small, provider-agnostic client for the LLM used by
// the diagnostic agent.
//
// The agent depends only on the LLMClient interface, so it can be tested
// without a running model server (use FakeClient) and can swap providers
// (LMStudioClient today, a cloud client later) without changes to the agent.
package llm

import (
	"context"
	"errors"
)

// Message is a single chat message in the OpenAI-style roles.
type Message struct {
	Role    string `json:"role"`    // "system" | "user" | "assistant"
	Content string `json:"content"`
}

// CompleteRequest is a request to generate a chat completion.
type CompleteRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

// CompleteResponse holds the completion result.
type CompleteResponse struct {
	Content          string // The assistant's message text
	FinishReason     string // "stop", "length", etc.
	PromptTokens     int
	CompletionTokens int
}

// LLMClient generates chat completions. Implementations must be safe for
// concurrent use and must honour the request context (including its deadline).
type LLMClient interface {
	Complete(ctx context.Context, req CompleteRequest) (CompleteResponse, error)
}

// ErrLLMUnavailable is returned when the model server cannot be reached or
// returns an error. Callers should treat this as a signal to fall back to
// rule-based logic rather than failing the whole operation.
var ErrLLMUnavailable = errors.New("llm: client unavailable")
