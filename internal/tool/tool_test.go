package tool

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFileToolSecurity(t *testing.T) {
	rf := NewReadFileTool()
	ctx := context.Background()

	_, err := rf.Execute(ctx, map[string]any{"path": filepath.Join(".ssh", "id_rsa")})
	if err == nil {
		t.Errorf("expected error reading .ssh/id_rsa, got nil")
	}

	_, err = rf.Execute(ctx, map[string]any{"path": filepath.Join("..", "sensitive.txt")})
	if err == nil {
		t.Errorf("expected error reading file outside workspace, got nil")
	}

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

	_, err := wf.Execute(ctx, map[string]any{
		"path":    filepath.Join("..", "hacked.txt"),
		"content": "malicious",
	})
	if err == nil {
		t.Errorf("expected error writing outside workspace, got nil")
	}

	_, err = wf.Execute(ctx, map[string]any{
		"path":    ".env",
		"content": "SECRET=123",
	})
	if err == nil {
		t.Errorf("expected error writing to .env, got nil")
	}

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

func TestDevOpsToolValidationAndKubeconfigIsolation(t *testing.T) {
	dot := NewDevOpsTool()
	ctx := context.Background()

	_, err := dot.Execute(ctx, map[string]any{"target": "invalid_enum_val"})
	if err == nil {
		t.Errorf("expected error for invalid enum value, got nil")
	}
	if !strings.Contains(err.Error(), "invalid 'target'") {
		t.Errorf("expected invalid target error, got: %v", err)
	}

	os.Unsetenv("SQUIB_KUBECONFIG")
	out, err := dot.Execute(ctx, map[string]any{"target": "k8s"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "global ~/.kube/config access is disabled") {
		t.Errorf("expected k8s isolation notice when no controlled kubeconfig is set, got: %q", out)
	}

	outEscape, err := dot.Execute(ctx, map[string]any{
		"target":     "k8s",
		"kubeconfig": filepath.Join("..", "outside_kubeconfig"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(outEscape, "k8s isolation:") {
		t.Errorf("expected k8s isolation error for outside-workspace kubeconfig, got: %q", outEscape)
	}
}

func TestBashToolEnvSanitization(t *testing.T) {
	os.Setenv("GROQ_API_KEY", "gsk_supersecretkey123")
	defer os.Unsetenv("GROQ_API_KEY")

	b := NewBashTool()
	ctx := context.Background()

	cmdStr := "$env:GROQ_API_KEY"
	out, err := b.Execute(ctx, map[string]any{"command": cmdStr})
	if err != nil {
		t.Fatalf("bash execute error: %v", err)
	}

	if strings.Contains(out, "gsk_supersecretkey123") {
		t.Errorf("BashTool leaked GROQ_API_KEY into child execution! Output: %q", out)
	}
}

func TestLimitedWriterCapsMemoryDuringExecution(t *testing.T) {
	const limit = 1024
	lw := newLimitedWriter(limit)

	chunk := bytes.Repeat([]byte("A"), 64*1024)
	for i := 0; i < 160; i++ {
		n, err := lw.Write(chunk)
		if err != nil {
			t.Fatalf("unexpected write error: %v", err)
		}
		if n != len(chunk) {
			t.Fatalf("expected Write to consume %d bytes, got %d", len(chunk), n)
		}
		if lw.buf.Len() > limit {
			t.Fatalf("limitedWriter buffer exceeded limit: got %d > %d", lw.buf.Len(), limit)
		}
	}

	if !lw.truncated {
		t.Errorf("expected limitedWriter to mark output as truncated")
	}
	if !strings.Contains(lw.String(), "[output truncated: exceeded 1KB limit]") {
		t.Errorf("expected truncation notice in output, got len=%d", len(lw.String()))
	}
}

func TestRegistryDefinitionsDeterministicOrder(t *testing.T) {
	reg := NewRegistry()
	reg.Register(NewWriteFileTool())
	reg.Register(NewBashTool())
	reg.Register(NewDevOpsTool())
	reg.Register(NewReadFileTool())
	reg.Register(NewListDirectoryTool())
	reg.Register(NewFindFilesTool())
	reg.Register(NewCopyPathTool())
	reg.Register(NewMovePathTool())

	want := []string{"bash", "copy_path", "devops_status", "find_files", "list_directory", "move_path", "read_file", "write_file"}

	for iter := 0; iter < 20; iter++ {
		defs := reg.Definitions()
		if len(defs) != len(want) {
			t.Fatalf("expected %d definitions, got %d", len(want), len(defs))
		}
		for i, d := range defs {
			if d.Name != want[i] {
				t.Fatalf("iteration %d: defs[%d].Name = %q, want %q", iter, i, d.Name, want[i])
			}
		}
	}
}

func TestExternalFileTools(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "source")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(sourceDir, "Boletin 3.pdf")
	if err := os.WriteFile(source, []byte("notes"), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	listed, err := NewListDirectoryTool().Execute(ctx, map[string]any{"path": sourceDir})
	if err != nil || !strings.Contains(listed, "Boletin 3.pdf") {
		t.Fatalf("list_directory = %q, %v", listed, err)
	}
	found, err := NewFindFilesTool().Execute(ctx, map[string]any{"path": root, "pattern": "*.pdf"})
	if err != nil || !strings.Contains(found, source) {
		t.Fatalf("find_files = %q, %v", found, err)
	}

	copyPath := filepath.Join(root, "vault", "Boletin 3.pdf")
	if _, err := NewCopyPathTool().Execute(ctx, map[string]any{"source": source, "destination": copyPath}); err != nil {
		t.Fatalf("copy_path failed: %v", err)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("copy_path removed source: %v", err)
	}
	if data, err := os.ReadFile(copyPath); err != nil || string(data) != "notes" {
		t.Fatalf("copied content = %q, %v", data, err)
	}
	if _, err := NewCopyPathTool().Execute(ctx, map[string]any{"source": source, "destination": copyPath}); err == nil {
		t.Fatal("copy_path allowed an overwrite")
	}

	moved := filepath.Join(root, "vault", "Movido.pdf")
	if _, err := NewMovePathTool().Execute(ctx, map[string]any{"source": source, "destination": moved}); err != nil {
		t.Fatalf("move_path failed: %v", err)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("move_path left source in place: %v", err)
	}
	if _, err := os.Stat(moved); err != nil {
		t.Fatalf("move_path did not create destination: %v", err)
	}
}
