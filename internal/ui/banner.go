package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

var octopus = []string{
	`                                  ▒▒▒▒▓▓▓▓██`,
	`                              ▓▓▒▒▒▒▒▒▒▒▒▒▒▒▓▓██`,
	`                          ██▒▒▒▒▒▒░░░░░░▒▒▒▒▒▒▒▒████`,
	`                        ██▒▒▒▒▒▒░░░░░░▒▒▒▒▒▒▒▒▒▒▒▒▒▒██`,
	`                      ██▒▒▒▒░░▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒██`,
	`                      ██▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒██`,
	`                    ██▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▓▓██`,
	`                    ██▓▓▒▒▒▒▒▒▒▒▒▒▓▓▓▓▒▒▒▒▒▒▒▒▒▒▒▒░░▒▒▒▒██`,
	`                    ██░░▓▓▒▒▒▒▒▒▓▓▒▒░░▓▓▒▒▒▒▒▒░░░░▒▒▒▒▒▒▓▓██`,
	`                    ██▒▒▓▓▒▒▒▒▓▓▒▒▒▒▒▒▓▓▒▒▒▒░░▒▒▒▒▒▒▒▒▒▒▓▓██`,
	`                  ██░░▓▓▓▓▒▒▒▒▓▓▒▒▒▒▒▒▓▓░░▒▒▒▒▓▓▒▒▒▒▒▒▒▒▓▓██`,
	`      ████        ██▒▒░░▒▒▒▒▒▒▒▒▓▓▓▓▓▓▒▒▒▒▓▓▒▒▓▓▓▓▒▒▒▒▓▓▓▓██`,
	`    ████████        ██▒▒▓▓▒▒▒▒▒▒▒▒░░░░▒▒▓▓▒▒▓▓▓▓██▓▓▓▓████`,
	`  ██████    ██      ██▓▓▓▓▒▒▓▓▒▒▓▓▓▓▓▓▓▓▒▒▓▓▓▓▓▓██████`,
	`  ████                ██▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓██▓▓▓▓██████`,
	`██▓▓██                  ██▓▓██▓▓▓▓▓▓▓▓▓▓██▓▓▒▒▓▓▒▒██`,
	`██▓▓▓▓                ██▓▓▓▓▓▓██▓▓▒▒▓▓▓▓▓▓██▓▓▒▒░░▒▒██`,
	`██▓▓▒▒▓▓          ████▒▒▒▒▒▒▓▓██▒▒▓▓▒▒▓▓▒▒▓▓██▓▓▒▒▒▒▒▒`,
	`██▓▓▒▒▒▒▓▓████▓▓▓▓▒▒▒▒░░▒▒▓▓██▓▓▒▒▒▒▓▓▓▓▒▒▒▒▓▓██▒▒▒▒▒▒▓▓`,
	`  ██▓▓▒▒▒▒▒▒▒▒▒▒░░▒▒▓▓██████▓▓▒▒▒▒▓▓████▓▓▒▒▒▒██▓▓▒▒▒▒▓▓`,
	`    ██▓▓▒▒▒▒▓▓████▓▓▓▓▒▒░░▒▒▒▒▒▒▓▓████▓▓██▒▒▒▒▒▒██▒▒░░▒▒`,
	`        ████▓▓▒▒▒▒▒▒░░▒▒▒▒▒▒▓▓████▓▓▓▓    ██▒▒▒▒██▒▒░░▒▒▓▓`,
	`      ████▓▓▒▒▒▒▒▒▒▒▒▒▓▓██████▓▓▓▓▓▓      ██▒▒▒▒██▒▒▒▒▒▒▓▓`,
	`  ██▓▓▓▓██▒▒▒▒▓▓██████████▓▓▓▓▓▓▓▓          ▓▓▒▒██▒▒▒▒▒▒██`,
	`██▓▓▓▓██▒▒▓▓████      ██▓▓▓▓▓▓▓▓      ██    ▓▓▒▒▓▓▒▒▒▒▒▒██`,
	`██▓▓▓▓██▓▓██        ██▓▓▓▓▓▓          ██▓▓  ██▒▒▓▓▒▒▒▒▓▓`,
	`██▓▓  ██▓▓          ██▓▓▓▓        ██  ██▓▓▓▓▓▓██▓▓▒▒▒▒██`,
	`▓▓▓▓    ██        ██▓▓▓▓        ██      ██▓▓▓▓██░░▒▒▒▒██`,
	`  ▓▓      ██      ██▓▓▓▓        ████      ████▓▓▒▒▒▒▓▓`,
	`    ▓▓            ██▓▓          ██▓▓████▓▓▓▓▒▒▒▒▒▒▓▓██`,
	`                    ██▓▓          ██▓▓▒▒▒▒▒▒▒▒▓▓████`,
	`                                      ▓▓▓▓▓▓▓▓`,
}

