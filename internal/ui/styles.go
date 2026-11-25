package ui

import (
	"fmt"
	"os"
	"strings"

	"agent-harness/Squib/internal/security"
)

type rgb struct{ r, g, b uint8 }

var (
	colorBoxBorder   = rgb{45, 115, 195}
	colorPanelBorder = rgb{40, 62, 98}
	colorTitleBlue   = rgb{65, 165, 255}
	colorCmdBlue     = rgb{85, 175, 255}
	colorSecondary   = rgb{145, 160, 180}
	colorData        = rgb{235, 242, 250}
	colorGreen       = rgb{60, 205, 115}
	colorBlue        = rgb{30, 100, 255}
	colorCyan        = rgb{0, 215, 255}
	colorWhite       = rgb{240, 250, 255}
	colorYellow      = rgb{255, 200, 60}
	colorRed         = rgb{255, 85, 85}
	colorGray        = rgb{120, 135, 155}
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

func paint(c rgb, s string) string {
	if !enabled || s == "" {
		return s
	}
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[0m", c.r, c.g, c.b, s)
}

func paintBold(c rgb, s string) string {
	if !enabled || s == "" {
		return s
	}
	return fmt.Sprintf("\x1b[1;38;2;%d;%d;%dm%s\x1b[0m", c.r, c.g, c.b, s)
}

func Prompt() string {
	if !enabled {
		return "> "
	}
	placeholder := "Enter a task..."
	hint := "Ctrl+C to exit"
	spaces := strings.Repeat(" ", 76-4-len(placeholder)-len(hint))
	return "  " + paintBold(colorCyan, "> ") + paint(colorGray, placeholder) + spaces + paint(colorGray, hint) + "\x1b[5G"
}

func ClearPromptLine() {
	if !enabled {
		return
	}
	if waitForFirstKey() {
		fmt.Print("\r\x1b[2K  " + paintBold(colorCyan, "> "))
	}
}

func ToolCall(name, args string) string {
	cleanArgs := security.SanitizeTerminal(args)
	const maxArgs = 90
	if len(cleanArgs) > maxArgs {
		cleanArgs = cleanArgs[:maxArgs] + "…"
	}
	return paint(colorBlue, "  → ") + paintBold(colorCyan, name) + " " + paint(colorGray, cleanArgs)
}

func Warn(s string) string { return paintBold(colorYellow, s) }

func Error(s string) string { return paintBold(colorRed, s) }

func Dim(s string) string { return paint(colorGray, s) }

func Accent(s string) string { return paintBold(colorCyan, s) }
