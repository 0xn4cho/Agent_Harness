package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Role struct {
	Name                string
	Model               string
	MaxCompletionTokens int
	ReasoningEffort     string
	Instructions        string
	AllowedTools        map[string]bool
}

func ResolveDir(dir string) string {
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		return dir
	}
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), dir)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return dir
}

func Load(dir string) (map[string]Role, error) {
	resolvedDir := ResolveDir(dir)
	roles := make(map[string]Role)
	for _, name := range []string{"leader", "implementer", "reviewer"} {
		role, err := loadRole(filepath.Join(resolvedDir, name+".md"))
		if err != nil {
			return nil, err
		}
		if role.Name == "" {
			role.Name = name
		}
		roles[name] = role
	}
	return roles, nil
}

func loadRole(path string) (Role, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Role{}, fmt.Errorf("read agent definition %q: %w", path, err)
	}

	role := Role{AllowedTools: make(map[string]bool)}
	content := strings.TrimPrefix(string(data), "\uFEFF")
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		role.Instructions = strings.TrimSpace(content)
		return role, nil
	}

	closingIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			closingIdx = i
			break
		}
	}
	if closingIdx < 0 {
		return Role{}, fmt.Errorf("agent definition %q has an unclosed front matter block", path)
	}

	frontMatter := lines[1:closingIdx]
	role.Instructions = strings.TrimSpace(strings.Join(lines[closingIdx+1:], "\n"))
	for _, line := range frontMatter {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		switch key {
		case "name":
			role.Name = value
		case "model":
			role.Model = value
		case "max_completion_tokens":
			if value != "" {
				n, err := strconv.Atoi(value)
				if err != nil || n <= 0 {
					return Role{}, fmt.Errorf("agent definition %q has invalid max_completion_tokens %q", path, value)
				}
				role.MaxCompletionTokens = n
			}
		case "reasoning_effort":
			role.ReasoningEffort = strings.ToLower(value)
		case "tools":
			for _, tool := range strings.Split(value, ",") {
				if tool = strings.TrimSpace(tool); tool != "" {
					role.AllowedTools[tool] = true
				}
			}
		}
	}
	return role, nil
}
