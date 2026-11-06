package tool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"agent-harness/Squib/internal/api"
	"agent-harness/Squib/internal/security"
)

const maxWriteFileBytes = 1024 * 1024

type WriteFileTool struct{}

func NewWriteFileTool() *WriteFileTool {
	return &WriteFileTool{}
}

func (w *WriteFileTool) Definition() api.ToolDef {
	return api.ToolDef{
		Name:        "write_file",
		Description: "Write content to a file at the specified path within the workspace (up to 1MB).",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Path to the file to write (must be within workspace).",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "The text content to write into the file.",
				},
			},
			"required": []string{"path", "content"},
		},
	}
}

func (w *WriteFileTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("missing or invalid 'path' argument")
	}
	content, ok := args["content"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'content' argument")
	}

	if len(content) > maxWriteFileBytes {
		return "", fmt.Errorf("content size (%d bytes) exceeds maximum limit of %d bytes (1MB)", len(content), maxWriteFileBytes)
	}

	safePath, err := security.ValidateSafePath(path)
	if err != nil {
		return "", err
	}

	dir := filepath.Dir(safePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	err = os.WriteFile(safePath, []byte(content), 0644)
	if err != nil {
		return "", err
	}
	return "wrote " + path, nil
}
