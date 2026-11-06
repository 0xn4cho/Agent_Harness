package tool

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"agent-harness/Squib/internal/api"
	"agent-harness/Squib/internal/security"
)

const (
	maxBashOutputBytes = 100 * 1024
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

	assessment := security.ClassifyCommand(cmdStr)
	if assessment.Risk == security.RiskBlocked {
		return "", fmt.Errorf("command blocked by security policy: %s", strings.Join(assessment.Reasons, "; "))
	}

	cmdCtx, cancel := context.WithTimeout(ctx, defaultCmdTimeout)
	defer cancel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(cmdCtx, "powershell", "-NoProfile", "-NonInteractive", "-Command", cmdStr)
	} else {
		cmd = exec.CommandContext(cmdCtx, "sh", "-c", cmdStr)
	}

	cmd.Env = security.SanitizedEnviron()

	if cwd, err := os.Getwd(); err == nil {
		cmd.Dir = cwd
	}

	out, err := runWithLimit(cmdCtx, cmd, maxBashOutputBytes)
	if cmdCtx.Err() == context.DeadlineExceeded {
		return fmt.Sprintf("%s\n[command timed out after %v]", out, defaultCmdTimeout), nil
	}
	if err != nil {
		return fmt.Sprintf("%s\n[exit error: %v]", out, err), nil
	}
	return out, nil
}

type limitedWriter struct {
	mu        sync.Mutex
	buf       bytes.Buffer
	limit     int64
	truncated bool
}

func newLimitedWriter(limit int64) *limitedWriter {
	return &limitedWriter{limit: limit}
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	origLen := len(p)
	if origLen == 0 {
		return 0, nil
	}

	remaining := w.limit - int64(w.buf.Len())
	if remaining <= 0 {
		w.truncated = true
		return origLen, nil
	}

	if int64(origLen) > remaining {
		_, _ = w.buf.Write(p[:remaining])
		w.truncated = true
		return origLen, nil
	}

	_, err := w.buf.Write(p)
	return origLen, err
}

func (w *limitedWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.truncated {
		kb := w.limit / 1024
		if kb == 0 {
			return fmt.Sprintf("%s\n... [output truncated: exceeded %dB limit]", w.buf.String(), w.limit)
		}
		return fmt.Sprintf("%s\n... [output truncated: exceeded %dKB limit]", w.buf.String(), kb)
	}
	return w.buf.String()
}

func runWithLimit(ctx context.Context, cmd *exec.Cmd, limit int64) (string, error) {
	lw := newLimitedWriter(limit)
	cmd.Stdout = lw
	cmd.Stderr = lw

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
	case <-ctx.Done():
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		waitErr = ctx.Err()
	case waitErr = <-done:
	}

	return lw.String(), waitErr
}
