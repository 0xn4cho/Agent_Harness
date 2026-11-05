package tool

import (
	"context"
	"fmt"
	"io"
	"os"

	"agent-harness/Squib/internal/api"
	"agent-harness/Squib/internal/security"
)

const maxReadFileBytes = 256 * 1024 // 256 KiB limit

type ReadFileTool struct{}

func NewReadFileTool() *ReadFileTool {
	return &ReadFileTool{}
}

func (r *ReadFileTool) Definition() api.ToolDef {
	return api.ToolDef{
		Name:        "read_file",
		Description: "Read the content of a file within the workspace (up to 256KB). Path must be relative to the workspace or within it.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Path to the file to read (must be within workspace).",
				},
			},
			"required": []string{"path"},
		},
	}
}

func (r *ReadFileTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("missing or invalid 'path' argument")
	}

	safePath, err := security.ValidateSafePath(path)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(safePath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("path %q is a directory, not a file", path)
	}

	f, err := os.Open(safePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Read with size limit to prevent memory exhaustion
	limitReader := io.LimitReader(f, maxReadFileBytes+1)
	data, err := io.ReadAll(limitReader)
	if err != nil {
		return "", fmt.Errorf("failed reading file: %w", err)
	}

	if len(data) > maxReadFileBytes {
		return string(data[:maxReadFileBytes]) + "\n... [file truncated: exceeded 256KB read limit]", nil
	}

	return string(data), nil
}
