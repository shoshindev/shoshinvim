package engine

import (
	"fmt"
	"strings"
)

// borderStyle is the chosen popup border, for plugins that draw their own frames
// instead of following 'winborder'. It is empty when no border was chosen or when
// borders are off; those plugins then keep their own frame.
func borderStyle(s State) string {
	switch b, _ := s.Values["options.winborder"].(string); b {
	case "rounded", "single", "double", "solid", "bold", "shadow":
		return b
	}
	return ""
}

// refinementsLua sets the options distributions such as LazyVim tune for a calmer,
// steadier screen. Each one is left out when the state sets that option, so a value
// chosen in ShoshinVim always wins; user.lua runs later and wins too.
func refinementsLua(s State) string {
	var lines []string
	add := func(option, lua, why string) {
		if _, set := s.Values["options."+option]; !set {
			lines = append(lines, fmt.Sprintf("%-44s -- %s", lua, why))
		}
	}
	add("smoothscroll", "vim.o.smoothscroll = true", "scroll long wrapped lines line by line")
	add("splitkeep", "vim.o.splitkeep = 'screen'", "text stays still when a split or sidebar opens")
	add("pumheight", "vim.o.pumheight = 10", "at most 10 items in a completion menu")
	if b := borderStyle(s); b != "" {
		add("pumborder", "vim.o.pumborder = "+LuaString(b), "completion menus framed like other popups")
	}
	if Value(s, "appearance.statusline", "native") != "native" {
		add("ruler", "vim.o.ruler = false", "the status line already shows the position")
	}
	flags := "WcC"
	if Value(s, "workspace.start_screen", "none") != "none" {
		flags += "I" // The start screen replaces Neovim's intro message.
	}
	add("shortmess", "vim.opt.shortmess:append("+LuaString(flags)+")", "fewer routine messages")
	// Only keys the chosen value lacks are added, so the Editor's empty-lines choice stays.
	fill, _ := s.Values["options.fillchars"].(string)
	var chars []string
	for _, c := range [][2]string{{"fold", "' '"}, {"foldsep", "' '"}, {"diff", "'╱'"}} {
		if !strings.Contains(","+fill, ","+c[0]+":") {
			chars = append(chars, c[0]+" = "+c[1])
		}
	}
	if len(chars) > 0 {
		lines = append(lines, fmt.Sprintf("%-44s -- %s", "vim.opt.fillchars:append({ "+strings.Join(chars, ", ")+" })", "quiet folds and diffs"))
	}
	if Value(s, "syntax.engine", "native") == "nvim-treesitter" {
		// Folds follow the code's structure; files without a parser get none. Everything
		// starts unfolded, and a closed fold shows its first line with its colors.
		add("foldmethod", "vim.o.foldmethod = 'expr'", "fold by code structure (za toggles a fold)")
		add("foldexpr", "vim.o.foldexpr = 'v:lua.vim.treesitter.foldexpr()'", "Tree-sitter folds")
		add("foldlevel", "vim.o.foldlevel = 99", "files open unfolded")
		add("foldlevelstart", "vim.o.foldlevelstart = 99", "files open unfolded")
		add("foldtext", "vim.o.foldtext = ''", "a closed fold keeps its line's colors")
	}
	if len(lines) == 0 {
		return ""
	}
	return "-- Refinements for the choices above. An option set in ShoshinVim or in user.lua takes precedence.\n" + strings.Join(lines, "\n") + "\n"
}
