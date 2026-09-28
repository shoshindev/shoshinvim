package engine

import (
	"fmt"
	"strings"
)

type comparison struct{ choice, strength, cost, source string }

var comparisons = map[string][]comparison{
	"navigation.picker": {
		{"native", "Filename completion using Neovim's input prompt", "No downloads; fewer search sources", "https://neovim.io/doc/user/editing.html"},
		{"snacks", "Unified file/text/Git/diagnostic pickers and a sidebar", "Larger feature set; ripgrep needed for text search", "https://github.com/folke/snacks.nvim"},
		{"mini.pick", "Compact, focused fuzzy file and live text selection", "Smaller scope; no project sidebar", "https://github.com/nvim-mini/mini.pick"},
	},
	"navigation.explorer": {
		{"native", "Built-in directory navigation with netrw", "No installation; traditional Vim interface", "https://neovim.io/doc/user/pi_netrw.html"},
		{"oil", "Edit directories as buffers, then save changes", "Extra plugin; file operations use an editing workflow", "https://github.com/stevearc/oil.nvim"},
		{"snacks", "Persistent project sidebar and searchable file operations", "Extra plugin; integrates with Snacks picker", "https://github.com/folke/snacks.nvim"},
	},
	"formatting.provider": {
		{"native", "Use an attached language server's formatting", "Depends on server support; on-save runs only with one formatting client", "https://neovim.io/doc/user/lsp.html"},
		{"conform", "Prettier for Web, StyLua for Lua, optional format on save", "Requires plugin and selected external formatters", "https://github.com/stevearc/conform.nvim"},
	},
	"plugins.manager": {
		{"none", "Independent native Neovim configuration", "Plugin-backed choices must be disabled", "https://neovim.io/doc/user/starting.html"},
		{"lazy", "Load reviewed pinned plugins already present on disk", "Git required for approved acquisition; automatic installation disabled", "https://lazy.folke.io/configuration"},
	},
}

func (c Catalog) Comparison(id string) (string, error) {
	s, err := c.Find(id)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n%s\n\n", s.Label, s.Description)
	if rows, ok := comparisons[id]; ok {
		for _, r := range rows {
			fmt.Fprintf(&b, "%s\n  Good for: %s\n  Tradeoff: %s\n  Source: %s\n\n", r.choice, r.strength, r.cost, r.source)
		}
	} else {
		fmt.Fprintf(&b, "Choices: %s\n", strings.Join(s.Choices, ", "))
	}
	fmt.Fprintf(&b, "Catalog: %s. macOS integration evidence; cross-platform release gates remain open.\n/set %s <choice> changes the draft only.", CatalogVersion, id)
	return b.String(), nil
}
