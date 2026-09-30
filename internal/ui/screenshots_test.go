package ui

import (
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

// TestScreenshots draws the wordmark and setup screens as SVG files for the README:
//
//	SHOSHINVIM_SCREENSHOTS=docs/assets go test ./internal/ui -run TestScreenshots -count=1
//
// A relative folder is taken from the repository root. The review screen needs Neovim.
// Screens show a dark terminal, the usual default, so they look the same on GitHub's
// light and dark pages; the wordmark uses the light-terminal accents, which keep enough
// contrast on both.
func TestScreenshots(t *testing.T) {
	dir := os.Getenv("SHOSHINVIM_SCREENSHOTS")
	if dir == "" {
		t.Skip("set SHOSHINVIM_SCREENSHOTS to the folder that receives the images")
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join("..", "..", dir)
	}
	if err := os.MkdirAll(filepath.Join(dir, "screens"), 0o755); err != nil {
		t.Fatal(err)
	}
	// A fresh home so the header reads ~/.config/nvim, as it would for a new user.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	write := func(name string, data []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("wordmark.svg", wordmarkSVG(newTheme(false, false)))

	m, err := New(filepath.Join(home, ".config", "nvim"), "", false) // Dark palette.
	if err != nil {
		t.Fatal(err)
	}
	shoot := func(name string) {
		t.Helper()
		write(filepath.Join("screens", name+".svg"), screenSVG(m.View().Content, m.width, m.height))
	}
	advance := func(id string) {
		t.Helper()
		for i := 0; m.screen != stepScreen || m.question().id != id; i++ {
			if i > 60 {
				t.Fatalf("question %s not reached:\n%s", id, screenText(m))
			}
			press(m, "enter")
		}
	}
	// Wide enough for the section sidebar and the whole shortcut list.
	m.Update(tea.WindowSizeMsg{Width: 110, Height: 32})
	press(m, "enter", "enter") // Start, accept the Editor form.
	advance("appearance.colorscheme")
	shoot("colorscheme")
	advance("keymaps-editor")
	shoot("shortcuts")
	advance("navigation.picker")
	shoot("picker")
	if _, err := exec.LookPath("nvim"); err != nil {
		t.Log("Neovim not found: the review screen was not drawn")
		return
	}
	acceptAll(t, m)
	shoot("review")
}

// A dark terminal in Kanagawa's own colors, which the interface palette follows.
const shotBg, shotFg, shotFrame, shotTitle = "#1F1F28", "#DCD7BA", "#363646", "#727169"

const (
	cellW, cellH = 9, 20 // One terminal cell, in pixels.
	shotPad      = 18
	shotBar      = 30 // Window title bar.
)

// screenSVG draws a rendered view as a terminal window. Every glyph is placed on its
// cell, so wide characters and box drawing stay aligned whatever font the viewer has.
func screenSVG(view string, w, h int) []byte {
	buf := uv.NewScreenBuffer(w, h)
	buf.Method = ansi.GraphemeWidth
	uv.NewStyledString(view).Draw(buf, buf.Bounds())

	width, height := w*cellW+2*shotPad, h*cellH+2*shotPad+shotBar
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" font-family="ui-monospace,SFMono-Regular,Menlo,Consolas,'Liberation Mono',monospace" font-size="15">`+"\n", width, height, width, height)
	fmt.Fprintf(&b, `<rect x="0.5" y="0.5" width="%d" height="%d" rx="10" fill="%s" stroke="%s"/>`+"\n", width-1, height-1, shotBg, shotFrame)
	fmt.Fprintf(&b, `<path d="M1 %d H%d" stroke="%s"/>`+"\n", shotBar, width-1, shotFrame)
	for i, c := range []string{"#FF5F57", "#FEBC2E", "#28C840"} {
		fmt.Fprintf(&b, `<circle cx="%d" cy="%d" r="6" fill="%s"/>`+"\n", 20+i*20, shotBar/2, c)
	}
	fmt.Fprintf(&b, `<text x="%d" y="%d" fill="%s" font-size="13" text-anchor="middle">shoshinvim</text>`+"\n", width/2, shotBar/2+5, shotTitle)

	// Backgrounds first, merged along each row so no seams show between cells;
	// then text; then box drawing, one path per color.
	var fills, texts strings.Builder
	lines := map[string]*strings.Builder{}
	var order []string
	for y := range h {
		top := shotBar + shotPad + y*cellH
		var run []*uv.Cell
		var xs []int
		flush := func() {
			if len(run) > 0 {
				writeRun(&texts, run, xs, top)
			}
			run, xs = nil, nil
		}
		fillStart, fillColor := 0, ""
		fill := func(end int, next string) {
			if fillColor != "" && end > fillStart {
				fmt.Fprintf(&fills, `<rect x="%d" y="%d" width="%d" height="%d" fill="%s"/>`+"\n", fillStart, top, end-fillStart, cellH, fillColor)
			}
			fillStart, fillColor = end, next
		}
		for x := 0; x < w; x++ {
			c := buf.CellAt(x, y)
			if c == nil || c.Width == 0 {
				continue
			}
			left := shotPad + x*cellW
			fg, bg := cellColors(c)
			if bg != fillColor {
				fill(left, bg)
			}
			if d := boxPath(c.Content, left, top); d != "" {
				flush()
				if lines[fg] == nil {
					lines[fg] = &strings.Builder{}
					order = append(order, fg)
				}
				lines[fg].WriteString(d)
				continue
			}
			// Runs stop at spaces: viewers may collapse whitespace inside SVG text.
			if c.Content == " " || len(run) > 0 && !sameLook(run[0], c) || len([]rune(c.Content)) != 1 {
				flush()
			}
			if c.Content == " " {
				continue
			}
			run, xs = append(run, c), append(xs, left)
			if len([]rune(c.Content)) != 1 {
				flush()
			}
		}
		flush()
		fill(shotPad+w*cellW, "")
	}
	fmt.Fprintf(&b, `<g shape-rendering="crispEdges">`+"\n"+`%s</g>`+"\n", fills.String())
	b.WriteString(texts.String())
	for _, fg := range order {
		fmt.Fprintf(&b, `<path d="%s" stroke="%s" stroke-width="1.2" fill="none"/>`+"\n", lines[fg].String(), fg)
	}
	b.WriteString("</svg>\n")
	return []byte(b.String())
}

func writeRun(b *strings.Builder, run []*uv.Cell, xs []int, top int) {
	fg, _ := cellColors(run[0])
	pos := make([]string, len(xs))
	for i, x := range xs {
		pos[i] = fmt.Sprint(x)
	}
	var text strings.Builder
	for _, c := range run {
		text.WriteString(c.Content)
	}
	weight := ""
	if run[0].Style.Attrs&uv.AttrBold != 0 {
		weight = ` font-weight="bold"`
	}
	if run[0].Style.Attrs&uv.AttrItalic != 0 {
		weight += ` font-style="italic"`
	}
	fmt.Fprintf(b, `<text x="%s" y="%d" fill="%s"%s>%s</text>`+"\n", strings.Join(pos, " "), top+15, fg, weight, xmlEscape(text.String()))
}

func sameLook(a, c *uv.Cell) bool {
	return a.Style.Attrs == c.Style.Attrs && hexColor(a.Style.Fg) == hexColor(c.Style.Fg) && hexColor(a.Style.Bg) == hexColor(c.Style.Bg)
}

func cellColors(c *uv.Cell) (fg, bg string) {
	fg, bg = hexColor(c.Style.Fg), hexColor(c.Style.Bg)
	if fg == "" {
		fg = shotFg
	}
	if c.Style.Attrs&uv.AttrReverse != 0 {
		if bg == "" {
			bg = shotBg
		}
		fg, bg = bg, fg
	}
	return fg, bg
}

func hexColor(c color.Color) string {
	if c == nil {
		return ""
	}
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02X%02X%02X", r>>8, g>>8, b>>8)
}

// boxPath draws the box-drawing characters the interface uses as lines, which join
// between cells where font glyphs would leave gaps.
func boxPath(s string, left, top int) string {
	x0, y0 := float64(left), float64(top)
	x1, y1 := x0+cellW, y0+cellH
	cx, cy, r := x0+cellW/2.0, y0+cellH/2.0, cellW/2.0
	switch s {
	case "─", "━":
		return fmt.Sprintf("M%g %gH%g", x0, cy, x1)
	case "│":
		return fmt.Sprintf("M%g %gV%g", cx, y0, y1)
	case "╭":
		return fmt.Sprintf("M%g %gA%g %g 0 0 0 %g %gV%g", x1, cy, r, r, cx, cy+r, y1)
	case "╮":
		return fmt.Sprintf("M%g %gA%g %g 0 0 1 %g %gV%g", x0, cy, r, r, cx, cy+r, y1)
	case "╰":
		return fmt.Sprintf("M%g %gV%gA%g %g 0 0 0 %g %g", cx, y0, cy-r, r, r, x1, cy)
	case "╯":
		return fmt.Sprintf("M%g %gV%gA%g %g 0 0 1 %g %g", cx, y0, cy-r, r, r, x0, cy)
	}
	return ""
}

// wordmarkSVG draws the welcome screen's block wordmark, one terminal cell per pixel,
// shading each row as the interface does.
func wordmarkSVG(t theme) []byte {
	const px, py = 8, 16
	w := logoWidth()
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`+"\n", w*px, logoRows*py, w*px, logoRows*py)
	for y := range logoRows {
		var row strings.Builder
		for i, r := range logoWord {
			if i > 0 {
				row.WriteByte('.')
			}
			row.WriteString(logoGlyphs[r][y])
		}
		pixels := row.String()
		var d strings.Builder
		for x := 0; x < len(pixels); {
			n := len(pixels[x:]) - len(strings.TrimLeft(pixels[x:], pixels[x:x+1]))
			if pixels[x] == '#' {
				fmt.Fprintf(&d, "M%d %dh%dv%dh-%dz", x*px, y*py, n*px, py, n*px)
			}
			x += n
		}
		fmt.Fprintf(&b, `<path d="%s" fill="%s" shape-rendering="crispEdges"/>`+"\n", d.String(), hexColor(t.logo[y]))
	}
	b.WriteString("</svg>\n")
	return []byte(b.String())
}

func xmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(s)
}
