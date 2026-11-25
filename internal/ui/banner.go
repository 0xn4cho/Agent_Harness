package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	bannerBrand    = "  SC  "
	bannerVersion  = "v0.1"
	bannerSubtitle = "experimental agent harness · local tools + human approval"
	bannerWidth    = 74
	leftColWidth   = 40
	rightColWidth  = 33
	cmdColWidth    = 38
)

// Banner keeps the identity and useful startup details together while leaving
// the terminal focused on the next prompt.
func Banner(model string) string {
	directory, err := os.Getwd()
	if err != nil {
		directory = "unavailable"
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(boxBorder(colorBoxBorder, "╭", "─", "╮") + "\n")
	b.WriteString(boxLine(colorBoxBorder, "  "+paint(colorBoxBorder, "╭──────╮")+"  "+paintBold(colorTitleBlue, "SquibCode")+"  "+paint(colorSecondary, bannerVersion)) + "\n")
	b.WriteString(boxLine(colorBoxBorder, "  "+paint(colorBoxBorder, "│")+paintBold(colorTitleBlue, bannerBrand)+paint(colorBoxBorder, "│")+"  "+paint(colorSecondary, bannerSubtitle)) + "\n")
	b.WriteString(boxLine(colorBoxBorder, "  "+paint(colorBoxBorder, "╰──────╯")+"  "+paint(colorBoxBorder, strings.Repeat("─", 60))) + "\n")
	b.WriteString(boxLine(colorBoxBorder, "") + "\n")
	b.WriteString(splitBoxLine(
		metaItem("◈", "Model", 9, shorten(model, 23), colorData),
		metaItem("✦", "Tools", 8, "4 available", colorData),
	) + "\n")
	b.WriteString(splitBoxLine(
		metaItem("■", "Workspace", 9, formatWorkspace(directory, 23), colorData),
		metaItem("●", "Approval", 8, "Human approval", colorGreen),
	) + "\n")
	b.WriteString(splitBoxLine(
		metaItem("●", "Author", 9, shorten("github.com/0xn4cho", 23), colorData),
		"",
	) + "\n")
	b.WriteString(boxBorder(colorBoxBorder, "╰", "─", "╯") + "\n")
	b.WriteString("\n")
	b.WriteString(Welcome() + "\n")
	b.WriteString("    " + paint(colorSecondary, "Your local AI development environment is ready.") + "\n")
	b.WriteString("    " + paint(colorSecondary, "Ask me to develop, analyze or troubleshoot your projects.") + "\n")
	b.WriteString("\n")
	b.WriteString(boxBorder(colorPanelBorder, "╭", "─", "╮") + "\n")
	b.WriteString(boxLine(colorPanelBorder, "  "+paintBold(colorYellow, "✦")+" "+paintBold(colorTitleBlue, "QUICK COMMANDS")) + "\n")
	b.WriteString(cmdRow(quickCmd("/compact", "Compact context"), quickCmd("exit", "Exit SquibCode")) + "\n")
	b.WriteString(boxBorder(colorPanelBorder, "╰", "─", "╯") + "\n")
	return b.String()
}

func Welcome() string {
	return "  " + paintBold(colorTitleBlue, ">") + " " + paintBold(colorData, "Welcome to SquibCode!")
}

func metaItem(icon, label string, labelWidth int, value string, valueColor rgb) string {
	return "  " + paint(colorTitleBlue, icon) + " " +
		paint(colorSecondary, fmt.Sprintf("%-*s :", labelWidth, label)) + " " +
		paint(valueColor, value)
}

func splitBoxLine(left, right string) string {
	leftPad := max(0, leftColWidth-visibleWidth(left))
	rightPad := max(0, rightColWidth-visibleWidth(right))
	return paint(colorBoxBorder, "│") +
		left + strings.Repeat(" ", leftPad) +
		paint(colorBoxBorder, "│") +
		right + strings.Repeat(" ", rightPad) +
		paint(colorBoxBorder, "│")
}

func quickCmd(cmd, desc string) string {
	return paint(colorCmdBlue, fmt.Sprintf("%-9s", cmd)) + paint(colorSecondary, " · "+desc)
}

func cmdRow(left, right string) string {
	leftPad := max(0, cmdColWidth-visibleWidth(left))
	return boxLine(colorPanelBorder, "  "+left+strings.Repeat(" ", leftPad)+right)
}

func boxBorder(c rgb, left, fill, right string) string {
	return paint(c, left+strings.Repeat(fill, bannerWidth)+right)
}

func boxLine(c rgb, content string) string {
	padding := max(0, bannerWidth-visibleWidth(content))
	return paint(c, "│") + content + strings.Repeat(" ", padding) + paint(c, "│")
}

func formatWorkspace(directory string, maxLen int) string {
	if directory == "" || directory == "unavailable" {
		return "unavailable"
	}
	base := filepath.Base(directory)
	if base != "" && base != "." && base != string(filepath.Separator) && base != "/" {
		return shorten("~/"+base, maxLen)
	}
	return shortenPath(directory, maxLen)
}

func shorten(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	if maxLen == 1 {
		return "…"
	}
	runes := []rune(s)
	return "…" + string(runes[len(runes)-(maxLen-1):])
}

func shortenPath(path string, maxLen int) string {
	if utf8.RuneCountInString(path) <= maxLen {
		return path
	}
	if volume := strings.Index(path, `\`); volume > 0 && maxLen > volume+4 {
		prefix := path[:volume+1] + "…\\"
		tailLength := maxLen - utf8.RuneCountInString(prefix)
		runes := []rune(path)
		return prefix + string(runes[len(runes)-tailLength:])
	}
	return shorten(path, maxLen)
}

func visibleWidth(s string) int {
	width := 0
	for i := 0; i < len(s); {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			i += 2
			for i < len(s) && (s[i] < '@' || s[i] > '~') {
				i++
			}
			if i < len(s) {
				i++
			}
			continue
		}
		_, size := utf8.DecodeRuneInString(s[i:])
		width++
		i += size
	}
	return width
}
