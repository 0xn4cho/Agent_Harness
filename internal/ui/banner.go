package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	bigBannerMinWidth = 73
	BigBannerWidth    = 70
)

const bigBanner = "\n" +
	"███████╗ ██████╗ ██╗   ██╗██╗██████╗  ██████╗ ██████╗ ██████╗ ███████╗\n" +
	"██╔════╝██╔═══██╗██║   ██║██║██╔══██╗██╔════╝██╔═══██╗██╔══██╗██╔════╝\n" +
	"███████╗██║   ██║██║   ██║██║██████╔╝██║     ██║   ██║██║  ██║█████╗  \n" +
	"╚════██║██║▄▄ ██║██║   ██║██║██╔══██╗██║     ██║   ██║██║  ██║██╔══╝  \n" +
	"███████║╚██████╔╝╚██████╔╝██║██████╔╝╚██████╗╚██████╔╝██████╔╝███████╗\n" +
	"╚══════╝ ╚══▀▀═╝  ╚═════╝ ╚═╝╚═════╝  ╚═════╝ ╚═════╝ ╚═════╝ ╚══════╝\n"

type StartupInfo struct {
	Workspace string
	RulesFile string
	Models    string
	MockMode  bool
}

type rgb struct{ r, g, b uint8 }

var (
	bannerGradStart = rgb{122, 162, 247}
	bannerGradEnd   = rgb{125, 207, 255}
	bannerShadow    = rgb{59, 66, 97}
	shineBlockPeak  = rgb{235, 244, 255}
	shineShadowPeak = rgb{154, 178, 232}
	shineWeights    = [...]int{100, 82, 62, 42, 24, 10}
)

func PrintBanner(model string) {
	fmt.Print(Banner(model))
}

func PrintAnimatedBanner(info StartupInfo) {
	width := TermWidth()
	if !enabled || width < bigBannerMinWidth || SuppressSpinner {
		fmt.Print(RenderBanner(width, info))
		return
	}

	fmt.Print("\x1b[?25l")
	defer fmt.Print("\x1b[?25h")

	fmt.Print("\n")
	first := true
	for shineCol := -4; shineCol <= BigBannerWidth+16; shineCol += 5 {
		if !first {
			fmt.Print("\x1b[6A\r")
		}
		first = false
		fmt.Print(AnimatedArt(shineCol))
		time.Sleep(15 * time.Millisecond)
	}
	fmt.Print("\x1b[6A\r")
	fmt.Print(AnimatedArt(-1))
	fmt.Print("\n" + RenderBannerFooter(width, info))
}

func Banner(model string) string {
	return BannerText(TermWidth(), model)
}

func BannerText(width int, model string) string {
	return RenderBanner(width, StartupInfo{Models: model})
}

func RenderBanner(width int, info StartupInfo) string {
	if width >= bigBannerMinWidth {
		return "\n" + AnimatedArt(-1) + "\n" + RenderBannerFooter(width, info)
	}
	return "\n  " + Cyan("SQUIBCODE") + Dim("  ·  coding & DevOps agent") + "\n" +
		RenderBannerFooter(width, info)
}

func RenderBannerFooter(width int, info StartupInfo) string {
	workspace := strings.TrimSpace(info.Workspace)
	if workspace == "" {
		workspace = currentWorkspaceLabel()
	}

	var status strings.Builder
	status.WriteString("  ")
	status.WriteString(Dim("workspace ") + Text(workspace))
	if rules := strings.TrimSpace(info.RulesFile); rules != "" {
		status.WriteString(Dim(" · reglas ") + Text(filepath.Base(rules)) + " " + Success("✓"))
	}
	if models := strings.TrimSpace(info.Models); models != "" {
		status.WriteString(Dim(" · modelos ") + Cyan(models))
	}
	if info.MockMode {
		status.WriteString(Dim(" · ") + Warn("mock"))
	}

	ruleWidth := 68
	if width < bigBannerMinWidth {
		if width <= 0 {
			ruleWidth = 44
		} else {
			ruleWidth = min(max(width-4, 24), 48)
		}
	}

	return status.String() + "\n" +
		"  " + Dim("/help · /roles · /compact · /clear · exit") + "\n" +
		"  " + Border(strings.Repeat("─", ruleWidth)) + "\n\n"
}

func AnimatedArt(shineCol int) string {
	trimmed := strings.Trim(bigBanner, "\n")
	if !enabled {
		return trimmed + "\n"
	}

	var sb strings.Builder
	for row, line := range strings.Split(trimmed, "\n") {
		col := 0
		for _, r := range line {
			if r == ' ' {
				sb.WriteRune(' ')
				col++
				continue
			}
			c, bold := bannerCellColor(r, col, row, shineCol)
			if bold {
				fmt.Fprintf(&sb, "\x1b[1;38;2;%d;%d;%dm%c\x1b[0m", c.r, c.g, c.b, r)
			} else {
				fmt.Fprintf(&sb, "\x1b[38;2;%d;%d;%dm%c\x1b[0m", c.r, c.g, c.b, r)
			}
			col++
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

func bannerCellColor(r rune, col, row, shineCol int) (rgb, bool) {
	isBlock := r == '█' || r == '▄' || r == '▀'
	base := bannerShadow
	peak := shineShadowPeak
	if isBlock {
		base = lerpRGB(bannerGradStart, bannerGradEnd, col, BigBannerWidth-1)
		peak = shineBlockPeak
	}

	if shineCol < 0 {
		return base, isBlock
	}

	effCol := col + row*2
	diff := effCol - shineCol
	var d int
	if diff >= 0 {
		d = diff
	} else {
		d = (-diff + 1) / 2
	}
	if d >= len(shineWeights) {
		return base, isBlock
	}
	return blendRGB(base, peak, shineWeights[d]), isBlock
}

func lerpRGB(a, b rgb, pos, maxPos int) rgb {
	if maxPos <= 0 {
		return a
	}
	if pos <= 0 {
		return a
	}
	if pos >= maxPos {
		return b
	}
	return rgb{
		r: uint8(int(a.r) + (int(b.r)-int(a.r))*pos/maxPos),
		g: uint8(int(a.g) + (int(b.g)-int(a.g))*pos/maxPos),
		b: uint8(int(a.b) + (int(b.b)-int(a.b))*pos/maxPos),
	}
}

func blendRGB(base, target rgb, pct int) rgb {
	return rgb{
		r: uint8(int(base.r) + (int(target.r)-int(base.r))*pct/100),
		g: uint8(int(base.g) + (int(target.g)-int(base.g))*pct/100),
		b: uint8(int(base.b) + (int(target.b)-int(base.b))*pct/100),
	}
}

func currentWorkspaceLabel() string {
	dir, err := os.Getwd()
	if err != nil || dir == "" {
		return "unavailable"
	}
	base := filepath.Base(dir)
	if base != "" && base != "." && base != string(filepath.Separator) && base != "/" {
		return "~/" + base
	}
	return dir
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
