package ui

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

const (
	bannerVersion = "v1.0.0"
	leftPanel     = 58
	rightPanel    = 22
	bannerWidth   = leftPanel + rightPanel + 1
)

// Banner keeps the identity and useful startup details together while leaving
// the terminal focused on the next prompt.
func Banner(model string) string {
	directory, err := os.Getwd()
	if err != nil {
		directory = "directorio no disponible"
	}

	left := append(orcaMark(), "",
		info("Modelo", model, leftPanel),
		info("Directorio", shortenPath(directory, leftPanel-utf8.RuneCountInString("  Directorio: ")), leftPanel),
		info("Autor", "github.com/0xn4cho", leftPanel),
	)
	right := capabilities()
	height := max(len(left), len(right))

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(border("┌", "─", "┐"))
	b.WriteString("\n")
	b.WriteString(singleLine("  "+paintBold(colorCyan, "SquibCode")+" "+paint(colorGray, bannerVersion), 0))
	b.WriteString("\n")
	b.WriteString(border("├", "─", "┤"))
	b.WriteString("\n")

	for i := 0; i < height; i++ {
		leftLine, rightLine := "", ""
		if i < len(left) {
			leftLine = left[i]
		}
		if i < len(right) {
			rightLine = right[i]
		}
		b.WriteString(splitLine(leftLine, rightLine))
		b.WriteString("\n")
	}

	b.WriteString(border("└", "─", "┘"))
	b.WriteString("\n")
	return b.String()
}

func Welcome() string {
	return paintBold(colorWhite, "  Bienvenido a SquibCode") + paint(colorGray, " — listo para programar y resolver tareas DevOps.")
}

func capabilities() []string {
	return []string{
		"",
		"  " + paintBold(colorYellow, "Capacidades"),
		"  " + paintBold(colorWhite, "• Coding"),
		"  " + paintBold(colorCyan, "• Docker"),
		"  " + paintBold(colorWhite, "• Kubernetes"),
	}
}

var orcaPixels = []string{
	".............................BB.....................",
	"............................BBB.....................",
	"...........................BBBB.....................",
	"..........................BBBBBB....................",
	"........................BBBBBBBB....................",
	"......................BBBBBBBBBBB...................",
	"............BBBBBBBBBBBBBBBBBBBBBBBBB............BBB",
	"........BBBBBBBBBBBBBBBBBBBBBBBGGGGGBBBB.......BBBBB",
	".....BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB...BBBBBB.",
	"...BBBBBBBBWWWWWBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB....",
	"..BBBBBBBBBBWWWWWBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB.....",
	".BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBWWWWBBBBBBBBBBB......",
	".BWWWWWWWBBBBBBBBBBBBBBBBBBBWWWWWWWWWBBBB....BBBBB..",
	"..BWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWWB........BBBBBB",
	"....BWWWWWWWWWWWWWWWWWWWWWWWWWWWWB..............BBBB",
	".......BWWWWWWBBBBBWWWWWWWWWB.......................",
	"...............BBBBB................................",
	".................BBB................................",
}

var (
	orcaBack  = rgb{18, 42, 100}
	orcaBelly = rgb{40, 95, 215}
)

func orcaMark() []string {
	height := len(orcaPixels)
	margin := strings.Repeat(" ", max(0, (leftPanel-len(orcaPixels[0]))/2))
	lines := make([]string, 0, (height+1)/2)
	for y := 0; y < height; y += 2 {
		var b strings.Builder
		b.WriteString(margin)
		for x := range orcaPixels[y] {
			top := orcaPixels[y][x]
			bottom := byte('.')
			if y+1 < height {
				bottom = orcaPixels[y+1][x]
			}
			b.WriteString(orcaCell(top, bottom, y, height))
		}
		lines = append(lines, strings.TrimRight(b.String(), " "))
	}
	return lines
}

func orcaColor(pixel byte, y, height int) (rgb, bool) {
	switch pixel {
	case 'B':
		return gradient([]rgb{orcaBack, orcaBelly}, float64(y)/float64(height-1)), true
	case 'W':
		return colorWhite, true
	case 'G':
		return colorGray, true
	default:
		return rgb{}, false
	}
}

func orcaCell(top, bottom byte, y, height int) string {
	if !enabled {
		hasTop, hasBottom := top == 'B' || top == 'G', bottom == 'B' || bottom == 'G'
		switch {
		case hasTop && hasBottom:
			return "█"
		case hasTop:
			return "▀"
		case hasBottom:
			return "▄"
		default:
			return " "
		}
	}

	topColor, hasTop := orcaColor(top, y, height)
	bottomColor, hasBottom := orcaColor(bottom, y+1, height)
	switch {
	case hasTop && hasBottom:
		return fmt.Sprintf("\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀\x1b[0m",
			topColor.r, topColor.g, topColor.b, bottomColor.r, bottomColor.g, bottomColor.b)
	case hasTop:
		return paint(topColor, "▀")
	case hasBottom:
		return paint(bottomColor, "▄")
	default:
		return " "
	}
}

func info(label, value string, width int) string {
	prefix := "  " + label + ": "
	value = shorten(value, width-utf8.RuneCountInString(prefix))
	return prefix + value
}

func border(left, fill, right string) string {
	return paint(colorDeepBlue, left+strings.Repeat(fill, bannerWidth)+right)
}

func singleLine(content string, contentWidth int) string {
	if contentWidth == 0 {
		contentWidth = visibleWidth(content)
	}
	return paint(colorDeepBlue, "│") + content + strings.Repeat(" ", bannerWidth-contentWidth) + paint(colorDeepBlue, "│")
}

func splitLine(left, right string) string {
	return paint(colorDeepBlue, "│") + paintInfo(left) + strings.Repeat(" ", leftPanel-visibleWidth(left)) +
		paint(colorDeepBlue, "│") + right + strings.Repeat(" ", rightPanel-visibleWidth(right)) + paint(colorDeepBlue, "│")
}

func paintInfo(line string) string {
	if strings.HasPrefix(line, "  Modelo: ") {
		return "  " + paintBold(colorCyan, "Modelo:") + " " + paint(colorWhite, strings.TrimPrefix(line, "  Modelo: "))
	}
	if strings.HasPrefix(line, "  Directorio: ") {
		return "  " + paintBold(colorCyan, "Directorio:") + " " + paint(colorWhite, strings.TrimPrefix(line, "  Directorio: "))
	}
	if strings.HasPrefix(line, "  Autor: ") {
		return "  " + paintBold(colorCyan, "Autor:") + " " + paint(colorRed, strings.TrimPrefix(line, "  Autor: "))
	}
	return line
}

func shorten(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return "…" + string(runes[len(runes)-(max-1):])
}

func shortenPath(path string, max int) string {
	if utf8.RuneCountInString(path) <= max {
		return path
	}
	if volume := strings.Index(path, `\`); volume > 0 && max > volume+4 {
		prefix := path[:volume+1] + "…\\"
		tailLength := max - utf8.RuneCountInString(prefix)
		runes := []rune(path)
		return prefix + string(runes[len(runes)-tailLength:])
	}
	return shorten(path, max)
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
