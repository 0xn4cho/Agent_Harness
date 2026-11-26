package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"agent-harness/Squib/internal/security"
)

const (
	TokyoBlue   = "\033[1;38;2;122;162;247m"
	TokyoCyan   = "\033[1;38;2;125;207;255m"
	TokyoPurple = "\033[1;38;2;187;154;247m"
	TokyoGreen  = "\033[38;2;158;206;106m"
	TokyoAmber  = "\033[38;2;224;175;104m"
	TokyoRed    = "\033[1;38;2;247;118;142m"
	TokyoText   = "\033[38;2;192;202;245m"
	TokyoDim    = "\033[38;2;86;95;137m"
	TokyoBorder = "\033[38;2;59;66;97m"

	BoldCyan    = TokyoCyan
	PromptColor = TokyoPurple
	Reset       = "\033[0m"
)

var enabled = detectColorSupport()

func detectColorSupport() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	info, err := os.Stdout.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	return enableVirtualTerminal()
}

func wrap(code, s string) string {
	if !enabled || s == "" {
		return s
	}
	return code + s + Reset
}

func Cyan(s string) string { return wrap(TokyoCyan, s) }

func Blue(s string) string { return wrap(TokyoBlue, s) }

func Purple(s string) string { return wrap(TokyoPurple, s) }

func Text(s string) string { return wrap(TokyoText, s) }

func Border(s string) string { return wrap(TokyoBorder, s) }

func Dimmed(s string) string { return wrap(TokyoDim, s) }

func Dim(s string) string { return Dimmed(s) }

func Accent(s string) string { return Cyan(s) }

func Warn(s string) string { return wrap(TokyoAmber, s) }

func Error(s string) string { return wrap(TokyoRed, s) }

func Success(s string) string { return wrap(TokyoGreen, s) }

func Prompt() string { return wrap(PromptColor, "❯ ") }

func PromptGlyph() string { return wrap(PromptColor, "❯") }

func RestoreCursor() {
	if enabled {
		fmt.Print("\x1b[?25h\r\033[K")
	}
}

func RoleColor(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "leader":
		return TokyoBlue
	case "implementer":
		return TokyoPurple
	case "reviewer":
		return TokyoGreen
	default:
		return TokyoCyan
	}
}

func HighlightVerdict(text string) string {
	if !enabled || text == "" {
		return text
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(strings.Trim(strings.TrimSpace(line), "*`"))
		switch {
		case strings.EqualFold(trimmed, "APPROVED"):
			lines[i] = strings.Replace(line, "APPROVED", Success("APPROVED"), 1)
		case strings.EqualFold(trimmed, "CHANGES_REQUESTED"):
			lines[i] = strings.Replace(line, "CHANGES_REQUESTED", Warn("CHANGES_REQUESTED"), 1)
		}
	}
	return strings.Join(lines, "\n")
}

func SummarizeToolArgs(name, rawArgs string) string {
	var parsed map[string]any
	summary := ""
	if err := json.Unmarshal([]byte(rawArgs), &parsed); err == nil && parsed != nil {
		strArg := func(key string) string {
			v, _ := parsed[key].(string)
			return strings.TrimSpace(v)
		}
		switch name {
		case "read_file", "write_file", "list_directory":
			summary = strArg("path")
		case "bash":
			summary = strArg("command")
		case "find_files":
			pattern := strArg("pattern")
			dir := strArg("path")
			switch {
			case pattern != "" && dir != "":
				summary = pattern + " in " + dir
			case pattern != "":
				summary = pattern
			default:
				summary = dir
			}
		case "copy_path", "move_path":
			src := strArg("source")
			dst := strArg("destination")
			if src != "" && dst != "" {
				summary = src + " → " + dst
			} else if src != "" {
				summary = src
			} else {
				summary = dst
			}
		case "devops_status":
			summary = strArg("target")
		}
	}
	if summary == "" {
		summary = strings.TrimSpace(rawArgs)
	}
	clean := security.SanitizeTerminal(summary)
	clean = strings.Join(strings.Fields(clean), " ")
	const maxLen = 72
	runes := []rune(clean)
	if len(runes) > maxLen {
		return string(runes[:maxLen]) + "…"
	}
	return clean
}

func ToolCall(name, rawArgs string) string {
	summary := SummarizeToolArgs(name, rawArgs)
	if summary == "" {
		return "  " + Border("├─") + " " + Cyan(name)
	}
	return "  " + Border("├─") + " " + Cyan(fmt.Sprintf("%-11s", name)) + " " + Dim(summary)
}

func ToolResult(branch, name, rawArgs string, ok bool) string {
	if branch == "" {
		branch = "└─"
	}
	status := Success("✓")
	if !ok {
		status = Error("✗")
	}
	summary := SummarizeToolArgs(name, rawArgs)
	if summary == "" {
		return "  " + Border(branch) + " " + Cyan(fmt.Sprintf("%-11s", name)) + " " + status
	}
	return "  " + Border(branch) + " " + Cyan(fmt.Sprintf("%-11s", name)) + " " + Text(summary) + "  " + status
}
