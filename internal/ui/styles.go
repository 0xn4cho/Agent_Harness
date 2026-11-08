package ui

import (
	"fmt"
	"os"

	"agent-harness/Squib/internal/security"
)

type rgb struct{ r, g, b uint8 }

var (
	colorDeepBlue = rgb{20, 60, 140}
	colorBlue     = rgb{30, 100, 255}
	colorCyan     = rgb{0, 215, 255}
	colorWhite    = rgb{240, 250, 255}
	colorYellow   = rgb{255, 200, 60}
	colorRed      = rgb{255, 85, 85}
	colorGray     = rgb{120, 135, 155}
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

func lerp(a, b rgb, t float64) rgb {
	mix := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return rgb{mix(a.r, b.r), mix(a.g, b.g), mix(a.b, b.b)}
}

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

func Prompt() string {
	return paintBold(colorCyan, "> ")
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
