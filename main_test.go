package main

import (
	"bufio"
	"context"
	"strings"
	"testing"

	"agent-harness/Squib/internal/api"
	"agent-harness/Squib/internal/permission"
	"agent-harness/Squib/internal/provider"
	"agent-harness/Squib/internal/tool"
)

func TestPruneMessages(t *testing.T) {
	system := api.Message{Role: "system", Content: "sys"}

	msgs := []api.Message{system}
	for i := 0; i < 30; i++ {
		msgs = append(msgs, api.Message{
			Role:    "user",
			Content: "hello",
		})
		msgs = append(msgs, api.Message{
			Role:    "assistant",
			Content: "hi",
		})
	}

	pruned := pruneMessages(msgs, 10)
	if len(pruned) > 12 {
		t.Errorf("expected pruned messages length to be <= 12, got %d", len(pruned))
	}

	if pruned[0].Role != "system" {
		t.Errorf("expected first message to remain system, got %s", pruned[0].Role)
	}

	for i, m := range pruned {
		if m.Role == "tool" && i > 0 && pruned[i-1].Role != "assistant" && pruned[i-1].Role != "tool" {
			t.Errorf("tool message at index %d has no preceding assistant/tool call", i)
		}
	}
}

func TestPruneMessagesByTokensProtectsToolSequence(t *testing.T) {
	heavy := strings.Repeat("token ", 500)
	msgs := []api.Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "old prompt " + heavy},
		{
			Role: "assistant",
			ToolCalls: []api.ToolCall{
				{ID: "call_old", Name: "read_file", Arguments: `{"path":"go.mod"}`},
			},
		},
		{Role: "tool", ToolCallID: "call_old", Content: heavy},
		{Role: "assistant", Content: "old done"},
		{Role: "user", Content: "recent question"},
		{Role: "assistant", Content: "recent answer"},
	}

	pruned := pruneMessagesByTokens(msgs, 25, 300)
	if len(pruned) != 3 {
		t.Fatalf("expected 3 messages after token compaction, got %d: %+v", len(pruned), pruned)
	}
	if pruned[0].Role != "system" || pruned[1].Content != "recent question" {
		t.Errorf("unexpected pruned slice: %+v", pruned)
	}
}

func TestAgentLoopWithMockProvider(t *testing.T) {
	mock := provider.NewMock(
		api.Message{
			Role: "assistant",
			ToolCalls: []api.ToolCall{
				{ID: "tc_1", Name: "read_file", Arguments: `{"path":"go.mod"}`},
			},
		},
		api.Message{
			Role:    "assistant",
			Content: "El módulo es agent-harness/Squib.",
		},
	)

	reg := tool.NewRegistry()
	reg.Register(tool.NewReadFileTool())
	reg.Register(tool.NewWriteFileTool())
	reg.Register(tool.NewBashTool())
	reg.Register(tool.NewDevOpsTool())

	gate := permission.NewGate(bufio.NewScanner(strings.NewReader("y\n")))

	initial := []api.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: "¿Cuál es el nombre del módulo en go.mod?"},
	}

	finalMsgs := agentLoop(context.Background(), mock, reg, gate, initial)

	if len(mock.Calls) != 2 {
		t.Fatalf("expected MockProvider to be called 2 times, got %d", len(mock.Calls))
	}

	gotToolNames := make([]string, len(mock.Calls[0].Tools))
	for i, td := range mock.Calls[0].Tools {
		gotToolNames[i] = td.Name
	}
	wantOrder := "bash,devops_status,read_file,write_file"
	if strings.Join(gotToolNames, ",") != wantOrder {
		t.Errorf("tools passed to provider = %v, want %s", gotToolNames, wantOrder)
	}

	lastMsg := finalMsgs[len(finalMsgs)-1]
	if lastMsg.Role != "assistant" || !strings.Contains(lastMsg.Content, "agent-harness/Squib") {
		t.Errorf("unexpected final message: %+v", lastMsg)
	}
}
