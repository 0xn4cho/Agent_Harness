package compact

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-harness/Squib/internal/api"
	"agent-harness/Squib/internal/provider"
)

func TestSafeSplitPointProtectsToolBoundaries(t *testing.T) {
	msgs := []api.Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "turn 1"},
		{
			Role: "assistant",
			ToolCalls: []api.ToolCall{
				{ID: "tc1", Name: "read_file", Arguments: `{"path":"a.go"}`},
				{ID: "tc2", Name: "read_file", Arguments: `{"path":"b.go"}`},
			},
		},
		{Role: "tool", ToolCallID: "tc1", Content: "content a"},
		{Role: "tool", ToolCallID: "tc2", Content: "content b"},
		{Role: "assistant", Content: "done turn 1"},
		{Role: "user", Content: "turn 2"},
		{Role: "assistant", Content: "done turn 2"},
	}

	for target := 2; target <= 5; target++ {
		split := SafeSplitPoint(msgs, target)
		if msgs[split].Role == "tool" {
			t.Errorf("SafeSplitPoint(msgs, %d) = %d landed on a tool message!", target, split)
		}
		if split != 1 {
			t.Errorf("SafeSplitPoint(msgs, %d) = %d, expected backward walk to user turn at index 1", target, split)
		}

		budgetSplit := SafeSplitPointForBudget(msgs, target)
		if msgs[budgetSplit].Role == "tool" {
			t.Errorf("SafeSplitPointForBudget(msgs, %d) = %d landed on a tool message!", target, budgetSplit)
		}
		if budgetSplit != 6 {
			t.Errorf("SafeSplitPointForBudget(msgs, %d) = %d, expected forward advance to user turn at index 6", target, budgetSplit)
		}
	}
}

func TestSlidingWindowTokenCompaction(t *testing.T) {
	largeText := strings.Repeat("abcd", 1000)
	msgs := []api.Message{
		{Role: "system", Content: "system"},
		{Role: "user", Content: "first request " + largeText},
		{
			Role: "assistant",
			ToolCalls: []api.ToolCall{
				{ID: "tc1", Name: "read_file", Arguments: `{"path":"large.txt"}`},
			},
		},
		{Role: "tool", ToolCallID: "tc1", Content: largeText},
		{Role: "assistant", Content: "first answer " + largeText},
		{Role: "user", Content: "second short request"},
		{Role: "assistant", Content: "second short answer"},
	}

	sw := NewSlidingWindow(500, 25)
	compacted, err := sw.Compact(context.Background(), msgs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(compacted) != 3 {
		t.Fatalf("expected 3 messages (system + last user + last assistant), got %d", len(compacted))
	}
	if compacted[0].Role != "system" || compacted[1].Content != "second short request" {
		t.Errorf("unexpected compacted messages: %+v", compacted)
	}
	if EstimateTokens(compacted) > 500 {
		t.Errorf("expected compacted tokens <= 500, got %d", EstimateTokens(compacted))
	}
}

func TestSummarizeStrategyWithMockProvider(t *testing.T) {
	mock := provider.NewMock(api.Message{
		Role:    "assistant",
		Content: "• Inspected main.go and verified build.",
	})

	summarizer := &Summarize{
		Provider:   mock,
		MaxTokens:  100,
		Threshold:  4,
		KeepRecent: 2,
	}

	msgs := []api.Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "read main.go"},
		{
			Role:      "assistant",
			ToolCalls: []api.ToolCall{{ID: "1", Name: "read_file", Arguments: `{"path":"main.go"}`}},
		},
		{Role: "tool", ToolCallID: "1", Content: "package main"},
		{Role: "assistant", Content: "main.go read."},
		{Role: "user", Content: "what next?"},
		{Role: "assistant", Content: "all good."},
	}

	tmpLog := filepath.Join(t.TempDir(), "compact.log")
	logged := WithLogging(summarizer, tmpLog)

	compacted, err := logged.Compact(context.Background(), msgs)
	if err != nil {
		t.Fatalf("Summarize.Compact failed: %v", err)
	}

	if len(compacted) != 4 {
		t.Fatalf("expected 4 messages after summarize compaction, got %d: %+v", len(compacted), compacted)
	}
	if !strings.Contains(compacted[1].Content, "Inspected main.go") {
		t.Errorf("expected summary message to include mock summary, got: %q", compacted[1].Content)
	}

	logBytes, err := os.ReadFile(tmpLog)
	if err != nil || !strings.Contains(string(logBytes), "messages: 7 -> 4") {
		t.Errorf("expected compaction log entry, got: %q (err=%v)", string(logBytes), err)
	}
}
