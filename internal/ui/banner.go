package ui

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// octopus is Squib's mascot as pixel art. Every pixel is TWO characters wide
// ("██", "▓▓", "▒▒", "░░" or two spaces), and the shade of the block selects
// the color:
//
//	█ outline  ▓ dark shade  ▒ mid shade  ░ highlight
//
// It is not printed as-is: octopusPixels turns it into a pixel grid and
// renderOctopus draws it with half blocks (▀), packing 2x2 characters into a
// single terminal cell. That shrinks it from 60x32 to 30x16 without losing
// detail.
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

// Letters in the "ANSI Shadow" figlet style. Each letter has 6 rows and all
// rows of a letter have the same width, so they can be concatenated safely.
var (
	letterS = []string{"███████╗", "██╔════╝", "███████╗", "╚════██║", "███████║", "╚══════╝"}
	letterQ = []string{" ██████╗ ", "██╔═══██╗", "██║   ██║", "██║▄▄ ██║", "╚██████╔╝", " ╚══▀▀═╝ "}
	letterU = []string{"██╗   ██╗", "██║   ██║", "██║   ██║", "██║   ██║", "╚██████╔╝", " ╚═════╝ "}
	letterI = []string{"██╗", "██║", "██║", "██║", "██║", "╚═╝"}
	letterB = []string{"██████╗ ", "██╔══██╗", "██████╔╝", "██╔══██╗", "██████╔╝", "╚═════╝ "}
)

// logo returns the word SQUIB built from the letters above.
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

// Banner renders the startup screen: the small octopus on the left and, next
// to it, the SQUIB logo (blue -> cyan -> white gradient), a tagline and the
// model name, vertically centered.
func Banner(model string) string {
	left := renderOctopus()
	right := logo()
	logoWidth := maxWidth(right)

	// Right column: logo + empty line + tagline + model.
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
				b.WriteString(paintBold(colorWhite, "Author:") + paint(colorRed, "0xn4cho"))
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

// ---- Octopus rendering ----

// octopusPixels converts the art into a grid of shades (one rune per pixel,
// ' ' means transparent), taking the first character of every 2-char pixel.
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
	// Pad every row to the same width so the art stays rectangular.
	for y := range grid {
		for len(grid[y]) < cols {
			grid[y] = append(grid[y], ' ')
		}
	}
	return grid
}

func octopusWidth() int { return len(octopusPixels()[0]) }

// shadeColor maps a shading block to a palette color. Mid tones shift from
// cyan (head) to blue (tentacles) depending on the pixel row.
func shadeColor(shade rune, y, height int) rgb {
	switch shade {
	case '█':
		return colorDeepBlue
	case '▓':
		return colorBlue
	case '░':
		return colorWhite
	default: // '▒'
		return gradient([]rgb{colorCyan, colorBlue}, 0.6*float64(y)/float64(height-1))
	}
}

// renderOctopus draws two pixel rows per terminal line using the upper half
// block '▀': its foreground paints the top pixel and its background paints
// the bottom one.
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

// halfBlockCell renders one terminal cell holding two vertical pixels.
func halfBlockCell(top, bot rune, y, height int) string {
	hasTop, hasBot := top != ' ', bot != ' '

	if !enabled { // no colors: use the shapes only
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

// ---- Logo rendering ----

// paintLogoLine colors a row of the logo: solid blocks get a horizontal
// blue -> cyan -> white gradient, the "shadow" box-drawing chars stay deep blue.
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
		default: // ╗ ╔ ═ ╝ ║ ╚
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
