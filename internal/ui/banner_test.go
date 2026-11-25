package ui

import (
	"fmt"
	"strings"
	"testing"
)

func TestBannerShowsSquibCodeStartupDetails(t *testing.T) {
	previous := enabled
	enabled = false
	t.Cleanup(func() { enabled = previous })

	banner := Banner("mock-model")
	for _, want := range []string{
		"SC",
		"SquibCode",
		"v0.1",
		"experimental agent harness · local tools + human approval",
		"Model",
		"mock-model",
		"Workspace",
		"Author",
		"github.com/0xn4cho",
		"Tools",
		"4 available",
		"Approval",
		"Human approval",
		"Welcome to SquibCode!",
		"QUICK COMMANDS",
		"/compact",
		"Compact context",
		"exit",
		"Exit SquibCode",
	} {
		if !strings.Contains(banner, want) {
			t.Errorf("banner does not contain %q:\n%s", want, banner)
		}
	}
	for _, unwanted := range []string{"READY", "/help", "/tools", "/status", "Recent activity", "Welcome back", "Capacidades", "█", "▀", "▄"} {
		if strings.Contains(banner, unwanted) {
			t.Errorf("banner contains unwanted content %q:\n%s", unwanted, banner)
		}
	}

	for _, line := range strings.Split(strings.TrimSpace(banner), "\n") {
		if strings.HasPrefix(line, "╭") || strings.HasPrefix(line, "│") || strings.HasPrefix(line, "╰") {
			if got, want := visibleWidth(line), bannerWidth+2; got != want {
				t.Errorf("bordered line %q has visibleWidth %d, want %d", line, got, want)
			}
		}
	}
}

func TestWelcomeUsesSquibCodeMessage(t *testing.T) {
	previous := enabled
	enabled = false
	t.Cleanup(func() { enabled = previous })

	if got := Welcome(); !strings.Contains(got, "Welcome to SquibCode!") {
		t.Errorf("Welcome() = %q, want a SquibCode welcome message", got)
	}
}

func TestPromptIsMinimal(t *testing.T) {
	previous := enabled
	enabled = false
	t.Cleanup(func() { enabled = previous })

	if got, want := Prompt(), "> "; got != want {
		t.Errorf("Prompt() = %q, want %q", got, want)
	}
}

func TestShortenPathKeepsWindowsVolume(t *testing.T) {
	got := shortenPath(`C:\Users\yakon\OneDrive\Escritorio\Squib`, 34)
	if !strings.HasPrefix(got, `C:\…\`) || !strings.HasSuffix(got, `\Squib`) {
		t.Errorf("shortenPath() = %q, want a path with volume and final directory", got)
	}
}

func TestBannerLayoutAndColors(t *testing.T) {
	previous := enabled
	enabled = true
	t.Cleanup(func() { enabled = previous })

	banner := Banner("openai/gpt-oss-120b")

	wantTokens := []string{
		fmt.Sprintf("\x1b[1;38;2;%d;%d;%dm  SC  \x1b[0m", colorTitleBlue.r, colorTitleBlue.g, colorTitleBlue.b),
		fmt.Sprintf("\x1b[1;38;2;%d;%d;%dmSquibCode\x1b[0m", colorTitleBlue.r, colorTitleBlue.g, colorTitleBlue.b),
		fmt.Sprintf("\x1b[38;2;%d;%d;%dmv0.1\x1b[0m", colorSecondary.r, colorSecondary.g, colorSecondary.b),
		fmt.Sprintf("\x1b[38;2;%d;%d;%dm╭", colorBoxBorder.r, colorBoxBorder.g, colorBoxBorder.b),
		fmt.Sprintf("\x1b[38;2;%d;%d;%dmopenai/gpt-oss-120b\x1b[0m", colorData.r, colorData.g, colorData.b),
		fmt.Sprintf("\x1b[38;2;%d;%d;%dmHuman approval\x1b[0m", colorGreen.r, colorGreen.g, colorGreen.b),
		fmt.Sprintf("\x1b[1;38;2;%d;%d;%dmWelcome to SquibCode!\x1b[0m", colorData.r, colorData.g, colorData.b),
		fmt.Sprintf("\x1b[1;38;2;%d;%d;%dmQUICK COMMANDS\x1b[0m", colorTitleBlue.r, colorTitleBlue.g, colorTitleBlue.b),
	}
	for _, token := range wantTokens {
		if !strings.Contains(banner, token) {
			t.Errorf("colored banner missing ANSI sequence %q:\n%q", token, banner)
		}
	}

	if strings.Contains(banner, "READY") {
		t.Errorf("colored banner must not contain READY:\n%s", banner)
	}

	boxPrefix := fmt.Sprintf("\x1b[38;2;%d;%d;%dm", colorBoxBorder.r, colorBoxBorder.g, colorBoxBorder.b)
	panelPrefix := fmt.Sprintf("\x1b[38;2;%d;%d;%dm", colorPanelBorder.r, colorPanelBorder.g, colorPanelBorder.b)
	for _, line := range strings.Split(strings.TrimSpace(banner), "\n") {
		if strings.HasPrefix(line, boxPrefix) || strings.HasPrefix(line, panelPrefix) {
			if got, want := visibleWidth(line), bannerWidth+2; got != want {
				t.Errorf("colored bordered line %q has visibleWidth %d, want %d", line, got, want)
			}
		}
	}

	prompt := Prompt()
	if !strings.Contains(prompt, "Enter a task...") || !strings.Contains(prompt, "Ctrl+C to exit") {
		t.Errorf("colored Prompt() = %q, want placeholder and Ctrl+C hint", prompt)
	}
	if got, want := visibleWidth(strings.TrimSuffix(prompt, "\x1b[5G")), bannerWidth+2; got != want {
		t.Errorf("colored Prompt() visibleWidth = %d, want %d", got, want)
	}
}
