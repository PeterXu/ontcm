package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Config configures the LM Studio (OpenAI-compatible) client.
type Config struct {
	Enabled  bool          // if false, the client is not used (agent stays rule-based)
	Endpoint string        // e.g. "http://192.168.50.17:1234"
	Model    string        // e.g. "shizhengpt-7b-vl-i1"
	Timeout  time.Duration // per-request timeout
}

// DefaultConfig returns sensible defaults pointing at a local LM Studio.
func DefaultConfig() Config {
	return Config{
		Enabled:  false, // opt-in
		Endpoint: "http://192.168.50.17:1234",
		Model:    "shizhengpt-7b-vl-i1",
		Timeout:  60 * time.Second,
	}
}

// LMStudioClient calls an OpenAI-compatible /v1/chat/completions endpoint
// (LM Studio, Ollama's OpenAI compat layer, etc.).
type LMStudioClient struct {
	cfg     Config
	http    *http.Client
	baseURL string // Endpoint + "/v1"
}

// NewLMStudioClient constructs a client. If cfg.Endpoint has no scheme/host
// the client still constructed but calls will fail; callers should validate.
func NewLMStudioClient(cfg Config) *LMStudioClient {
	endpoint := strings.TrimRight(strings.TrimSpace(cfg.Endpoint), "/")
	return &LMStudioClient{
		cfg:     cfg,
		http:    &http.Client{Timeout: cfg.Timeout},
		baseURL: endpoint + "/v1",
	}
}

// chatResponse captures the fields we need from the OpenAI response.
type chatResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// Complete sends a chat-completion request and returns the assistant's reply.
//
// Network errors, non-2xx responses, and context cancellation are all reported
// as ErrLLMUnavailable so callers can fall back to rule-based logic uniformly.
func (c *LMStudioClient) Complete(ctx context.Context, req CompleteRequest) (CompleteResponse, error) {
	if req.Model == "" {
		req.Model = c.cfg.Model
	}
	body, err := json.Marshal(req)
	if err != nil {
		return CompleteResponse{}, ErrLLMUnavailable
	}

	url := c.baseURL + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return CompleteResponse{}, ErrLLMUnavailable
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.http.Do(httpReq)
	if err != nil {
		return CompleteResponse{}, ErrLLMUnavailable
	}
	defer httpResp.Body.Close()

	// Cap the response body: MaxTokens limits generated tokens, not the HTTP
	// body, so a misconfigured/proxy endpoint returning a huge payload could
	// otherwise OOM the handler goroutine. 1 MiB is far above any valid
	// short-formula-choice JSON response.
	respBody, err := io.ReadAll(io.LimitReader(httpResp.Body, 1<<20))
	if err != nil {
		return CompleteResponse{}, ErrLLMUnavailable
	}
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return CompleteResponse{}, fmt.Errorf("%w: status %d", ErrLLMUnavailable, httpResp.StatusCode)
	}

	var parsed chatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return CompleteResponse{}, ErrLLMUnavailable
	}
	if len(parsed.Choices) == 0 {
		return CompleteResponse{}, ErrLLMUnavailable
	}

	ch := parsed.Choices[0]
	return CompleteResponse{
		Content:          ch.Message.Content,
		FinishReason:     ch.FinishReason,
		PromptTokens:     parsed.Usage.PromptTokens,
		CompletionTokens: parsed.Usage.CompletionTokens,
	}, nil
}
