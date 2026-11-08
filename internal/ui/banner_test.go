package ui

import (
	"strings"
	"testing"
)

func TestBannerShowsSquibCodeStartupDetails(t *testing.T) {
	previous := enabled
	enabled = false
	t.Cleanup(func() { enabled = previous })

	banner := Banner("mock-model")
	for _, want := range []string{"SquibCode", "v1.0.0", "Modelo: mock-model", "Directorio:", "Autor: github.com/0xn4cho", "Capacidades", "Coding", "Docker", "Kubernetes"} {
		if !strings.Contains(banner, want) {
			t.Errorf("banner does not contain %q:\n%s", want, banner)
		}
	}
	if strings.Contains(banner, "Recent activity") || strings.Contains(banner, "Welcome back") {
		t.Errorf("banner contains unwanted welcome content:\n%s", banner)
	}
}

func TestWelcomeUsesSquibCodeMessage(t *testing.T) {
	previous := enabled
	enabled = false
	t.Cleanup(func() { enabled = previous })

	if got := Welcome(); !strings.Contains(got, "Bienvenido a SquibCode") {
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

func TestOrcaMarkIsCompactAndFitsThePanel(t *testing.T) {
	previous := enabled
	enabled = false
	t.Cleanup(func() { enabled = previous })

	for _, row := range orcaPixels {
		if len(row) != len(orcaPixels[0]) {
			t.Fatalf("orca pixel rows must share the same width: %q", row)
		}
	}

	mark := orcaMark()
	if want := (len(orcaPixels) + 1) / 2; len(mark) != want {
		t.Fatalf("orcaMark() returned %d lines, want %d", len(mark), want)
	}
	for _, line := range mark {
		if visibleWidth(line) > leftPanel {
			t.Errorf("orca line exceeds panel width %d: %q", leftPanel, line)
		}
	}
	if !strings.Contains(strings.Join(mark, "\n"), "█") {
		t.Errorf("orca mark has no filled body cells: %#v", mark)
	}
}
