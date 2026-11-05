// Package ui contains everything related to how Squib looks in the terminal:
// colors, the startup banner and small helpers to style messages.
//
// It uses plain ANSI escape codes (no external dependencies). Colors are
// disabled automatically when the output is not a terminal (e.g. when piping
// to a file) or when the NO_COLOR environment variable is set.
package ui

import (
	"fmt"
	"os"

	"agent-harness/Squib/internal/security"
)

// rgb is a 24-bit color.
type rgb struct{ r, g, b uint8 }

// Squib's palette: deep blue -> cyan -> white.
var (
	colorDeepBlue = rgb{20, 60, 140}
	colorBlue     = rgb{30, 100, 255}
	colorCyan     = rgb{0, 215, 255}
	colorWhite    = rgb{240, 250, 255}
	colorYellow   = rgb{255, 200, 60}
	colorRed      = rgb{255, 85, 85}
	colorGray     = rgb{120, 135, 155}
)

// enabled reports whether ANSI colors should be emitted.
var enabled = detectColorSupport()

func detectColorSupport() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	info, err := os.Stdout.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return false // not a terminal (pipe, file, CI log...)
	}
	return enableVirtualTerminal()
}

// paint wraps s in a truecolor foreground escape sequence.
func paint(c rgb, s string) string {
	if !enabled || s == "" {
		return s
	}
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[0m", c.r, c.g, c.b, s)
}

// paintBold is like paint but also bold.
func paintBold(c rgb, s string) string {
	if !enabled || s == "" {
		return s
	}
	return fmt.Sprintf("\x1b[1;38;2;%d;%d;%dm%s\x1b[0m", c.r, c.g, c.b, s)
}

// lerp linearly interpolates between two colors (t in [0,1]).
func lerp(a, b rgb, t float64) rgb {
	mix := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return rgb{mix(a.r, b.r), mix(a.g, b.g), mix(a.b, b.b)}
}

// gradient returns the color at position t in [0,1] along several color stops.
func gradient(stops []rgb, t float64) rgb {
	if t <= 0 {
		return stops[0]
	}
	if t >= 1 {
		return stops[len(stops)-1]
	}
	segment := t * float64(len(stops)-1)
	i := int(segment)
	return lerp(stops[i], stops[i+1], segment-float64(i))
}

// ---- Public helpers used by main.go and the permission gate ----

// Prompt returns the styled input prompt.
func Prompt() string {
	return paintBold(colorCyan, "squib") + paint(colorBlue, " $> ")
}

// ToolCall formats the line shown when the agent invokes a tool.
func ToolCall(name, args string) string {
	cleanArgs := security.SanitizeTerminal(args)
	const maxArgs = 90
	if len(cleanArgs) > maxArgs {
		cleanArgs = cleanArgs[:maxArgs] + "…"
	}
	return paint(colorBlue, "  → ") + paintBold(colorCyan, name) + " " + paint(colorGray, cleanArgs)
}

// Warn styles a warning message (yellow).
func Warn(s string) string { return paintBold(colorYellow, s) }

// Error styles an error message (red).
func Error(s string) string { return paintBold(colorRed, s) }

// Dim styles secondary information (gray).
func Dim(s string) string { return paint(colorGray, s) }

// Accent styles highlighted text (cyan).
func Accent(s string) string { return paintBold(colorCyan, s) }
