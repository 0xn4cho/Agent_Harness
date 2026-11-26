package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func sanitizeRoleName(roleName string) (string, error) {
	safe := filepath.Base(strings.TrimSpace(roleName))
	if safe == "" || safe == "." || safe == ".." || safe == string(filepath.Separator) || safe == "/" {
		return "", fmt.Errorf("invalid role name %q", roleName)
	}
	return safe, nil
}

func WriteProgress(dir, roleName, content string) (string, error) {
	if strings.TrimSpace(dir) == "" {
		return "", nil
	}
	safeRole, err := sanitizeRoleName(roleName)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create progress directory %q: %w", dir, err)
	}

	target := filepath.Join(dir, safeRole+".md")
	body := fmt.Sprintf("# %s report\n\nUpdated: %s\n\n%s\n",
		safeRole, time.Now().Format(time.RFC3339), strings.TrimSpace(content))
	if err := os.WriteFile(target, []byte(body), 0o644); err != nil {
		return "", fmt.Errorf("write progress report %q: %w", target, err)
	}
	return target, nil
}

func ReadProgress(dir, roleName string, maxChars int) (string, error) {
	safeRole, err := sanitizeRoleName(roleName)
	if err != nil {
		return "", err
	}
	target := filepath.Join(dir, safeRole+".md")
	data, err := os.ReadFile(target)
	if err != nil {
		return "", fmt.Errorf("read progress report %q: %w", target, err)
	}
	text := strings.TrimSpace(string(data))
	if maxChars > 0 {
		runes := []rune(text)
		if len(runes) > maxChars {
			text = string(runes[:maxChars]) + "\n... [truncated]"
		}
	}
	return text, nil
}
