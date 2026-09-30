package engine

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// nerdIcons reports whether the terminal font has Nerd Font glyphs. A state saved
// before the choice existed keeps plain symbols, so regenerating it changes nothing.
func nerdIcons(s State) bool { return Value(s, "appearance.icons", "plain") == "nerd" }

// An icon is one named glyph; note names a Nerd Font glyph, so the generated file
// says which one it is even where the font is missing.
type icon struct{ key, glyph, note string }

func nf(key string, r rune, note string) icon { return icon{key, string(r), note} }

type iconGroup struct {
	name  string
	icons []icon
}

// iconGroups are the icons ShoshinVim itself assigns: signs, status line symbols,
// completion kinds and start screen entries. Plugins keep their own icons otherwise.
func iconGroups(s State) []iconGroup {
	if !nerdIcons(s) {
		return []iconGroup{
			{"diagnostics", []icon{{"Error", "E", ""}, {"Warn", "W", ""}, {"Info", "I", ""}, {"Hint", "H", ""}}},
			{"git", []icon{{"added", "+", ""}, {"modified", "~", ""}, {"removed", "-", ""},
				{"sign", "▎", ""}, {"delete", "▁", ""}, {"topdelete", "▔", ""}}},
			{"ui", []icon{{"clock", "", ""}, {"modified", "[+]", ""}, {"readonly", "[-]", ""},
				{"section_left", "", ""}, {"section_right", "", ""}, {"component_left", "|", ""}, {"component_right", "|", ""},
				{"search", ">", ""}, {"caret", ">", ""}}},
		}
	}
	return []iconGroup{
		// LazyVim's icons (lua/lazyvim/config/init.lua).
		{"diagnostics", []icon{nf("Error", 0xf057, "fa-times_circle"), nf("Warn", 0xf071, "fa-warning"), nf("Info", 0xf05a, "fa-info_circle"), nf("Hint", 0xf0eb, "fa-lightbulb_o")}},
		{"git", []icon{nf("added", 0xf0fe, "fa-plus_square"), nf("modified", 0xf14b, "fa-pencil_square"), nf("removed", 0xf146, "fa-minus_square"),
			{"sign", "▎", ""}, nf("delete", 0xf0da, "fa-caret_right"), nf("topdelete", 0xf0da, "fa-caret_right")}},
		{"ui", []icon{nf("clock", 0xf017, "fa-clock_o"), {"modified", "●", ""}, nf("readonly", 0xf023, "fa-lock"),
			nf("section_left", 0xe0b0, "pl-left_hard_divider"), nf("section_right", 0xe0b2, "pl-right_hard_divider"),
			nf("component_left", 0xe0b1, "pl-left_soft_divider"), nf("component_right", 0xe0b3, "pl-right_soft_divider"),
			nf("search", 0xf002, "fa-search"), nf("caret", 0xf054, "fa-chevron_right")}},
		{"dap", []icon{nf("Stopped", 0xf0055, "md-arrow_right_thick"), nf("Breakpoint", 0xf111, "fa-circle"), nf("BreakpointCondition", 0xf059, "fa-question_circle"),
			nf("BreakpointRejected", 0xf06a, "fa-exclamation_circle"), {"LogPoint", ".>", ""}}},
		// blink.cmp's kind icons, so every completion menu shows the same ones.
		{"kinds", []icon{nf("Text", 0xf027f, "md-format_size"), nf("Method", 0xf0295, "md-function"), nf("Function", 0xf0295, "md-function"),
			nf("Constructor", 0xf0493, "md-cog"), nf("Field", 0xf0722, "md-tag_plus"), nf("Variable", 0xf01a6, "md-cube"),
			nf("Property", 0xf05b7, "md-wrench"), nf("Class", 0xf1860, "md-format_list_group"), nf("Interface", 0xf1860, "md-format_list_group"),
			nf("Struct", 0xf1860, "md-format_list_group"), nf("Module", 0xf0169, "md-code_braces"), nf("Unit", 0xf0a9a, "md-calculator_variant"),
			nf("Value", 0xf09a8, "md-text"), nf("Enum", 0xf09a8, "md-text"), nf("EnumMember", 0xf09a8, "md-text"),
			nf("Keyword", 0xf0efe, "md-identifier"), nf("Constant", 0xf03ff, "md-pi"), nf("Snippet", 0xf113d, "md-text_recognition"),
			nf("Color", 0xf03d8, "md-palette"), nf("File", 0xf0214, "md-file"), nf("Reference", 0xf0b32, "md-file_replace"),
			nf("Folder", 0xf024b, "md-folder"), nf("Event", 0xf140b, "md-lightning_bolt"), nf("Operator", 0xf0a9a, "md-calculator_variant"),
			nf("TypeParameter", 0xf0b1b, "md-alpha_t_box")}},
		{"start", []icon{nf("ShoshinFiles", 0xf002, "fa-search"), nf("ShoshinExplore", 0xf07c, "fa-folder_open"), nf("ShoshinRecent", 0xf1da, "fa-history"),
			nf("enew", 0xf15b, "fa-file"), nf("ShoshinHelp", 0xf030c, "md-keyboard"), nf("qa", 0xf426, "oct-sign_out")}},
	}
}

// iconsLua is the icons module other modules read as require('user_config.icons').
// Glyphs are written as they are, so the file reads well in a Nerd Font.
func iconsLua(s State) string {
	if Value(s, "plugins.manager", "none") != "lazy" && !diagnosticsChosen(s) {
		return "" // Nothing uses icons.
	}
	var b strings.Builder
	if nerdIcons(s) {
		b.WriteString("-- Icons ShoshinVim uses in signs, the status line and menus: Nerd Font glyphs.\n-- Choose plain symbols in ShoshinVim if they show as boxes.\n")
	} else {
		b.WriteString("-- Symbols ShoshinVim uses in signs and the status line: plain text, as no Nerd Font was chosen.\n")
	}
	b.WriteString("return {\n")
	for _, g := range iconGroups(s) {
		fmt.Fprintf(&b, "  %s = {\n", g.name)
		for _, ic := range g.icons {
			key := ic.key
			if !luaName(key) {
				key = "[" + LuaString(key) + "]"
			}
			fmt.Fprintf(&b, "    %s = %s,", key, luaText(ic.glyph))
			if ic.note != "" {
				fmt.Fprintf(&b, " -- nf-%s", ic.note)
			}
			b.WriteByte('\n')
		}
		b.WriteString("  },\n")
	}
	b.WriteString("}\n")
	return b.String()
}

func luaName(s string) bool {
	for i, c := range s {
		if !(c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || i > 0 && c >= '0' && c <= '9') {
			return false
		}
	}
	return s != ""
}

// luaText is LuaString, except that printable non-ASCII characters (including the
// private-use code points of Nerd Font glyphs) are written as they are, not as byte
// escapes. Control characters and invalid UTF-8 are still escaped.
func luaText(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == '"' || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r >= 32 && r < 127 || r >= 0xa0 && r != utf8.RuneError && r != 0x2028 && r != 0x2029:
			b.WriteString(s[i : i+size])
		default:
			for _, c := range []byte(s[i : i+size]) {
				fmt.Fprintf(&b, "\\%03d", c)
			}
		}
		i += size
	}
	b.WriteByte('"')
	return b.String()
}
