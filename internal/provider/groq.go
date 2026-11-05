package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"agent-harness/Squib/internal/api"
	"agent-harness/Squib/internal/security"
)

const (
	defaultAPIURL            = "https://api.groq.com/openai/v1/chat/completions"
	maxHTTPResponseBodyBytes = 2 * 1024 * 1024 // 2 MiB limit
)

type GroqProvider struct {
	apiKey     string
	model      string
	apiURL     string
	httpClient *http.Client
}

func NewGroq(apiKey, model string) *GroqProvider {
	if model == "" {
		model = "openai/gpt-oss-120b"
	}
	return &GroqProvider{
		apiKey: apiKey,
		model:  model,
		apiURL: defaultAPIURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
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
	Model       string          `json:"model"`
	Messages    []groqMessage   `json:"messages"`
	Tools       []groqToolDef   `json:"tools,omitempty"`
	Temperature float64         `json:"temperature"`
}

type groqChatResponse struct {
	Choices []struct {
		Message groqMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
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
		// Groq requires 'content' on tool messages; an empty string would be
		// dropped by omitempty, so send a placeholder instead.
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
		Model:       g.model,
		Messages:    gMessages,
		Tools:       gTools,
		Temperature: 0.1,
	}

	raw, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

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
	defer resp.Body.Close()

	// Limit response body reading to maxHTTPResponseBodyBytes to prevent memory exhaustion
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxHTTPResponseBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet := security.SanitizeTerminal(security.TruncateString(string(body), 250))
		return nil, fmt.Errorf("groq API HTTP %d error: %s", resp.StatusCode, snippet)
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