var (
	letterS = []string{"███████╗", "██╔════╝", "███████╗", "╚════██║", "███████║", "╚══════╝"}
	letterQ = []string{" ██████╗ ", "██╔═══██╗", "██║   ██║", "██║▄▄ ██║", "╚██████╔╝", " ╚══▀▀═╝ "}
	letterU = []string{"██╗   ██╗", "██║   ██║", "██║   ██║", "██║   ██║", "╚██████╔╝", " ╚═════╝ "}
	letterI = []string{"██╗", "██║", "██║", "██║", "██║", "╚═╝"}
	letterB = []string{"██████╗ ", "██╔══██╗", "██████╔╝", "██╔══██╗", "██████╔╝", "╚═════╝ "}
)

func logo() []string {
	letters := [][]string{letterS, letterQ, letterU, letterI, letterB}
	rows := make([]string, len(letterS))
	for i := range rows {
		var b strings.Builder
		for _, l := range letters {
			b.WriteString(l[i])
		}
		rows[i] = b.String()
	}
	return rows
}

func Banner(model string) string {
	left := renderOctopus()
	right := logo()
	logoWidth := maxWidth(right)

	right = append(right, "", "tagline", "model", "author")
	top := (len(left) - len(right)) / 2
	if top < 0 {
		top = 0
	}

	var b strings.Builder
	b.WriteString("\n")
	for i, artLine := range left {
		b.WriteString("  ")
		b.WriteString(artLine)
		b.WriteString("    ")

		if j := i - top; j >= 0 && j < len(right) {
			switch {
			case j < len(right)-3:
				b.WriteString(paintLogoLine(right[j], logoWidth))
			case right[j] == "tagline":
				b.WriteString(paintBold(colorWhite, "Coding") + paint(colorBlue, "  ·  ") +
					paintBold(colorCyan, "Docker") + paint(colorBlue, "  ·  ") +
					paintBold(colorWhite, "Kubernetes"))
			case right[j] == "model":
				b.WriteString(Dim("model: ") + paint(colorCyan, model))

			case right[j] == "author":
				b.WriteString(paintBold(colorWhite, "Author: ") + paint(colorRed, "github.com/0xn4cho"))
			}

		}
		b.WriteString("\n")
	}

	width := 2 + octopusWidth() + 4 + logoWidth
	b.WriteString("\n")
	b.WriteString(paint(colorDeepBlue, strings.Repeat("─", width)))
	b.WriteString("\n")
	b.WriteString(Dim("  Escribe tu petición. 'exit' para salir.  ·  Herramientas peligrosas pedirán permiso."))
	b.WriteString("\n")
	return b.String()
}

func octopusPixels() [][]rune {
	cols := 0
	grid := make([][]rune, len(octopus))
	for y, line := range octopus {
		runes := []rune(line)
		for x := 0; x < len(runes); x += 2 {
			grid[y] = append(grid[y], runes[x])
		}
		if len(grid[y]) > cols {
			cols = len(grid[y])
		}
	}
	for y := range grid {
		for len(grid[y]) < cols {
			grid[y] = append(grid[y], ' ')
		}
	}
	return grid
}

func octopusWidth() int { return len(octopusPixels()[0]) }

func shadeColor(shade rune, y, height int) rgb {
	switch shade {
	case '█':
		return colorDeepBlue
	case '▓':
		return colorBlue
	case '░':
		return colorWhite
	default:
		return gradient([]rgb{colorCyan, colorBlue}, 0.6*float64(y)/float64(height-1))
	}
}

func renderOctopus() []string {
	grid := octopusPixels()
	height := len(grid)

	var lines []string
	for y := 0; y < height; y += 2 {
		var b strings.Builder
		for x := range grid[y] {
			topPx := grid[y][x]
			botPx := ' '
			if y+1 < height {
				botPx = grid[y+1][x]
			}
			b.WriteString(halfBlockCell(topPx, botPx, y, height))
		}
		lines = append(lines, b.String())
	}
	return lines
}

func halfBlockCell(top, bot rune, y, height int) string {
	hasTop, hasBot := top != ' ', bot != ' '

	if !enabled {
		switch {
		case hasTop && hasBot:
			return "█"
		case hasTop:
			return "▀"
		case hasBot:
			return "▄"
		default:
			return " "
		}
	}

	switch {
	case hasTop && hasBot:
		t, bt := shadeColor(top, y, height), shadeColor(bot, y+1, height)
		return fmt.Sprintf("\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀\x1b[0m", t.r, t.g, t.b, bt.r, bt.g, bt.b)
	case hasTop:
		return paint(shadeColor(top, y, height), "▀")
	case hasBot:
		return paint(shadeColor(bot, y+1, height), "▄")
	default:
		return " "
	}
}

func paintLogoLine(line string, width int) string {
	if !enabled {
		return line
	}
	stops := []rgb{colorBlue, colorCyan, colorWhite}

	var b strings.Builder
	col := 0
	for _, r := range line {
		switch r {
		case ' ':
			b.WriteRune(r)
		case '█', '▄', '▀':
			b.WriteString(paint(gradient(stops, float64(col)/float64(width-1)), string(r)))
		default:
			b.WriteString(paint(colorDeepBlue, string(r)))
		}
		col++
	}
	return b.String()
}

func maxWidth(lines []string) int {
	w := 0
	for _, l := range lines {
		if n := utf8.RuneCountInString(l); n > w {
			w = n
		}
	}
	return w
}
