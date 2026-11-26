package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRepoAgents(t *testing.T) {
	roles, err := Load(filepath.Join("..", "..", "agents"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	leader := roles["leader"]
	if leader.Model != "openai/gpt-oss-20b" || leader.MaxCompletionTokens != 600 || leader.ReasoningEffort != "low" {
		t.Errorf("unexpected leader config: %+v", leader)
	}
	for _, toolName := range []string{"read_file", "list_directory", "find_files"} {
		if !leader.AllowedTools[toolName] {
			t.Errorf("leader missing expected tool %q", toolName)
		}
	}
	if len(leader.AllowedTools) != 3 {
		t.Errorf("unexpected leader tools count: %+v", leader.AllowedTools)
	}

	impl := roles["implementer"]
	if impl.Model != "openai/gpt-oss-120b" || impl.MaxCompletionTokens != 1024 || impl.ReasoningEffort != "low" {
		t.Errorf("unexpected implementer config: %+v", impl)
	}
	for _, toolName := range []string{"bash", "copy_path", "devops_status", "find_files", "list_directory", "move_path", "read_file", "write_file"} {
		if !impl.AllowedTools[toolName] {
			t.Errorf("implementer missing expected tool %q", toolName)
		}
	}

	rev := roles["reviewer"]
	if rev.Model != "openai/gpt-oss-20b" || rev.MaxCompletionTokens != 600 || rev.ReasoningEffort != "low" {
		t.Errorf("unexpected reviewer config: %+v", rev)
	}
	for _, toolName := range []string{"bash", "devops_status", "find_files", "list_directory", "read_file"} {
		if !rev.AllowedTools[toolName] {
			t.Errorf("reviewer missing expected tool %q", toolName)
		}
	}
	for _, forbidden := range []string{"write_file", "copy_path", "move_path"} {
		if rev.AllowedTools[forbidden] {
			t.Errorf("reviewer must not allow %s", forbidden)
		}
	}
}

func TestLoadRoleEdgeCases(t *testing.T) {
	dir := t.TempDir()

	crlfAtEOF := "\uFEFF---\r\nname: leader\r\nmodel: test/model\r\nmax_completion_tokens: 300\r\nreasoning_effort: MEDIUM\r\ntools: read_file, bash\r\n---"
	path := filepath.Join(dir, "eof.md")
	if err := os.WriteFile(path, []byte(crlfAtEOF), 0o644); err != nil {
		t.Fatal(err)
	}
	role, err := loadRole(path)
	if err != nil {
		t.Fatalf("loadRole failed on EOF delimiter: %v", err)
	}
	if role.Model != "test/model" || role.MaxCompletionTokens != 300 || role.ReasoningEffort != "medium" {
		t.Errorf("unexpected parsed role: %+v", role)
	}
	if !role.AllowedTools["read_file"] || !role.AllowedTools["bash"] {
		t.Errorf("unexpected tools: %+v", role.AllowedTools)
	}

	unclosed := filepath.Join(dir, "unclosed.md")
	if err := os.WriteFile(unclosed, []byte("---\nname: leader\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadRole(unclosed); err == nil {
		t.Errorf("expected error for unclosed front matter")
	}

	badTokens := filepath.Join(dir, "bad_tokens.md")
	if err := os.WriteFile(badTokens, []byte("---\nmax_completion_tokens: -10\n---\nbody"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadRole(badTokens); err == nil {
		t.Errorf("expected error for negative max_completion_tokens")
	}

	if _, err := Load(filepath.Join(dir, "missing")); err == nil {
		t.Errorf("expected error for missing agents directory")
	}
}

func TestWriteAndReadProgress(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "progress")

	writtenPath, err := WriteProgress(dir, "../leader", "Plan detallado con acentos: áéíóú")
	if err != nil {
		t.Fatalf("WriteProgress error: %v", err)
	}
	if filepath.Dir(writtenPath) != dir || filepath.Base(writtenPath) != "leader.md" {
		t.Fatalf("unexpected writtenPath %q", writtenPath)
	}

	full, err := ReadProgress(dir, "leader", 0)
	if err != nil {
		t.Fatalf("ReadProgress error: %v", err)
	}
	if !strings.Contains(full, "# leader report") || !strings.Contains(full, "Plan detallado con acentos: áéíóú") {
		t.Errorf("unexpected report content:\n%s", full)
	}

	truncated, err := ReadProgress(dir, "leader", 12)
	if err != nil {
		t.Fatalf("ReadProgress truncated error: %v", err)
	}
	if !strings.HasSuffix(truncated, "... [truncated]") {
		t.Errorf("expected truncation marker, got %q", truncated)
	}
}
