package provider

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"agent-harness/Squib/internal/api"
)

func TestNewGroqWithOptionsSendsReasoningAndMaxTokens(t *testing.T) {
	var received groqChatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer srv.Close()

	g := NewGroqWithOptions("test-key", "openai/gpt-oss-20b", GroqOptions{
		MaxCompletionTokens: 500,
		ReasoningEffort:     "medium",
	})
	g.apiURL = srv.URL

	resp, err := g.Send(context.Background(), []api.Message{{Role: "user", Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if resp.Content != "ok" {
		t.Errorf("resp.Content = %q, want %q", resp.Content, "ok")
	}
	if received.Model != "openai/gpt-oss-20b" {
		t.Errorf("received.Model = %q, want openai/gpt-oss-20b", received.Model)
	}
	if received.MaxCompletionTokens != 500 {
		t.Errorf("received.MaxCompletionTokens = %d, want 500", received.MaxCompletionTokens)
	}
	if received.ReasoningEffort != "medium" || received.ReasoningFormat != "hidden" {
		t.Errorf("unexpected reasoning fields: effort=%q format=%q", received.ReasoningEffort, received.ReasoningFormat)
	}
}

func TestGroqOmitsReasoningForNonReasoningModelsAndUsesEnvFallback(t *testing.T) {
	t.Setenv("SQUIB_MAX_COMPLETION_TOKENS", "777")

	var received map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"done"}}]}`))
	}))
	defer srv.Close()

	g := NewGroqWithOptions("test-key", "llama-3.3-70b-versatile", GroqOptions{})
	g.apiURL = srv.URL

	if _, err := g.Send(context.Background(), []api.Message{{Role: "user", Content: "hi"}}, nil); err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if got := int(received["max_completion_tokens"].(float64)); got != 777 {
		t.Errorf("max_completion_tokens = %d, want 777 from env fallback", got)
	}
	if _, ok := received["reasoning_effort"]; ok {
		t.Errorf("expected reasoning_effort to be omitted for non-reasoning model")
	}
	if _, ok := received["reasoning_format"]; ok {
		t.Errorf("expected reasoning_format to be omitted for non-reasoning model")
	}
}

func TestGroqRetriesOn429RateLimit(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n == 1 {
			w.Header().Set("Retry-After", "0.01")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"message":"rate limit"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"recovered"}}]}`))
	}))
	defer srv.Close()

	g := NewGroq("test-key", "openai/gpt-oss-20b")
	g.apiURL = srv.URL

	resp, err := g.Send(context.Background(), []api.Message{{Role: "user", Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("Send failed after retry: %v", err)
	}
	if resp.Content != "recovered" {
		t.Errorf("resp.Content = %q, want recovered", resp.Content)
	}
	if attempts.Load() != 2 {
		t.Errorf("attempts = %d, want 2", attempts.Load())
	}
}

func TestGroqReturnsToolUseFailedError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"Tool call validation failed: attempted to call tool 'write_file' which was not in request.tools","type":"invalid_request_error","code":"tool_use_failed","failed_generation":"{\"name\": \"write_file\"}"}}`))
	}))
	defer srv.Close()

	g := NewGroq("test-key", "openai/gpt-oss-20b")
	g.apiURL = srv.URL

	_, err := g.Send(context.Background(), []api.Message{{Role: "user", Content: "add note"}}, nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	var toolErr *ToolUseFailedError
	if !errors.As(err, &toolErr) {
		t.Fatalf("expected *ToolUseFailedError, got %T (%v)", err, err)
	}
	if toolErr.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want 400", toolErr.StatusCode)
	}
	if !strings.Contains(toolErr.Message, "not in request.tools") {
		t.Errorf("unexpected Message: %q", toolErr.Message)
	}
	if !strings.Contains(toolErr.FailedGeneration, "write_file") {
		t.Errorf("unexpected FailedGeneration: %q", toolErr.FailedGeneration)
	}
}
