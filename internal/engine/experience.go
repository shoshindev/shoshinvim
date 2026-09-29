package engine

import (
	"fmt"
	"strings"
)

func appearanceLua(s State) string {
	var b strings.Builder
	b.WriteString(colorschemeLua(s))
	if Value(s, "appearance.statusline", "native") == "lualine" {
		b.WriteString(`if not vim.g.shoshinvim_probe then
  local ok, line = pcall(require, 'lualine')
  if ok then line.setup({
    options = { theme = 'auto', icons_enabled = false, globalstatus = true,
      component_separators = '|', section_separators = '', ignore_focus = { 'snacks_picker_list', 'snacks_picker_input' }, disabled_filetypes = { statusline = { 'snacks_dashboard' } } },
    sections = {
      lualine_a = { 'mode' }, lualine_b = { 'branch', 'diff' },
      lualine_c = { { 'filename', path = 1 }, { 'diagnostics', symbols = { error = 'E:', warn = 'W:', info = 'I:', hint = 'H:' } } },
      lualine_x = { function() return vim.lsp.status() end, 'filetype' },
      lualine_y = { 'progress' }, lualine_z = { 'location' },
    },
    extensions = { 'quickfix', 'oil', 'neo-tree', 'nvim-tree' },
  }) end
end
`)
	}
	if Value(s, "appearance.statusline", "native") == "mini.statusline" {
		b.WriteString(`if not vim.g.shoshinvim_probe then
  local ok, line = pcall(require, 'mini.statusline')
  if ok then line.setup({ use_icons = false }) end
end
`)
	}
	if Value(s, "appearance.buffers", "native") == "bufferline" {
		b.WriteString(`if not vim.g.shoshinvim_probe then
  local ok, line = pcall(require, 'bufferline')
  if ok then line.setup({ options = {
    diagnostics = 'nvim_lsp', separator_style = 'thin', show_buffer_icons = false,
    show_buffer_close_icons = false, show_close_icon = false,
    offsets = {
      { filetype = 'snacks_layout_box', text = 'PROJECT', text_align = 'left' },
      { filetype = 'neo-tree', text = 'PROJECT', text_align = 'left' },
      { filetype = 'NvimTree', text = 'PROJECT', text_align = 'left' },
    },
  } }) end
end
`)
	}
	if Value(s, "appearance.buffers", "native") == "barbar" {
		b.WriteString(`if not vim.g.shoshinvim_probe then
  local ok, bar = pcall(require, 'barbar')
  if ok then bar.setup({
    icons = { filetype = { enabled = false }, button = '', separator = { left = '▎', right = '' } },
    sidebar_filetypes = { NvimTree = true, ['neo-tree'] = true, snacks_layout_box = true },
  }) end
end
`)
	}
	if Value(s, "appearance.buffers", "native") == "mini.tabline" {
		b.WriteString(`if not vim.g.shoshinvim_probe then
  local ok, line = pcall(require, 'mini.tabline')
  if ok then line.setup({ show_icons = false }) end
end
`)
	}
	groups := shortcutGroups(s)
	switch Value(s, "navigation.shortcuts", "native") {
	case "which-key":
		fmt.Fprintf(&b, `if not vim.g.shoshinvim_probe then
  local ok, wk = pcall(require, 'which-key')
  if ok then
    local key_names = {}
    for _, key in ipairs({ 'Up', 'Down', 'Left', 'Right', 'C', 'M', 'D', 'S', 'CR', 'Esc', 'ScrollWheelDown', 'ScrollWheelUp', 'NL', 'BS', 'Space', 'Tab', 'F1', 'F2', 'F3', 'F4', 'F5', 'F6', 'F7', 'F8', 'F9', 'F10', 'F11', 'F12' }) do key_names[key] = key end
    wk.setup({ preset = 'modern', delay = 300, icons = { mappings = false, keys = key_names, breadcrumb = '>', separator = '->', group = '+' } })
    wk.add({ %s })
  end
end
`, strings.Join(groups.whichKey, ", "))
	case "mini.clue":
		fmt.Fprintf(&b, `if not vim.g.shoshinvim_probe then
  local ok, clue = pcall(require, 'mini.clue')
  if ok then clue.setup({
    triggers = {
      { mode = 'n', keys = '<Leader>' }, { mode = 'x', keys = '<Leader>' }, { mode = 'n', keys = 'g' }, { mode = 'x', keys = 'g' },
      { mode = 'n', keys = '[' }, { mode = 'n', keys = ']' }, { mode = 'n', keys = '<C-w>' }, { mode = 'n', keys = 'z' },
      { mode = 'n', keys = "'" }, { mode = 'n', keys = '"' }, { mode = 'i', keys = '<C-r>' },
    },
    clues = { clue.gen_clues.g(), clue.gen_clues.marks(), clue.gen_clues.registers(), clue.gen_clues.windows(), clue.gen_clues.z(), %s },
    window = { delay = 300 },
  }) end
end
`, strings.Join(groups.miniClue, ", "))
	}
	return b.String()
}

