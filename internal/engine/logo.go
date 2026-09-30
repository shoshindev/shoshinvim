package engine

import "strings"

// The ShoshinVim wordmark, drawn by the setup's welcome screen and by the start
// screens of generated configurations.

// logoGlyphs is a block font for the wordmark, one terminal cell per pixel.
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

const logoWord = "SHOSHINVIM"

// LogoMark is the one-line wordmark for spaces too narrow for LogoPixels.
const LogoMark = "S H O S H I N V I M"

// LogoPixels returns the wordmark's rows: '#' for a filled cell and '.' for an
// empty one, with an empty column between letters.
func LogoPixels() []string {
	rows := make([]string, len(logoGlyphs['S']))
	for y := range rows {
		letters := make([]string, 0, len(logoWord))
		for _, r := range logoWord {
			letters = append(letters, logoGlyphs[r][y])
		}
		rows[y] = strings.Join(letters, ".")
	}
	return rows
}

// LogoColors are the wordmark's row colors, shading the interface's pink accent into
// its blue one; dark backgrounds get the lighter pair. internal/ui checks that they
// match its theme.
func LogoColors(dark bool) []string {
	if dark {
		return []string{"#D27E99", "#C286A9", "#B08EB8", "#9A95C8", "#7E9CD8"}
	}
	return []string{"#B35B79", "#9F6081", "#88648A", "#6E6793", "#4D699B"}
}
