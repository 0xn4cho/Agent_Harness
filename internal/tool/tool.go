package tool

import (
	"context"

	"agent-harness/Squib/internal/api"
)

type Tool interface {
	Definition() api.ToolDef
	Execute(ctx context.Context, args map[string]any) (string, error)
}

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

func (r *Registry) Register(t Tool) {
	r.tools[t.Definition().Name] = t
}

func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

func (r *Registry) Definitions() []api.ToolDef {
	var defs []api.ToolDef
	for _, t := range r.tools {
		defs = append(defs, t.Definition())
	}
	return defs
}