type groupLists struct{ whichKey, miniClue []string }

// shortcutGroups names the leader groups of the chosen style (for example Search).
func shortcutGroups(s State) groupLists {
	var out groupLists
	st, _ := keymapStyle(Value(withLegacyWorkspace(s), "keymaps.style", "none").(string))
	for _, g := range st.Groups {
		out.whichKey = append(out.whichKey, fmt.Sprintf("{ %s, group = %s }", LuaString(g[0]), LuaString(g[1])))
		out.miniClue = append(out.miniClue, fmt.Sprintf("{ mode = 'n', keys = %s, desc = %s }", LuaString(strings.ReplaceAll(g[0], "<leader>", "<Leader>")), LuaString("+"+g[1])))
	}
	return out
}

// snacksOptions switches on only the parts of snacks.nvim that were chosen, so it
// never competes with another plugin for the same job.
func snacksOptions(s State) string {
	on := func(id, value string, fallback any) string { return Lua(Value(s, id, fallback) == value) }
	var b strings.Builder
	fmt.Fprintf(&b, "{ picker = { enabled = %s, prompt = '> ', icons = { files = { enabled = false } } },\n", on("navigation.picker", "snacks", "native"))
	fmt.Fprintf(&b, " explorer = { enabled = %s }, indent = { enabled = %s },\n", on("navigation.explorer", "snacks", "native"), on("appearance.indent", "snacks", "none"))
	notify := on("appearance.notifications", "snacks", "none")
	fmt.Fprintf(&b, " input = { enabled = %s }, notifier = { enabled = %s, timeout = 3000 },\n", notify, notify)
	if Value(s, "workspace.start_screen", "none") == "snacks" {
		b.WriteString(` dashboard = { enabled = true, preset = {
   header = 'S H O S H I N\n\nA fresh mind. A focused workspace.',
   keys = {
`)
		for _, item := range startItems(s) {
			fmt.Fprintf(&b, "     { icon = '', key = %s, desc = %s, action = %s },\n", LuaString(item.key), LuaString(item.label), LuaString(":"+item.command))
		}
		b.WriteString(`   },
 }, sections = { { section = 'header' }, { section = 'keys', gap = 1, padding = 1 }, { text = 'Press your leader key in a file to discover shortcuts.', align = 'center', padding = 1 } } },
`)
	}
	b.WriteString(" }")
	return b.String()
}

type startItem struct{ key, label, command string }

// startItems are the start screen's entries; the guide appears only when shortcuts exist.
func startItems(s State) []startItem {
	items := []startItem{{"f", "Find a file", "ShoshinFiles"}, {"e", "Explore this project", "ShoshinExplore"}, {"r", "Recent files", "ShoshinRecent"}, {"n", "New file", "enew"}}
	if len(EffectiveKeymaps(s)) > 0 {
		items = append(items, startItem{"?", "Shortcut guide", "ShoshinHelp"})
	}
	return append(items, startItem{"q", "Quit", "qa"})
}

// interfaceLua sets up the start screen, indent guides and notifications that are not
// part of snacks.nvim.
func interfaceLua(s State) string {
	var b strings.Builder
	setup := func(module, call string) {
		fmt.Fprintf(&b, "if not vim.g.shoshinvim_probe then\n  local ok, m = pcall(require, %s)\n  if ok then %s end\nend\n", LuaString(module), call)
	}
	if Value(s, "workspace.start_screen", "none") == "mini.starter" {
		var items []string
		for _, item := range startItems(s) {
			items = append(items, fmt.Sprintf("{ name = %s, action = %s, section = 'Start' }", LuaString(item.label), LuaString(item.command)))
		}
		setup("mini.starter", "m.setup({ header = 'S H O S H I N\\n\\nA fresh mind. A focused workspace.', items = { "+strings.Join(items, ", ")+" }, footer = 'Press your leader key in a file to discover shortcuts.' })")
	}
	switch Value(s, "appearance.indent", "none") {
	case "indent-blankline":
		setup("ibl", "m.setup({ indent = { char = '│' }, scope = { enabled = false } })")
	case "mini.indentscope":
		setup("mini.indentscope", "m.setup({ symbol = '│' })")
	}
	if Value(s, "appearance.notifications", "none") == "fidget" {
		setup("fidget", "m.setup({})")
	}
	return b.String()
}
