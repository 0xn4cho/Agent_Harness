package ui

import (
	"strings"
	"testing"
)

func TestRenderBannerWidePlainAndTokyoNight(t *testing.T) {
	previous := enabled
	t.Cleanup(func() { enabled = previous })

	info := StartupInfo{
		Workspace: "~/project",
		RulesFile: "SQUIB.md",
		Models:    "20b / 120b",
	}

	enabled = false
	plain := RenderBanner(80, info)
	for _, want := range []string{
		"███████╗",
		"workspace ~/project",
		"reglas SQUIB.md ✓",
		"modelos 20b / 120b",
		"/help · /roles · /compact · /clear · exit",
		"────",
	} {
		if !strings.Contains(plain, want) {
			t.Errorf("plain wide banner missing %q:\n%s", want, plain)
		}
	}
	if strings.Contains(plain, "\033[") {
		t.Errorf("plain wide banner must not contain ANSI escapes when colors disabled")
	}

	artLines := strings.Split(strings.Trim(AnimatedArt(-1), "\n"), "\n")
	if len(artLines) != 6 {
		t.Fatalf("expected 6 ASCII art rows, got %d", len(artLines))
	}
	for i, line := range artLines {
		if got := visibleWidth(line); got != BigBannerWidth {
			t.Errorf("ASCII art row %d has visibleWidth %d, want %d: %q", i+1, got, BigBannerWidth, line)
		}
	}

	enabled = true
	colored := RenderBanner(80, info)
	for _, seq := range []string{
		"122;162;247",
		"59;66;97",
		TokyoGreen,
		TokyoDim,
		Reset,
	} {
		if !strings.Contains(colored, seq) {
			t.Errorf("colored wide banner missing Tokyo Night sequence %q", seq)
		}
	}

	shined := AnimatedArt(12)
	if shined == AnimatedArt(-1) {
		t.Errorf("AnimatedArt(12) should differ from settled AnimatedArt(-1)")
	}
	shinedLines := strings.Split(strings.TrimSuffix(shined, "\n"), "\n")
	if len(shinedLines) != 6 {
		t.Fatalf("expected 6 shined ASCII art rows, got %d", len(shinedLines))
	}
	for i, line := range shinedLines {
		if got := visibleWidth(line); got != BigBannerWidth {
			t.Errorf("shined row %d has visibleWidth %d, want %d", i+1, got, BigBannerWidth)
		}
	}
}

func TestBannerTextNarrowAndNonTTYFallback(t *testing.T) {
	previous := enabled
	enabled = false
	t.Cleanup(func() { enabled = previous })

	for _, width := range []int{0, 40, 60, bigBannerMinWidth - 1} {
		out := BannerText(width, "20b / 120b")
		if !strings.Contains(out, "SQUIBCODE") {
			t.Errorf("width %d fallback missing SQUIBCODE:\n%s", width, out)
		}
		if !strings.Contains(out, "modelos 20b / 120b") {
			t.Errorf("width %d fallback missing models:\n%s", width, out)
		}
		if strings.Contains(out, "█") {
			t.Errorf("width %d fallback must not contain block ASCII art:\n%s", width, out)
		}
	}
}

func TestSummarizeToolArgsAndToolResult(t *testing.T) {
	previous := enabled
	enabled = false
	t.Cleanup(func() { enabled = previous })

	cases := []struct {
		name string
		args string
		want string
	}{
		{"read_file", `{"path":"notes/tasks.md"}`, "notes/tasks.md"},
		{"write_file", `{"path":"notes/ideas.md","content":"nota"}`, "notes/ideas.md"},
		{"bash", `{"command":"git status"}`, "git status"},
		{"find_files", `{"path":"C:\\Workspace","pattern":"*.md"}`, `*.md in C:\Workspace`},
		{"copy_path", `{"source":"a.txt","destination":"b.txt"}`, "a.txt → b.txt"},
		{"devops_status", `{"target":"docker"}`, "docker"},
		{"custom", "raw \x1b[31minput", "raw input"},
	}
	for _, tc := range cases {
		if got := SummarizeToolArgs(tc.name, tc.args); got != tc.want {
			t.Errorf("SummarizeToolArgs(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}

	okLine := ToolResult("├─", "read_file", `{"path":"notes/tasks.md"}`, true)
	if !strings.Contains(okLine, "├─") || !strings.Contains(okLine, "read_file") ||
		!strings.Contains(okLine, "notes/tasks.md") || !strings.Contains(okLine, "✓") {
		t.Errorf("unexpected ToolResult(ok): %q", okLine)
	}
	failLine := ToolResult("└─", "bash", `{"command":"git status"}`, false)
	if !strings.Contains(failLine, "└─") || !strings.Contains(failLine, "git status") || !strings.Contains(failLine, "✗") {
		t.Errorf("unexpected ToolResult(fail): %q", failLine)
	}
}

func TestPromptModelSwitchAndHighlightVerdict(t *testing.T) {
	previous := enabled
	t.Cleanup(func() { enabled = previous })

	enabled = false
	if got := Prompt(); got != "❯ " {
		t.Errorf("Prompt() = %q, want %q", got, "❯ ")
	}
	if got := HighlightVerdict("Todo ok\nAPPROVED"); got != "Todo ok\nAPPROVED" {
		t.Errorf("HighlightVerdict with colors disabled changed text: %q", got)
	}

	longArg := strings.Repeat("á", 100)
	tc := ToolCall("read_file", "\x1b[31m"+longArg)
	if !strings.Contains(tc, "├─") || !strings.Contains(tc, "read_file") {
		t.Errorf("ToolCall() missing expected formatting: %q", tc)
	}
	if strings.Contains(tc, "\x1b[31m") {
		t.Errorf("ToolCall() did not sanitize ANSI escape from args: %q", tc)
	}
	if !strings.HasSuffix(tc, "…") {
		t.Errorf("ToolCall() expected rune truncation suffix '…': %q", tc)
	}

	enabled = true
	if got := Prompt(); got != PromptColor+"❯ "+Reset {
		t.Errorf("colored Prompt() = %q, want %q", got, PromptColor+"❯ "+Reset)
	}

	msLeader := FormatModelSwitch("leader", "openai/gpt-oss-20b")
	if !strings.Contains(msLeader, TokyoBlue) || !strings.Contains(msLeader, "● leader") {
		t.Errorf("FormatModelSwitch(leader) = %q, expected TokyoBlue", msLeader)
	}
	msImpl := FormatModelSwitch("implementer", "openai/gpt-oss-120b")
	if !strings.Contains(msImpl, TokyoPurple) || !strings.Contains(msImpl, "● implementer") {
		t.Errorf("FormatModelSwitch(implementer) = %q, expected TokyoPurple", msImpl)
	}
	msRev := FormatModelSwitch("reviewer", "openai/gpt-oss-20b")
	if !strings.Contains(msRev, TokyoGreen) || !strings.Contains(msRev, "● reviewer") {
		t.Errorf("FormatModelSwitch(reviewer) = %q, expected TokyoGreen", msRev)
	}

	if got := HighlightVerdict("Ok\nAPPROVED"); !strings.Contains(got, TokyoGreen+"APPROVED"+Reset) {
		t.Errorf("HighlightVerdict(APPROVED) = %q", got)
	}
	if got := HighlightVerdict("Fix\nCHANGES_REQUESTED"); !strings.Contains(got, TokyoAmber+"CHANGES_REQUESTED"+Reset) {
		t.Errorf("HighlightVerdict(CHANGES_REQUESTED) = %q", got)
	}
}
