package compact

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"agent-harness/Squib/internal/api"
	"agent-harness/Squib/internal/provider"
)

const (
	DefaultMaxTokens   = 12000
	DefaultMaxMessages = 25
)

type CompactionStrategy interface {
	Compact(ctx context.Context, messages []api.Message) ([]api.Message, error)
}

func EstimateMessageTokens(m api.Message) int {
	tokens := 4
	tokens += (len(m.Content) + 3) / 4
	if m.ToolCallID != "" {
		tokens += (len(m.ToolCallID) + 3) / 4
	}
	for _, tc := range m.ToolCalls {
		tokens += 4
		tokens += (len(tc.ID) + len(tc.Name) + len(tc.Arguments) + 3) / 4
	}
	return tokens
}

func EstimateTokens(messages []api.Message) int {
	total := 0
	for _, m := range messages {
		total += EstimateMessageTokens(m)
	}
	return total
}

func SafeSplitPoint(messages []api.Message, targetIdx int) int {
	minIdx := 0
	if len(messages) > 0 && messages[0].Role == "system" {
		minIdx = 1
	}
	if targetIdx <= minIdx {
		return minIdx
	}
	if targetIdx >= len(messages) {
		return len(messages)
	}

	for i := targetIdx; i >= minIdx; i-- {
		if isCleanSplitBoundary(messages, i) && messages[i].Role == "user" {
			return i
		}
	}

	for i := targetIdx + 1; i < len(messages); i++ {
		if isCleanSplitBoundary(messages, i) && messages[i].Role == "user" {
			return i
		}
	}

	return minIdx
}

func SafeSplitPointForBudget(messages []api.Message, minTargetIdx int) int {
	minIdx := 0
	if len(messages) > 0 && messages[0].Role == "system" {
		minIdx = 1
	}
	if minTargetIdx < minIdx {
		minTargetIdx = minIdx
	}

	for i := minTargetIdx; i < len(messages); i++ {
		if isCleanSplitBoundary(messages, i) && messages[i].Role == "user" {
			return i
		}
	}

	for i := minTargetIdx - 1; i >= minIdx; i-- {
		if isCleanSplitBoundary(messages, i) && messages[i].Role == "user" {
			return i
		}
	}

	return minIdx
}

func isCleanSplitBoundary(messages []api.Message, i int) bool {
	if i <= 0 || i >= len(messages) {
		return true
	}
	if messages[i].Role == "tool" {
		return false
	}
	if messages[i-1].Role == "assistant" && len(messages[i-1].ToolCalls) > 0 {
		return false
	}
	return true
}

type NoCompaction struct{}

func (NoCompaction) Compact(_ context.Context, messages []api.Message) ([]api.Message, error) {
	return messages, nil
}

type SlidingWindow struct {
	MaxTokens   int
	MaxMessages int
}

func NewSlidingWindow(maxTokens, maxMessages int) *SlidingWindow {
	if maxTokens <= 0 {
		maxTokens = DefaultMaxTokens
	}
	if maxMessages <= 0 {
		maxMessages = DefaultMaxMessages
	}
	return &SlidingWindow{
		MaxTokens:   maxTokens,
		MaxMessages: maxMessages,
	}
}

func (s *SlidingWindow) Compact(_ context.Context, messages []api.Message) ([]api.Message, error) {
	maxTokens := s.MaxTokens
	if maxTokens <= 0 {
		maxTokens = DefaultMaxTokens
	}
	maxMsgs := s.MaxMessages
	if maxMsgs <= 0 {
		maxMsgs = DefaultMaxMessages
	}

	if len(messages) == 0 {
		return messages, nil
	}

	totalTokens := EstimateTokens(messages)
	if len(messages) <= maxMsgs && totalTokens <= maxTokens {
		return messages, nil
	}

	hasSystem := messages[0].Role == "system"
	minIdx := 0
	systemTokens := 0
	if hasSystem {
		minIdx = 1
		systemTokens = EstimateMessageTokens(messages[0])
	}

	msgTargetIdx := minIdx
	if len(messages) > maxMsgs {
		msgTargetIdx = len(messages) - maxMsgs
		if hasSystem {
			msgTargetIdx = len(messages) - (maxMsgs - 1)
		}
		if msgTargetIdx < minIdx {
			msgTargetIdx = minIdx
		}
	}

	tokenBudget := maxTokens - systemTokens
	if tokenBudget < 256 {
		tokenBudget = 256
	}

	tokTargetIdx := len(messages)
	runningTokens := 0
	for i := len(messages) - 1; i >= minIdx; i-- {
		msgTok := EstimateMessageTokens(messages[i])
		if runningTokens+msgTok > tokenBudget && tokTargetIdx < len(messages) {
			break
		}
		runningTokens += msgTok
		tokTargetIdx = i
	}

	targetIdx := msgTargetIdx
	if tokTargetIdx > targetIdx {
		targetIdx = tokTargetIdx
	}

	splitIdx := SafeSplitPointForBudget(messages, targetIdx)
	if splitIdx <= minIdx {
		return messages, nil
	}

	compacted := make([]api.Message, 0, (minIdx)+len(messages[splitIdx:]))
	if hasSystem {
		compacted = append(compacted, messages[0])
	}
	compacted = append(compacted, messages[splitIdx:]...)
	return compacted, nil
}

