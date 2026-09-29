package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// logoGlyphs is a block font for the welcome wordmark, one terminal cell per pixel.
var logoGlyphs = map[rune][]string{
	'S': {
		".#####",
		"##....",
		".####.",
		"....##",
		"#####.",
	},
	'H': {
		"##..##",
		"##..##",
		"######",
		"##..##",
		"##..##",
	},
	'O': {
		".####.",
		"##..##",
		"##..##",
		"##..##",
		".####.",
	},
	'I': {
		"##",
		"##",
		"##",
		"##",
		"##",
	},
	'N': {
		"###..##",
		"####.##",
		"##.####",
		"##..###",
		"##...##",
	},
	'V': {
		"##...##",
		"##...##",
		"##...##",
		".##.##.",
		"..###..",
	},
	'M': {
		"###..###",
		"########",
		"##.##.##",
		"##....##",
		"##....##",
	},
}

const logoWord, logoRows = "SHOSHINVIM", 5

// logoWidth is the wordmark's width in terminal columns.
func logoWidth() int {
	w := len(logoWord) - 1
	for _, r := range logoWord {
		w += len(logoGlyphs[r][0])
	}
	return w
}

// Logo draws the wordmark in horizontal bands of color that fade toward the
// terminal background, one band per row. Filled cells are painted as background
// color, which terminals extend over the whole cell, so rows join without the
// seams that block characters leave under extra line spacing.
func (t theme) Logo() []string {
	lines := make([]string, logoRows)
	for y := range lines {
		var row strings.Builder
		for i, r := range logoWord {
			if i > 0 {
				row.WriteByte('.')
			}
			row.WriteString(logoGlyphs[r][y])
		}
		pixels := row.String()
		if t.plain {
			lines[y] = strings.NewReplacer("#", "█", ".", " ").Replace(pixels)
			continue
		}
		fill := lipgloss.NewStyle().Background(t.logo[y])
		var b strings.Builder
		for pixels != "" {
			n := len(pixels) - len(strings.TrimLeft(pixels, pixels[:1]))
			if pixels[0] == '#' {
				b.WriteString(fill.Render(strings.Repeat(" ", n)))
			} else {
				b.WriteString(strings.Repeat(" ", n))
			}
			pixels = pixels[n:]
		}
		lines[y] = b.String()
	}
	return lines
}
