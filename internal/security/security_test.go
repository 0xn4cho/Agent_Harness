package security

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSanitizeTerminal(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "ansi color escape codes",
			input:    "\x1b[31mred text\x1b[0m",
			expected: "red text",
		},
		{
			name:     "ansi cursor movement and line clear",
			input:    "\x1b[2K\rAllow execution? [y/n]: y",
			expected: " Allow execution? [y/n]: y",
		},
		{
			name:     "control characters",
			input:    "line1\x00\x07\x08line2",
			expected: "line1   line2",
		},
		{
			name:     "preserve newline and tab",
			input:    "line1\n\tline2",
			expected: "line1\n\tline2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeTerminal(tt.input)
			if got != tt.expected {
				t.Errorf("SanitizeTerminal(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestValidateSafePath(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "safe relative file",
			path:    "security.go",
			wantErr: false,
		},
		{
			name:    "safe non-existent nested relative file",
			path:    filepath.Join("subdir", "newfile.txt"),
			wantErr: false,
		},
		{
			name:    "path traversal outside workspace",
			path:    filepath.Join("..", "secret.txt"),
			wantErr: true,
		},
		{
			name:    "sensitive .ssh directory",
			path:    filepath.Join(".ssh", "id_rsa"),
			wantErr: true,
		},
		{
			name:    "sensitive .kube config",
			path:    filepath.Join(".kube", "config"),
			wantErr: true,
		},
		{
			name:    "sensitive .aws credentials",
			path:    filepath.Join(".aws", "credentials"),
			wantErr: true,
		},
		{
			name:    "sensitive .env file",
			path:    ".env",
			wantErr: true,
		},
		{
			name:    "sensitive .env.production file",
			path:    ".env.production",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := ValidateSafePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSafePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
			if err == nil && !strings.HasPrefix(strings.ToLower(res), strings.ToLower(cwd)) {
				t.Errorf("ValidateSafePath(%q) = %q, expected prefix %q", tt.path, res, cwd)
			}
		})
	}
}

func TestValidateSafePathBlocksSymlinksAndJunctionsOutsideWorkspace(t *testing.T) {
	origCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tempRoot, err := os.MkdirTemp(origCwd, "symlink_test_*")
	if err != nil {
		t.Fatalf("failed to create temp root in workspace: %v", err)
	}
	defer os.RemoveAll(tempRoot)

	outsideDir := filepath.Join(tempRoot, "outside")
	insideWorkspace := filepath.Join(tempRoot, "workspace")
	if err := os.MkdirAll(outsideDir, 0755); err != nil {
		t.Fatalf("failed to create outsideDir: %v", err)
	}
	if err := os.MkdirAll(insideWorkspace, 0755); err != nil {
		t.Fatalf("failed to create insideWorkspace: %v", err)
	}

	outsideFile := filepath.Join(outsideDir, "outside_secret.txt")
	if err := os.WriteFile(outsideFile, []byte("secret"), 0644); err != nil {
		t.Fatalf("failed to create outside file: %v", err)
	}

	if err := os.Chdir(insideWorkspace); err != nil {
		t.Fatalf("failed to chdir into insideWorkspace: %v", err)
	}
	defer os.Chdir(origCwd)

	if runtime.GOOS == "windows" {
		junctionPath := filepath.Join(insideWorkspace, "test_junction_escape")
		cmd := exec.Command("cmd", "/c", "mklink", "/J", junctionPath, outsideDir)
		if out, err := cmd.CombinedOutput(); err == nil {
			if _, verr := ValidateSafePath(filepath.Join("test_junction_escape", "outside_secret.txt")); verr == nil {
				t.Errorf("expected ValidateSafePath to block reading existing file through NTFS junction pointing outside workspace")
			}

			if _, verr := ValidateSafePath(filepath.Join("test_junction_escape", "new_subdir", "escaped.txt")); verr == nil {
				t.Errorf("expected ValidateSafePath to block writing new file through NTFS junction pointing outside workspace")
			}
		} else {
			t.Logf("skipping NTFS junction creation in restricted sandbox (%v: %s)", err, strings.TrimSpace(string(out)))
		}
	}

	symlinkPath := filepath.Join(insideWorkspace, "test_symlink_escape.txt")
	if err := os.Symlink(outsideFile, symlinkPath); err == nil {
		if _, verr := ValidateSafePath("test_symlink_escape.txt"); verr == nil {
			t.Errorf("expected ValidateSafePath to block symlink pointing outside workspace")
		}
	}

	fakeLinkDir := filepath.Join(insideWorkspace, "linked_dir")
	if err := os.MkdirAll(fakeLinkDir, 0755); err != nil {
		t.Fatalf("failed to create fakeLinkDir: %v", err)
	}
	origEval := evalSymlinks
	defer func() { evalSymlinks = origEval }()
	evalSymlinks = func(path string) (string, error) {
		if strings.EqualFold(filepath.Clean(path), filepath.Clean(fakeLinkDir)) {
			return outsideDir, nil
		}
		return origEval(path)
	}

	if _, verr := ValidateSafePath(filepath.Join("linked_dir", "outside_secret.txt")); verr == nil {
		t.Errorf("expected ValidateSafePath to block resolved junction/symlink escaping to %s", outsideDir)
	}
	if _, verr := ValidateSafePath(filepath.Join("linked_dir", "nested", "newfile.txt")); verr == nil {
		t.Errorf("expected ValidateSafePath to block new file creation inside resolved junction/symlink escaping to %s", outsideDir)
	}
}

