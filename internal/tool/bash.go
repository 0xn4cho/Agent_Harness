package tool

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"time"

	"agent-harness/Squib/internal/api"
	"agent-harness/Squib/internal/security"
)

const (
	maxBashOutputBytes = 100 * 1024 // 100 KiB limit
	defaultCmdTimeout  = 30 * time.Second
)

type BashTool struct{}

func NewBashTool() *BashTool {
	return &BashTool{}
}

func (b *BashTool) Definition() api.ToolDef {
	return api.ToolDef{
		Name:        "bash",
		Description: "Run shell or PowerShell commands (git, docker, kubectl, etc.) within the workspace and return standard output and error (timeout 30s, max 100KB output).",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "The command line string to run.",
				},
			},
			"required": []string{"command"},
		},
	}
}

func (b *BashTool) Execute(ctx context.Context, args map[string]any) (string, error) {
	cmdStr, ok := args["command"].(string)
	if !ok || cmdStr == "" {
		return "", fmt.Errorf("missing or invalid 'command' argument")
	}

	cmdCtx, cancel := context.WithTimeout(ctx, defaultCmdTimeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(cmdCtx, "powershell", "-NoProfile", "-NonInteractive", "-Command", cmdStr)
	} else {
		cmd = exec.CommandContext(cmdCtx, "sh", "-c", cmdStr)
	}

	// Environment isolation: strip sensitive API keys (GROQ_API_KEY, tokens, etc.)
	cmd.Env = security.SanitizedEnviron()

	// Restrict execution directory to workspace
	if cwd, err := os.Getwd(); err == nil {
		cmd.Dir = cwd
	}

	// Capture output with size limit
	var buf bytes.Buffer
	limitWriter := io.LimitReader(&buf, maxBashOutputBytes)
	_ = limitWriter // we'll capture into limited buffer

	// Run command and capture combined stdout/stderr up to maxBashOutputBytes
	out, err := runWithLimit(cmd, maxBashOutputBytes)
	if cmdCtx.Err() == context.DeadlineExceeded {
		return fmt.Sprintf("%s\n[command timed out after %v]", out, defaultCmdTimeout), nil
	}
	if err != nil {
		return fmt.Sprintf("%s\n[exit error: %v]", out, err), nil
	}
	return out, nil
}

func runWithLimit(cmd *exec.Cmd, limit int64) (string, error) {
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Start()
	if err != nil {
		return "", err
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	var waitErr error
	select {
	case <-cmd.Context().Done():
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		waitErr = cmd.Context().Err()
	case waitErr = <-done:
	}

	data := buf.Bytes()
	if int64(len(data)) > limit {
		return string(data[:limit]) + "\n... [output truncated: exceeded 100KB limit]", waitErr
	}
	return string(data), waitErr
}
