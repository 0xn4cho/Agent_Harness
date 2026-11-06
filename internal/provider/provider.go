package provider

import (
	"context"

	"agent-harness/Squib/internal/api"
)

type Provider interface {
	Send(ctx context.Context, messages []api.Message, tools []api.ToolDef) (*api.Message, error)
	ModelName() string
}
