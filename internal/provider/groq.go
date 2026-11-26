package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"agent-harness/Squib/internal/api"
	"agent-harness/Squib/internal/security"
)

const (
	defaultAPIURL              = "https://api.groq.com/openai/v1/chat/completions"
	maxHTTPResponseBodyBytes   = 2 * 1024 * 1024
	defaultMaxCompletionTokens = 1024
	maxRateLimitRetries        = 3
)

type GroqProvider struct {
	apiKey              string
	model               string
	apiURL              string
	httpClient          *http.Client
	maxCompletionTokens int
	reasoningEffort     string
}

type GroqOptions struct {
	MaxCompletionTokens int
	ReasoningEffort     string
}

type ToolUseFailedError struct {
	StatusCode       int
	Message          string
	FailedGeneration string
}

func (e *ToolUseFailedError) Error() string {
	return fmt.Sprintf("groq API HTTP %d tool_use_failed: %s", e.StatusCode, e.Message)
}

func NewGroq(apiKey, model string) *GroqProvider {
	return NewGroqWithOptions(apiKey, model, GroqOptions{})
}

func NewGroqWithOptions(apiKey, model string, options GroqOptions) *GroqProvider {
	if model == "" {
		model = "openai/gpt-oss-120b"
	}
	maxTokens := options.MaxCompletionTokens
	if maxTokens <= 0 {
		maxTokens = completionTokenLimit()
	}
	reasoningEffort := options.ReasoningEffort
	if reasoningEffort == "" {
		reasoningEffort = "low"
	}
	return &GroqProvider{
		apiKey:              apiKey,
		model:               model,
		apiURL:              defaultAPIURL,
		maxCompletionTokens: maxTokens,
		reasoningEffort:     reasoningEffort,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func completionTokenLimit() int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv("SQUIB_MAX_COMPLETION_TOKENS")))
	if err == nil && value > 0 {
		return value
	}
	return defaultMaxCompletionTokens
}

func (g *GroqProvider) ModelName() string {
	return g.model
}

type groqToolDef struct {
	Type     string      `json:"type"`
	Function groqFuncDef `json:"function"`
}

type groqFuncDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type groqToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type groqMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	ToolCalls  []groqToolCall `json:"tool_calls,omitempty"`
}

type groqChatRequest struct {
	Model               string        `json:"model"`
	Messages            []groqMessage `json:"messages"`
	Tools               []groqToolDef `json:"tools,omitempty"`
	Temperature         float64       `json:"temperature"`
	MaxCompletionTokens int           `json:"max_completion_tokens"`
	ReasoningEffort     string        `json:"reasoning_effort,omitempty"`
	ReasoningFormat     string        `json:"reasoning_format,omitempty"`
}

type groqChatResponse struct {
	Choices []struct {
		Message groqMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message          string `json:"message"`
		Type             string `json:"type,omitempty"`
		Code             string `json:"code,omitempty"`
		FailedGeneration string `json:"failed_generation,omitempty"`
	} `json:"error,omitempty"`
}

func (g *GroqProvider) Send(ctx context.Context, messages []api.Message, tools []api.ToolDef) (*api.Message, error) {
	var gTools []groqToolDef
	for _, t := range tools {
		gTools = append(gTools, groqToolDef{
			Type: "function",
			Function: groqFuncDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		})
	}

	var gMessages []groqMessage
	for _, m := range messages {
		var tcs []groqToolCall
		for _, tc := range m.ToolCalls {
			var item groqToolCall
			item.ID = tc.ID
			item.Type = "function"
			item.Function.Name = tc.Name
			item.Function.Arguments = tc.Arguments
			tcs = append(tcs, item)
		}
		content := m.Content
		if m.Role == "tool" && content == "" {
			content = "(command completed with no output)"
		}
		gMessages = append(gMessages, groqMessage{
			Role:       m.Role,
			Content:    content,
			ToolCallID: m.ToolCallID,
			ToolCalls:  tcs,
		})
	}

	reqBody := groqChatRequest{
		Model:               g.model,
		Messages:            gMessages,
		Tools:               gTools,
		Temperature:         0.1,
		MaxCompletionTokens: g.maxCompletionTokens,
	}
	if supportsReasoningControls(g.model) {
		reqBody.ReasoningEffort = g.reasoningEffort
		reqBody.ReasoningFormat = "hidden"
	}

	raw, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	var body []byte
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.apiURL, bytes.NewReader(raw))
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+g.apiKey)

		resp, err := g.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("groq request error: %w", err)
		}
		body, err = io.ReadAll(io.LimitReader(resp.Body, maxHTTPResponseBodyBytes))
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read response: %w", err)
		}

		if resp.StatusCode != http.StatusTooManyRequests || attempt == maxRateLimitRetries {
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				if resp.StatusCode == http.StatusBadRequest {
					var errResp groqChatResponse
					if uerr := json.Unmarshal(body, &errResp); uerr == nil && errResp.Error != nil && errResp.Error.Code == "tool_use_failed" {
						return nil, &ToolUseFailedError{
							StatusCode:       resp.StatusCode,
							Message:          security.SanitizeTerminal(errResp.Error.Message),
							FailedGeneration: security.SanitizeTerminal(security.TruncateString(errResp.Error.FailedGeneration, 250)),
						}
					}
				}
				snippet := security.SanitizeTerminal(security.TruncateString(string(body), 250))
				return nil, fmt.Errorf("groq API HTTP %d error: %s", resp.StatusCode, snippet)
			}
			break
		}

		wait := retryAfter(resp.Header.Get("Retry-After"))
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
	}

	var parsed groqChatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		snippet := security.SanitizeTerminal(security.TruncateString(string(body), 250))
		return nil, fmt.Errorf("unmarshal response: %w (body snippet: %s)", err, snippet)
	}

	if parsed.Error != nil {
		return nil, fmt.Errorf("groq API error: %s", parsed.Error.Message)
	}

	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("no response choices from Groq")
	}

	choiceMsg := parsed.Choices[0].Message
	result := &api.Message{
		Role:    choiceMsg.Role,
		Content: choiceMsg.Content,
	}
	for _, tc := range choiceMsg.ToolCalls {
		result.ToolCalls = append(result.ToolCalls, api.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	return result, nil
}

func retryAfter(header string) time.Duration {
	seconds, err := strconv.ParseFloat(strings.TrimSpace(header), 64)
	if err == nil && seconds > 0 {
		return time.Duration(seconds * float64(time.Second))
	}
	return time.Second
}

func supportsReasoningControls(model string) bool {
	return strings.HasPrefix(model, "openai/gpt-oss-") || strings.HasPrefix(model, "qwen/qwen3")
}
