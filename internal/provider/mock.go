package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"agent-harness/Squib/internal/api"
)

type MockCall struct {
	Messages []api.Message
	Tools    []api.ToolDef
}

type MockProvider struct {
	mu        sync.Mutex
	model     string
	Responses []api.Message
	Calls     []MockCall
	Err       error
	SendFunc  func(ctx context.Context, messages []api.Message, tools []api.ToolDef) (*api.Message, error)
	callIndex int
}

func NewMock(responses ...api.Message) *MockProvider {
	return &MockProvider{
		model:     "mock-model",
		Responses: responses,
	}
}

func NewMockProvider(responses ...api.Message) *MockProvider {
	return NewMock(responses...)
}

func (m *MockProvider) ModelName() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.model == "" {
		return "mock-model"
	}
	return m.model
}

func (m *MockProvider) SetModel(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.model = name
}

func (m *MockProvider) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Calls = nil
	m.callIndex = 0
}

func (m *MockProvider) Send(ctx context.Context, messages []api.Message, tools []api.ToolDef) (*api.Message, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	copiedMsgs := make([]api.Message, len(messages))
	for i, msg := range messages {
		copiedMsgs[i] = msg
		if len(msg.ToolCalls) > 0 {
			copiedMsgs[i].ToolCalls = append([]api.ToolCall(nil), msg.ToolCalls...)
		}
	}
	copiedTools := append([]api.ToolDef(nil), tools...)

	m.Calls = append(m.Calls, MockCall{
		Messages: copiedMsgs,
		Tools:    copiedTools,
	})

	if m.Err != nil {
		return nil, m.Err
	}

	if m.SendFunc != nil {
		return m.SendFunc(ctx, copiedMsgs, copiedTools)
	}

	if m.callIndex < len(m.Responses) {
		resp := m.Responses[m.callIndex]
		m.callIndex++
		if resp.Role == "" {
			resp.Role = "assistant"
		}
		out := resp
		if len(resp.ToolCalls) > 0 {
			out.ToolCalls = append([]api.ToolCall(nil), resp.ToolCalls...)
		}
		return &out, nil
	}

	return m.interactiveFallback(copiedMsgs, len(m.Calls)), nil
}

func (m *MockProvider) interactiveFallback(messages []api.Message, callNum int) *api.Message {
	if len(messages) == 0 {
		return &api.Message{
			Role:    "assistant",
			Content: "[mock] Ready.",
		}
	}

	last := messages[len(messages)-1]

	if last.Role == "tool" {
		return &api.Message{
			Role:    "assistant",
			Content: fmt.Sprintf("[mock] Tool execution finished. Result:\n%s", last.Content),
		}
	}

	lower := strings.ToLower(strings.TrimSpace(last.Content))
	switch {
	case strings.HasPrefix(lower, "read "):
		path := strings.TrimSpace(last.Content[5:])
		args, _ := json.Marshal(map[string]string{"path": path})
		return &api.Message{
			Role: "assistant",
			ToolCalls: []api.ToolCall{
				{
					ID:        fmt.Sprintf("mock_call_%d", callNum),
					Name:      "read_file",
					Arguments: string(args),
				},
			},
		}
	case strings.HasPrefix(lower, "run ") || strings.HasPrefix(lower, "bash "):
		parts := strings.SplitN(last.Content, " ", 2)
		cmd := ""
		if len(parts) == 2 {
			cmd = strings.TrimSpace(parts[1])
		}
		args, _ := json.Marshal(map[string]string{"command": cmd})
		return &api.Message{
			Role: "assistant",
			ToolCalls: []api.ToolCall{
				{
					ID:        fmt.Sprintf("mock_call_%d", callNum),
					Name:      "bash",
					Arguments: string(args),
				},
			},
		}
	case strings.Contains(lower, "devops") || strings.Contains(lower, "docker") || strings.Contains(lower, "k8s"):
		target := "all"
		if strings.Contains(lower, "docker") && !strings.Contains(lower, "k8s") {
			target = "docker"
		} else if strings.Contains(lower, "k8s") && !strings.Contains(lower, "docker") {
			target = "k8s"
		}
		args, _ := json.Marshal(map[string]string{"target": target})
		return &api.Message{
			Role: "assistant",
			ToolCalls: []api.ToolCall{
				{
					ID:        fmt.Sprintf("mock_call_%d", callNum),
					Name:      "devops_status",
					Arguments: string(args),
				},
			},
		}
	default:
		return &api.Message{
			Role:    "assistant",
			Content: fmt.Sprintf("[mock] Received: %q (tip: try 'read main.go', 'run git status', or 'devops')", last.Content),
		}
	}
}
