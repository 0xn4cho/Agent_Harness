package tool

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFileToolSecurity(t *testing.T) {
	rf := NewReadFileTool()
	ctx := context.Background()

	// 1. Reading sensitive path like .ssh
	_, err := rf.Execute(ctx, map[string]any{"path": filepath.Join(".ssh", "id_rsa")})
	if err == nil {
		t.Errorf("expected error reading .ssh/id_rsa, got nil")
	}

	// 2. Reading path outside workspace
	_, err = rf.Execute(ctx, map[string]any{"path": filepath.Join("..", "sensitive.txt")})
	if err == nil {
		t.Errorf("expected error reading file outside workspace, got nil")
	}

	// 3. Reading existing safe file
	tmpFile, err := os.CreateTemp(".", "saferead_*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("safe content")
	tmpFile.Close()

	content, err := rf.Execute(ctx, map[string]any{"path": filepath.Base(tmpFile.Name())})
	if err != nil {
		t.Fatalf("unexpected error reading safe file: %v", err)
	}
	if content != "safe content" {
		t.Errorf("read content = %q, want 'safe content'", content)
	}
}

func TestWriteFileToolSecurity(t *testing.T) {
	wf := NewWriteFileTool()
	ctx := context.Background()

	// 1. Writing outside workspace
	_, err := wf.Execute(ctx, map[string]any{
		"path":    filepath.Join("..", "hacked.txt"),
		"content": "malicious",
	})
	if err == nil {
		t.Errorf("expected error writing outside workspace, got nil")
	}

	// 2. Writing sensitive .env file
	_, err = wf.Execute(ctx, map[string]any{
		"path":    ".env",
		"content": "SECRET=123",
	})
	if err == nil {
		t.Errorf("expected error writing to .env, got nil")
	}

	// 3. Writing safe file inside workspace
	safePath := "test_output_safe.txt"
	defer os.Remove(safePath)

	out, err := wf.Execute(ctx, map[string]any{
		"path":    safePath,
		"content": "hello world",
	})
	if err != nil {
		t.Fatalf("unexpected error writing safe file: %v", err)
	}
	if !strings.Contains(out, "wrote") {
		t.Errorf("expected 'wrote ...', got %q", out)
	}
}

func TestDevOpsToolValidation(t *testing.T) {
	dot := NewDevOpsTool()
	ctx := context.Background()

	// Invalid target enum
	_, err := dot.Execute(ctx, map[string]any{"target": "invalid_enum_val"})
	if err == nil {
		t.Errorf("expected error for invalid enum value, got nil")
	}
	if !strings.Contains(err.Error(), "invalid 'target'") {
		t.Errorf("expected invalid target error, got: %v", err)
	}
}

func TestBashToolEnvSanitization(t *testing.T) {
	os.Setenv("GROQ_API_KEY", "gsk_supersecretkey123")
	defer os.Unsetenv("GROQ_API_KEY")

	b := NewBashTool()
	ctx := context.Background()

	// Test checking GROQ_API_KEY inside bash/powershell
	cmdStr := "$env:GROQ_API_KEY"
	out, err := b.Execute(ctx, map[string]any{"command": cmdStr})
	if err != nil {
		t.Fatalf("bash execute error: %v", err)
	}

	if strings.Contains(out, "gsk_supersecretkey123") {
		t.Errorf("BashTool leaked GROQ_API_KEY into child execution! Output: %q", out)
	}
}