type Summarize struct {
	Provider   provider.Provider
	MaxTokens  int
	Threshold  int
	KeepRecent int
}

func (s *Summarize) Compact(ctx context.Context, messages []api.Message) ([]api.Message, error) {
	threshold := s.Threshold
	if threshold <= 0 {
		threshold = DefaultMaxMessages
	}
	maxTokens := s.MaxTokens
	if maxTokens <= 0 {
		maxTokens = DefaultMaxTokens
	}
	keepRecent := s.KeepRecent
	if keepRecent <= 0 {
		keepRecent = 6
	}

	if len(messages) <= threshold && EstimateTokens(messages) <= maxTokens {
		return messages, nil
	}

	hasSystem := len(messages) > 0 && messages[0].Role == "system"
	minIdx := 0
	if hasSystem {
		minIdx = 1
	}

	targetIdx := len(messages) - keepRecent
	if targetIdx <= minIdx {
		return messages, nil
	}

	splitIdx := SafeSplitPoint(messages, targetIdx)
	if splitIdx <= minIdx || splitIdx >= len(messages) {
		return messages, nil
	}

	toSummarize := messages[minIdx:splitIdx]
	recent := messages[splitIdx:]

	summaryText := s.buildSummary(ctx, toSummarize)
	summaryMsg := api.Message{
		Role:    "user",
		Content: fmt.Sprintf("[Conversation history compacted (%d earlier messages summarized)]\n%s", len(toSummarize), summaryText),
	}

	out := make([]api.Message, 0, minIdx+1+len(recent))
	if hasSystem {
		out = append(out, messages[0])
	}
	out = append(out, summaryMsg)
	out = append(out, recent...)
	return out, nil
}

func (s *Summarize) buildSummary(ctx context.Context, msgs []api.Message) string {
	var transcript strings.Builder
	for _, m := range msgs {
		if m.Content != "" {
			snippet := m.Content
			if len(snippet) > 300 {
				snippet = snippet[:300] + "..."
			}
			transcript.WriteString(fmt.Sprintf("- [%s]: %s\n", m.Role, snippet))
		}
		for _, tc := range m.ToolCalls {
			transcript.WriteString(fmt.Sprintf("- [tool_use %s]: %s\n", tc.Name, tc.Arguments))
		}
	}

	if s.Provider != nil {
		prompt := []api.Message{
			{
				Role:    "system",
				Content: "Summarize the following coding agent conversation history concisely in bullet points, preserving key file paths, commands run, and decisions made.",
			},
			{
				Role:    "user",
				Content: transcript.String(),
			},
		}
		if resp, err := s.Provider.Send(ctx, prompt, nil); err == nil && resp != nil && strings.TrimSpace(resp.Content) != "" {
			return strings.TrimSpace(resp.Content)
		}
	}

	return strings.TrimSpace(transcript.String())
}

type loggingDecorator struct {
	inner   CompactionStrategy
	logPath string
}

func WithLogging(inner CompactionStrategy, logPath string) CompactionStrategy {
	return &loggingDecorator{
		inner:   inner,
		logPath: logPath,
	}
}

func (l *loggingDecorator) Compact(ctx context.Context, messages []api.Message) ([]api.Message, error) {
	beforeCount := len(messages)
	beforeTokens := EstimateTokens(messages)

	compacted, err := l.inner.Compact(ctx, messages)
	if err != nil {
		return compacted, err
	}

	afterCount := len(compacted)
	afterTokens := EstimateTokens(compacted)
	if afterCount != beforeCount || afterTokens != beforeTokens {
		if f, ferr := os.OpenFile(l.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); ferr == nil {
			_, _ = fmt.Fprintf(f, "%s | messages: %d -> %d | est_tokens: %d -> %d\n",
				time.Now().Format(time.RFC3339), beforeCount, afterCount, beforeTokens, afterTokens)
			_ = f.Close()
		}
	}
	return compacted, nil
}