func TestClassifyCommand(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		wantRisk CommandRisk
	}{
		{
			name:     "allowlisted git status",
			cmd:      "git status",
			wantRisk: RiskLow,
		},
		{
			name:     "allowlisted dir in workspace",
			cmd:      "dir",
			wantRisk: RiskLow,
		},
		{
			name:     "allowlisted go vet",
			cmd:      "go vet ./...",
			wantRisk: RiskLow,
		},
		{
			name:     "pipeline chaining requires full shell approval",
			cmd:      "git status | Out-File status.txt",
			wantRisk: RiskHigh,
		},
		{
			name:     "accessing $HOME requires full shell approval",
			cmd:      "Get-ChildItem $HOME",
			wantRisk: RiskHigh,
		},
		{
			name:     "accessing Windows registry requires full shell approval",
			cmd:      "Get-ItemProperty HKCU:\\Software",
			wantRisk: RiskHigh,
		},
		{
			name:     "network access requires full shell approval",
			cmd:      "Invoke-WebRequest https://example.com",
			wantRisk: RiskHigh,
		},
		{
			name:     "absolute path outside workspace requires full shell approval",
			cmd:      "Get-Content C:\\Windows\\win.ini",
			wantRisk: RiskHigh,
		},
		{
			name:     "blocked encoded powershell command",
			cmd:      "powershell -EncodedCommand ZQBjAGgAbwA=",
			wantRisk: RiskBlocked,
		},
		{
			name:     "blocked ssh private key access",
			cmd:      "cat ~/.ssh/id_rsa",
			wantRisk: RiskBlocked,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyCommand(tt.cmd)
			if got.Risk != tt.wantRisk {
				t.Errorf("ClassifyCommand(%q).Risk = %q (reasons=%v), want %q", tt.cmd, got.Risk, got.Reasons, tt.wantRisk)
			}
		})
	}
}

func TestSanitizedEnviron(t *testing.T) {
	os.Setenv("GROQ_API_KEY", "gsk_test123456789")
	os.Setenv("MY_SECRET_TOKEN", "supersecret")
	os.Setenv("NORMAL_APP_VAR", "visible")
	defer func() {
		os.Unsetenv("GROQ_API_KEY")
		os.Unsetenv("MY_SECRET_TOKEN")
		os.Unsetenv("NORMAL_APP_VAR")
	}()

	env := SanitizedEnviron()
	for _, e := range env {
		if strings.HasPrefix(e, "GROQ_API_KEY=") {
			t.Errorf("SanitizedEnviron() leaked GROQ_API_KEY: %s", e)
		}
		if strings.HasPrefix(e, "MY_SECRET_TOKEN=") {
			t.Errorf("SanitizedEnviron() leaked MY_SECRET_TOKEN: %s", e)
		}
	}

	foundNormal := false
	for _, e := range env {
		if strings.HasPrefix(e, "NORMAL_APP_VAR=") {
			foundNormal = true
			break
		}
	}
	if !foundNormal {
		t.Errorf("SanitizedEnviron() did not retain NORMAL_APP_VAR")
	}
}
