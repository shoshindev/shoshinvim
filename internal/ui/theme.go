package ui

import (
	"image/color"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
)

// theme is a Kanagawa-inspired palette. Text uses the terminal's own foreground so
// it stays readable on any background; accents adapt to light and dark terminals.
type theme struct {
	plain                               bool
	accent, accent2, ok, warn, bad, dim color.Color
	border                              color.Color
}

func newTheme(dark, plain bool) theme {
	pick := lipgloss.LightDark(dark)
	return theme{
		plain:   plain,
		accent:  pick(lipgloss.Color("#B35B79"), lipgloss.Color("#D27E99")),
		accent2: pick(lipgloss.Color("#4D699B"), lipgloss.Color("#7E9CD8")),
		ok:      pick(lipgloss.Color("#6F894E"), lipgloss.Color("#98BB6C")),
		warn:    pick(lipgloss.Color("#77713F"), lipgloss.Color("#E6C384")),
		bad:     pick(lipgloss.Color("#C84053"), lipgloss.Color("#E46876")),
		dim:     pick(lipgloss.Color("#8A8980"), lipgloss.Color("#727169")),
		border:  pick(lipgloss.Color("#C9CBD1"), lipgloss.Color("#54546D")),
	}
}

func (t theme) fg(c color.Color) lipgloss.Style {
	if t.plain {
		return lipgloss.NewStyle()
	}
	return lipgloss.NewStyle().Foreground(c)
}

func (t theme) Accent(s string) string { return t.fg(t.accent).Bold(true).Render(s) }
func (t theme) Blue(s string) string   { return t.fg(t.accent2).Render(s) }
func (t theme) Ok(s string) string     { return t.fg(t.ok).Render(s) }
func (t theme) Warn(s string) string   { return t.fg(t.warn).Render(s) }
func (t theme) Bad(s string) string    { return t.fg(t.bad).Render(s) }
func (t theme) Bold(s string) string   { return lipgloss.NewStyle().Bold(true).Render(s) }
func (t theme) Dim(s string) string {
	if t.plain {
		return s
	}
	return lipgloss.NewStyle().Foreground(t.dim).Render(s)
}

// Badge renders a small pill such as "recommended".
func (t theme) Badge(s string, c color.Color) string {
	if t.plain {
		return "(" + s + ")"
	}
	return lipgloss.NewStyle().Foreground(c).Render("◆ " + s)
}

// Gradient colors each rune between the two accents.
func (t theme) Gradient(s string) string {
	runes := []rune(s)
	if t.plain || len(runes) < 2 {
		return t.Bold(s)
	}
	colors := lipgloss.Blend1D(len(runes), t.accent, t.accent2)
	var b strings.Builder
	for i, r := range runes {
		b.WriteString(lipgloss.NewStyle().Foreground(colors[i]).Bold(true).Render(string(r)))
	}
	return b.String()
}

// Swatch previews a color scheme: its syntax colors as dots on its own background
// (the first color), like a tiny editor window.
func (t theme) Swatch(colors []string) string {
	if len(colors) < 2 {
		return ""
	}
	bg := lipgloss.Color(colors[0])
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Background(bg).Render(" "))
	for _, c := range colors[1:] {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Background(bg).Render("●"))
	}
	b.WriteString(lipgloss.NewStyle().Background(bg).Render(" "))
	return b.String()
}

func (t theme) Panel(width int, body string) string {
	style := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 3).Width(width)
	if !t.plain {
		style = style.BorderForeground(t.border)
	}
	return style.Render(body)
}

func (t theme) Rule(width int) string { return t.Dim(strings.Repeat("─", max(0, width))) }

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
