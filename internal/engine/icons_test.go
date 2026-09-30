package engine

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

// decodeLua reads back a double-quoted Lua string written by LuaString or luaText.
func decodeLua(t *testing.T, lit string) string {
	t.Helper()
	if len(lit) < 2 || lit[0] != '"' || lit[len(lit)-1] != '"' {
		t.Fatalf("unquoted %q", lit)
	}
	var b strings.Builder
	for i := 1; i < len(lit)-1; i++ {
		c := lit[i]
		switch {
		case c == '"':
			t.Fatalf("unescaped quote in %q", lit)
		case c != '\\':
			b.WriteByte(c)
		case lit[i+1] == '"' || lit[i+1] == '\\':
			b.WriteByte(lit[i+1])
			i++
		default:
			n, err := strconv.Atoi(lit[i+1 : i+4])
			if err != nil {
				t.Fatalf("bad escape in %q", lit)
			}
			b.WriteByte(byte(n))
			i += 3
		}
	}
	return b.String()
}

func FuzzLuaText(f *testing.F) {
	for _, s := range []string{"", "hello", "\"\\\n", "Հայերեն", "\U000f0219 ", "\xff\xfe", " "} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		got := luaText(s)
		if strings.ContainsAny(got, "\n\r\x00") || !utf8.ValidString(got) {
			t.Fatalf("raw controls or invalid UTF-8 in %q", got)
		}
		if back := decodeLua(t, got); back != s {
			t.Fatalf("%q reads back as %q", s, back)
		}
	})
}

func TestLuaTextKeepsGlyphsReadable(t *testing.T) {
	if got := luaText("\U000f0219 x"); got != "\"\U000f0219 x\"" {
		t.Fatalf("glyph escaped: %s", got)
	}
	if got := luaText("a\tb\u0085"); got != `"a\009b\194\133"` {
		t.Fatalf("control characters must stay escaped: %s", got)
	}
}

var luaEscape = regexp.MustCompile(`\\(\d{3})`)

// nerdGlyphs lists the private-use characters (where Nerd Fonts put their icons) in
// generated files, reading LuaString's byte escapes back.
func nerdGlyphs(files map[string]string) map[string][]rune {
	out := map[string][]rune{}
	for path, text := range files {
		text = luaEscape.ReplaceAllStringFunc(text, func(m string) string {
			n, _ := strconv.Atoi(m[1:])
			return string([]byte{byte(n)})
		})
		for _, r := range text {
			if r >= 0xe000 && r <= 0xf8ff || r >= 0xf0000 {
				out[path] = append(out[path], r)
			}
		}
	}
	return out
}

// Every plugin choice, with plain symbols, must stay readable without a Nerd Font.
func TestPlainSymbolsNeedNoNerdFont(t *testing.T) {
	base, _ := LoadCatalog().Profile(NewState(), "rich-ide")
	base.Values["language.typescript.lsp"] = true
	choices := map[string][]string{}
	for _, set := range LoadCatalog().Settings {
		if set.Type == "enum" && len(set.Choices) > 1 && !strings.HasPrefix(set.ID, "appearance.custom_theme") && set.ID != "appearance.colorscheme" && set.ID != "appearance.icons" {
			choices[set.ID] = set.Choices
		}
	}
	for _, icons := range []string{"plain", "nerd"} {
		found := map[string]bool{}
		for id, values := range choices {
			for _, v := range values {
				s := Clone(base)
				s.Values["appearance.icons"] = icons
				s.Values[id] = v
				files, err := Generate(s)
				if err != nil {
					continue // Some combinations are invalid, such as a plugin without a manager.
				}
				for path := range nerdGlyphs(files) {
					found[path] = true
					if icons == "plain" {
						t.Errorf("%s=%s: Nerd Font glyphs in %s with plain symbols", id, v, path)
					}
				}
			}
		}
		if icons == "nerd" && !found["lua/user_config/icons.lua"] {
			t.Error("Nerd Font icons are missing from icons.lua")
		}
	}
}

func TestNerdIconsUseMiniIcons(t *testing.T) {
	s, _ := LoadCatalog().Profile(NewState(), "rich-ide")
	if slices.Contains(Plugins(s), "mini.icons") {
		t.Fatal("plain symbols must not install mini.icons")
	}
	s.Values["appearance.icons"] = "nerd"
	if !slices.Contains(Plugins(s), "mini.icons") {
		t.Fatal("Nerd Font icons need mini.icons")
	}
	files, err := Generate(s)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(files["lua/user_config/plugins.lua"], "package.preload['nvim-web-devicons']") {
		t.Fatal("plugins that ask for nvim-web-devicons must get mini.icons")
	}
}
