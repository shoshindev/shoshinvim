package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/shoshindev/shoshinvim/internal/engine"
)

var logoRows = len(engine.LogoPixels())

// logoWidth is the wordmark's width in terminal columns.
func logoWidth() int { return len(engine.LogoPixels()[0]) }

// Logo draws the wordmark in horizontal bands of color that fade toward the
// terminal background, one band per row. Filled cells are painted as background
// color, which terminals extend over the whole cell, so rows join without the
// seams that block characters leave under extra line spacing.
func (t theme) Logo() []string {
	lines := make([]string, logoRows)
	for y, pixels := range engine.LogoPixels() {
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
