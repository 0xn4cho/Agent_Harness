package security

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

var (
	// ansiRegex matches 7-bit and 8-bit C1 ANSI escape sequences
	ansiRegex = regexp.MustCompile(`\x1b(\[[0-9;?]*[a-zA-Z]|\][^\x07\x1b]*(\x07|\x1b\\)|[PX^_][^\x1b]*\x1b\\|.)`)

	// Sensitive directories and files that should never be read or written to
	sensitivePatterns = []string{
		".ssh",
		".kube",
		".aws",
		".azure",
		".gcp",
		".docker",
		".gnupg",
		".env",
		"id_rsa",
		"id_ecdsa",
		"id_ed25519",
		"credentials",
		"authorized_keys",
		"known_hosts",
		"kubeconfig",
		"config.json", // docker auth config
	}

	// Environment variable names to strip from child processes
	sensitiveEnvSubstrings = []string{
		"GROQ",
		"OPENAI",
		"ANTHROPIC",
		"API_KEY",
		"SECRET",
		"TOKEN",
		"PASSWORD",
		"PASSWD",
		"CREDENTIAL",
		"AUTH",
	}
)

// SanitizeTerminal strips ANSI escape sequences and non-printable control characters,
// preventing terminal injection and deceptive confirmation prompts.
func SanitizeTerminal(s string) string {
	// 1. Remove ANSI escape sequences
	cleaned := ansiRegex.ReplaceAllString(s, "")

	// 2. Filter out control characters except standard whitespace (\n, \t)
	var b strings.Builder
	b.Grow(len(cleaned))
	for _, r := range cleaned {
		if r == '\n' || r == '\t' || !unicode.IsControl(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	return b.String()
}

// TruncateString truncates a string to maxLen bytes, appending an ellipsis indicator if truncated.
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "... [truncated]"
}

// ValidateSafePath verifies that targetPath resides inside the allowed workspace directory
// and does not point to sensitive credentials or files.
func ValidateSafePath(targetPath string) (string, error) {
	if strings.TrimSpace(targetPath) == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// Resolve workspace (current working directory)
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	absCwd, err := filepath.Abs(cwd)
	if err != nil {
		return "", fmt.Errorf("failed to resolve workspace directory: %w", err)
	}

	// Resolve target path to an absolute, cleaned path
	var absTarget string
	if filepath.IsAbs(targetPath) {
		absTarget = filepath.Clean(targetPath)
	} else {
		absTarget = filepath.Clean(filepath.Join(absCwd, targetPath))
	}

	// Check if target is inside the workspace
	rel, err := filepath.Rel(absCwd, absTarget)
	if err != nil {
		return "", fmt.Errorf("failed to determine relative path: %w", err)
	}

	// If relative path starts with ".." or equals "..", it's outside the workspace
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("access denied: path %q is outside the allowed workspace directory (%s)", targetPath, absCwd)
	}

	// Check for sensitive files or directories anywhere in the path components
	lowerTarget := strings.ToLower(filepath.ToSlash(absTarget))
	for _, pattern := range sensitivePatterns {
		lowerPattern := strings.ToLower(pattern)
		// Check file/dir basename or path component
		base := strings.ToLower(filepath.Base(absTarget))
		if base == lowerPattern || strings.Contains(lowerTarget, "/"+lowerPattern+"/") || strings.HasSuffix(lowerTarget, "/"+lowerPattern) {
			return "", fmt.Errorf("access denied: sensitive path or file pattern %q is blocked for security", pattern)
		}
		if strings.HasPrefix(base, ".env") {
			return "", fmt.Errorf("access denied: environment file %q is blocked for security", base)
		}
	}

	return absTarget, nil
}

// SanitizedEnviron returns the current process environment excluding sensitive credentials and keys.
func SanitizedEnviron() []string {
	var env []string
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 0 {
			continue
		}
		key := strings.ToUpper(parts[0])

		isSensitive := false
		for _, sub := range sensitiveEnvSubstrings {
			if strings.Contains(key, sub) {
				isSensitive = true
				break
			}
		}

		if !isSensitive {
			env = append(env, e)
		}
	}
	return env
}
