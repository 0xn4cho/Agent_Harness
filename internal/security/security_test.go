package security

import (
	"os"
	"path/filepath"
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
			path:    "main.go",
			wantErr: false,
		},
		{
			name:    "safe nested relative file",
			path:    filepath.Join("internal", "api", "types.go"),
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
			if err == nil && !strings.HasPrefix(res, cwd) {
				t.Errorf("ValidateSafePath(%q) = %q, expected prefix %q", tt.path, res, cwd)
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
