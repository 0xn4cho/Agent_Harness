package provider

import (
	"context"
	"testing"

	"agent-harness/Squib/internal/api"
)

func TestMockProviderScriptedResponsesAndCallTracking(t *testing.T) {
	scripted := []api.Message{
		{
			Role: "assistant",
			ToolCalls: []api.ToolCall{
				{ID: "call_1", Name: "read_file", Arguments: `{"path":"main.go"}`},
			},
		},
		{
			Role:    "assistant",
			Content: "File inspected successfully.",
		},
	}

	mock := NewMock(scripted...)
	if mock.ModelName() != "mock-model" {
		t.Fatalf("expected model name 'mock-model', got %q", mock.ModelName())
	}

	ctx := context.Background()
	msgs := []api.Message{
		{Role: "system", Content: "system prompt"},
		{Role: "user", Content: "inspect main.go"},
	}
	tools := []api.ToolDef{
		{Name: "read_file", Description: "read a file"},
	}

	resp1, err := mock.Send(ctx, msgs, tools)
	if err != nil {
		t.Fatalf("unexpected error on call 1: %v", err)
	}
	if len(resp1.ToolCalls) != 1 || resp1.ToolCalls[0].Name != "read_file" {
		t.Fatalf("expected read_file tool call, got %+v", resp1)
	}

	msgs[1].Content = "mutated"

	msgs = append(msgs, *resp1, api.Message{
		Role:       "tool",
		ToolCallID: "call_1",
		Content:    "package main",
	})
	resp2, err := mock.Send(ctx, msgs, tools)
	if err != nil {
		t.Fatalf("unexpected error on call 2: %v", err)
	}
	if resp2.Content != "File inspected successfully." {
		t.Fatalf("unexpected resp2 content: %q", resp2.Content)
	}

	if len(mock.Calls) != 2 {
		t.Fatalf("expected 2 recorded calls, got %d", len(mock.Calls))
	}
	if mock.Calls[0].Messages[1].Content != "inspect main.go" {
		t.Errorf("recorded call was mutated: got %q", mock.Calls[0].Messages[1].Content)
	}
}

func TestMockProviderInteractiveFallback(t *testing.T) {
	mock := NewMock()
	ctx := context.Background()

	resp, err := mock.Send(ctx, []api.Message{
		{Role: "user", Content: "read go.mod"},
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "read_file" {
		t.Fatalf("expected interactive fallback to emit read_file tool call, got %+v", resp)
	}
}
