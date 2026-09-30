package engine

import (
	"path"
	"sort"
	"strings"
)

// repositoryNames are the upstream names of the plugins a state installs, such as
// gitsigns.nvim. Themes decide which plugins to style by these names, and ShoshinVim's
// lazy.nvim specs do not carry them: each spec is named after its folder.
func repositoryNames(s State) []string {
	var out []string
	for _, id := range Plugins(s) {
		if p, ok := PluginInfo(id); ok {
			out = append(out, path.Base(p.Repository))
		}
	}
	sort.Strings(out)
	return out
}

// materialPlugins names material.nvim's highlight modules for the plugins that have
// one; material.nvim styles only the plugins listed in its setup.
var materialPlugins = map[string]string{
	"blink.cmp": "blink", "nvim-dap": "dap", "fidget": "fidget", "gitsigns": "gitsigns", "indent-blankline": "indent-blankline",
	"neo-tree": "neo-tree", "noice": "noice", "nvim-cmp": "nvim-cmp", "nvim-tree": "nvim-tree", "telescope": "telescope", "which-key": "which-key",
}

// sainnheThemes are the themes by sainnhe, which share their settings' names.
var sainnheThemes = map[string]string{"everforest": "everforest", "gruvbox-material": "gruvbox_material", "sonokai": "sonokai", "edge": "edge"}

// themeSetup is Lua run before :colorscheme so the theme styles every plugin chosen
// with it and fits the rest of the setup. Themes not listed style what is installed
// by themselves, and keep their variant's own setting.
func themeSetup(s State, t Theme, v ThemeVariant) string {
	names := repositoryNames(s)
	switch t.ID {
	case "tokyonight", "solarized-osaka":
		// Both find plugins by lazy.nvim name, so they are named here.
		sidebars := ""
		if t.ID == "tokyonight" {
			sidebars = "styles = { sidebars = 'dark', floats = 'dark' }, "
		}
		set := make([]string, len(names))
		for i, n := range names {
			set[i] = "[" + LuaString(n) + "] = true"
		}
		return "require(" + LuaString(t.ID) + ").setup({ " + sidebars + "plugins = { " + strings.Join(set, ", ") + " } })"
	case "catppuccin":
		// catppuccin's own table maps plugin names to its integrations (noice is missing
		// from it); integrations with settings keep their defaults.
		return `local map = require('catppuccin.utils.integration_mappings')
    local defaults, integrations = require('catppuccin').default_options.integrations, {}
    for _, name in ipairs({ ` + luaList(names) + ` }) do
      local key = name == 'noice.nvim' and 'noice' or map[name:find('^mini%.') and 'mini.nvim' or name]
      if key then integrations[key] = type(defaults[key]) == 'table' and vim.tbl_extend('force', defaults[key], { enabled = true }) or true end
    end
    require('catppuccin').setup({ integrations = integrations })`
	case "material":
		var list []string
		mini := false
		for _, id := range Plugins(s) {
			if m, ok := materialPlugins[id]; ok {
				list = append(list, m)
			}
			mini = mini || strings.HasPrefix(id, "mini.")
		}
		if mini {
			list = append(list, "mini")
		}
		sort.Strings(list)
		return "require('material').setup({ plugins = { " + luaList(list) + " } })"
	case "kanagawa":
		// The README's gutter tweak: signs and line numbers sit on the text's background.
		return "require('kanagawa').setup({ colors = { theme = { all = { ui = { bg_gutter = 'none' } } } } })"
	}
	if prefix, ok := sainnheThemes[t.ID]; ok {
		// Inline diagnostics in their severity's color rather than grey, and popups a
		// shade darker than the text, as in LazyVim and NvChad ('blend' would need a
		// border on every popup).
		settings := "vim.g." + prefix + "_diagnostic_virtual_text = 'colored'\n    vim.g." + prefix + "_float_style = 'dim'"
		if v.Before == "" {
			return settings
		}
		return v.Before + "\n    " + settings
	}
	return v.Before
}

// bufferlineHighlights is Lua for bufferline's highlights where the theme provides its
// own, read once the theme has loaded; nil otherwise.
func bufferlineHighlights(s State) string {
	value, _ := s.Values["appearance.colorscheme"].(string)
	t, _, ok := ThemeFor(value)
	if !ok {
		return ""
	}
	switch t.ID {
	case "catppuccin":
		return "(function() local ok, theme = pcall(require, 'catppuccin.special.bufferline'); return ok and theme.get_theme() or nil end)()"
	case "rose-pine":
		return "(function() local ok, theme = pcall(require, 'rose-pine.plugins.bufferline'); return ok and theme or nil end)()"
	}
	return ""
}
