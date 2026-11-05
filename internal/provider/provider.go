package provider

import (
	"context"

	"agent-harness/Squib/internal/api"
)

// Provider abstracts the LLM backend (Groq, OpenAI, Anthropic, etc.).
type Provider interface {
	Send(ctx context.Context, messages []api.Message, tools []api.ToolDef) (*api.Message, error)
	ModelName() string
}
