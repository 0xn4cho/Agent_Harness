package main

import (
	"agent-harness/Squib/internal/api"
	"testing"
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
	if len(pruned) > 12 { // allowed leeway for boundary
		t.Errorf("expected pruned messages length to be <= 12, got %d", len(pruned))
	}

	if pruned[0].Role != "system" {
		t.Errorf("expected first message to remain system, got %s", pruned[0].Role)
	}

	// Verify no tool message is orphaned from its assistant message
	for i, m := range pruned {
		if m.Role == "tool" && i > 0 && pruned[i-1].Role != "assistant" {
			t.Errorf("tool message at index %d has no preceding assistant call", i)
		}
	}
}
