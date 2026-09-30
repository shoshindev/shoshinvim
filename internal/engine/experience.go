package engine

import (
	"fmt"
	"strings"
)

func appearanceLua(s State) string {
	var b strings.Builder
	b.WriteString(colorschemeLua(s))
	nerd := nerdIcons(s)
	if Value(s, "appearance.statusline", "native") == "lualine" {
		// A space follows each Nerd Font icon; plain letters read as E:2 W:1.
		gap, count := "''", "':'"
		if nerd {
			gap, count = "' '", "' '"
		}
		fmt.Fprintf(&b, `if not vim.g.shoshinvim_probe then
  local ok, line = pcall(require, 'lualine')
  if ok then
    local icons = require('user_config.icons')
    line.setup({
      options = { theme = 'auto', icons_enabled = %s, globalstatus = true,
        component_separators = '|', section_separators = '', ignore_focus = { 'snacks_picker_list', 'snacks_picker_input' }, disabled_filetypes = { statusline = { 'snacks_dashboard' } } },
      sections = {
        lualine_a = { 'mode' },
        lualine_b = { 'branch', { 'diff', symbols = { added = icons.git.added .. %[2]s, modified = icons.git.modified .. %[2]s, removed = icons.git.removed .. %[2]s } } },
        lualine_c = { { 'filename', path = 1 }, { 'diagnostics', symbols = { error = icons.diagnostics.Error .. %[3]s, warn = icons.diagnostics.Warn .. %[3]s, info = icons.diagnostics.Info .. %[3]s, hint = icons.diagnostics.Hint .. %[3]s } } },
        lualine_x = { function() return vim.lsp.status() end, 'filetype' },
        lualine_y = { 'progress' }, lualine_z = { 'location' },
      },
      extensions = { 'quickfix', 'oil', 'neo-tree', 'nvim-tree' },
    })
  end
end
`, Lua(nerd), gap, count)
	}
	if Value(s, "appearance.statusline", "native") == "mini.statusline" {
		fmt.Fprintf(&b, `if not vim.g.shoshinvim_probe then
  local ok, line = pcall(require, 'mini.statusline')
  if ok then line.setup({ use_icons = %s }) end
end
`, Lua(nerd))
	}
	if Value(s, "appearance.buffers", "native") == "bufferline" {
		markers := "" // bufferline's arrows and close buttons are Nerd Font glyphs by default.
		if !nerd {
			markers = "\n    left_trunc_marker = '<', right_trunc_marker = '>', buffer_close_icon = 'x', close_icon = 'x',"
		}
		fmt.Fprintf(&b, `if not vim.g.shoshinvim_probe then
  local ok, line = pcall(require, 'bufferline')
  if ok then line.setup({ options = {
    diagnostics = 'nvim_lsp', separator_style = 'thin', show_buffer_icons = %s,
    always_show_bufferline = false,
    show_buffer_close_icons = false, show_close_icon = false,%s
    offsets = {
      { filetype = 'snacks_layout_box', text = 'PROJECT', text_align = 'left' },
      { filetype = 'neo-tree', text = 'PROJECT', text_align = 'left' },
      { filetype = 'NvimTree', text = 'PROJECT', text_align = 'left' },
    },
  } }) end
end
`, Lua(nerd), markers)
	}
	if Value(s, "appearance.buffers", "native") == "barbar" {
		fmt.Fprintf(&b, `if not vim.g.shoshinvim_probe then
  local ok, bar = pcall(require, 'barbar')
  if ok then bar.setup({
    auto_hide = 1,
    icons = { filetype = { enabled = %s }, button = '', separator = { left = '▎', right = '' } },
    sidebar_filetypes = { NvimTree = true, ['neo-tree'] = true, snacks_layout_box = true },
  }) end
end
`, Lua(nerd))
	}
	if Value(s, "appearance.buffers", "native") == "mini.tabline" {
		fmt.Fprintf(&b, `if not vim.g.shoshinvim_probe then
  local ok, line = pcall(require, 'mini.tabline')
  if ok then line.setup({ show_icons = %s }) end
end
`, Lua(nerd))
	}
	groups := shortcutGroups(s)
	switch Value(s, "navigation.shortcuts", "native") {
	case "which-key":
		// With a Nerd Font, which-key's own icons, chosen by the group or shortcut name;
		// otherwise key names and plain symbols in place of its glyphs.
		setup := "wk.setup({ preset = 'modern', delay = 300 })"
		if !nerd {
			setup = `local key_names = {}
    for _, key in ipairs({ 'Up', 'Down', 'Left', 'Right', 'C', 'M', 'D', 'S', 'CR', 'Esc', 'ScrollWheelDown', 'ScrollWheelUp', 'NL', 'BS', 'Space', 'Tab', 'F1', 'F2', 'F3', 'F4', 'F5', 'F6', 'F7', 'F8', 'F9', 'F10', 'F11', 'F12' }) do key_names[key] = key end
    wk.setup({ preset = 'modern', delay = 300, icons = { mappings = false, keys = key_names, breadcrumb = '>', separator = '->', group = '+' } })`
		}
		fmt.Fprintf(&b, `if not vim.g.shoshinvim_probe then
  local ok, wk = pcall(require, 'which-key')
  if ok then
    %s
    wk.add({ %s })
  end
end
`, setup, strings.Join(groups.whichKey, ", "))
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

// shortcutGroups names the leader groups of the chosen style (for example Search) that
// hold at least one configured shortcut.
func shortcutGroups(s State) groupLists {
	var out groupLists
	st, _ := keymapStyle(Value(withLegacyWorkspace(s), "keymaps.style", "none").(string))
	keys := EffectiveKeymaps(s)
	for _, g := range st.Groups {
		used := false
		for _, k := range keys {
			used = used || (len(k.Key) > len(g[0]) && strings.HasPrefix(k.Key, g[0]))
		}
		if !used {
			continue
		}
		icon := ""
		if nerdIcons(s) && g[1] == "Hunks" {
			icon = ", icon = { cat = 'filetype', name = 'git' }" // No which-key rule matches this name.
		}
		out.whichKey = append(out.whichKey, fmt.Sprintf("{ %s, group = %s%s }", LuaString(g[0]), LuaString(g[1]), icon))
		out.miniClue = append(out.miniClue, fmt.Sprintf("{ mode = 'n', keys = %s, desc = %s }", LuaString(strings.ReplaceAll(g[0], "<leader>", "<Leader>")), LuaString("+"+g[1])))
	}
	return out
}

// snacksOptions switches on only the parts of snacks.nvim that were chosen, so it
// never competes with another plugin for the same job.
func snacksOptions(s State) string {
	on := func(id, value string, fallback any) string { return Lua(Value(s, id, fallback) == value) }
	nerd := nerdIcons(s)
	var b strings.Builder
	if nerd {
		fmt.Fprintf(&b, "{ picker = { enabled = %s },\n", on("navigation.picker", "snacks", "native"))
	} else {
		fmt.Fprintf(&b, "{ picker = { enabled = %s, prompt = '> ', icons = %s },\n", on("navigation.picker", "snacks", "native"), snacksPlainIcons())
	}
	fmt.Fprintf(&b, " explorer = { enabled = %s }, indent = { enabled = %s },\n", on("navigation.explorer", "snacks", "native"), on("appearance.indent", "snacks", "none"))
	notify := on("appearance.notifications", "snacks", "none")
	if nerd {
		fmt.Fprintf(&b, " input = { enabled = %s }, notifier = { enabled = %s, timeout = 3000 },\n", notify, notify)
	} else {
		fmt.Fprintf(&b, " input = { enabled = %s, icon = '' },\n notifier = { enabled = %s, timeout = 3000, icons = { error = 'E', warn = 'W', info = 'I', debug = 'D', trace = 'T' } },\n", notify, notify)
	}
	if Value(s, "workspace.start_screen", "none") == "snacks" {
		b.WriteString(` dashboard = { enabled = true, preset = {
   keys = {
`)
		for _, item := range startItems(s) {
			icon := "''"
			if nerd {
				icon = "icons.start." + item.command + " .. ' '"
			}
			fmt.Fprintf(&b, "     { icon = %s, key = %s, desc = %s, action = %s },\n", icon, LuaString(item.key), LuaString(item.label), LuaString(":"+item.command))
		}
		b.WriteString(`   },
 }, sections = { function(self) return require('user_config.logo').sections(self:size().width) end, { section = 'keys', gap = 1, padding = 1 }, { text = 'Press your leader key in a file to discover shortcuts.', align = 'center', padding = 1 } } },
`)
	}
	b.WriteString(" }")
	return b.String()
}

// snacksPlainIcons replaces every Nerd Font glyph snacks.nvim's picker and explorer
// show by default. Directory icons stay for the directory preview.
func snacksPlainIcons() string {
	kinds := make([]string, len(snacksKinds))
	for i, k := range snacksKinds {
		kinds[i] = k + " = ''"
	}
	return `{
    files = { enabled = false, dir = '▸ ', dir_open = '▾ ', file = '  ' },
    keymaps = { nowait = 'nowait ' }, undo = { saved = 'saved ' },
    ui = { live = 'live ', selected = '* ', unselected = '  ' },
    git = { commit = '', added = 'A', deleted = 'D', ignored = 'I', modified = 'M', renamed = 'R', unmerged = 'U', untracked = '?' },
    diagnostics = { Error = 'E ', Warn = 'W ', Hint = 'H ', Info = 'I ' },
    lsp = { unavailable = 'x ', enabled = '+ ', disabled = '- ', attached = '* ' },
    kinds = { ` + strings.Join(kinds, ", ") + ` },
  }`
}

// snacksKinds are the symbol kinds snacks.nvim draws an icon for.
var snacksKinds = []string{"Array", "Boolean", "Class", "Color", "Control", "Collapsed", "Constant", "Constructor", "Copilot", "Enum", "EnumMember", "Event",
	"Field", "File", "Folder", "Function", "Interface", "Key", "Keyword", "Method", "Module", "Namespace", "Null", "Number", "Object", "Operator",
	"Package", "Property", "Reference", "Snippet", "String", "Struct", "Text", "TypeParameter", "Unit", "Unknown", "Value", "Variable"}

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
		setup("mini.starter", "local logo = require('user_config.logo')\n  m.setup({ header = logo.header, items = { "+strings.Join(items, ", ")+" }, footer = 'Press your leader key in a file to discover shortcuts.',\n    content_hooks = { logo.paint, m.gen_hook.adding_bullet(), m.gen_hook.aligning('center', 'center') } })")
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

// logoLua is the start screen's wordmark module: the setup's block logo in its row
// colors, with a one-line mark for windows too narrow for it.
func logoLua(s State) string {
	start := Value(s, "workspace.start_screen", "none")
	if start != "snacks" && start != "mini.starter" {
		return ""
	}
	quote := func(list []string) string {
		parts := make([]string, len(list))
		for i, v := range list {
			parts[i] = LuaString(v)
		}
		return strings.Join(parts, ", ")
	}
	var b strings.Builder
	b.WriteString("-- The ShoshinVim wordmark on the start screen. In pixels, # is a filled cell.\nlocal M = {\n  pixels = {\n")
	for _, row := range LogoPixels() {
		fmt.Fprintf(&b, "    %s,\n", LuaString(row))
	}
	fmt.Fprintf(&b, `  },
  colors = {
    dark = { %s },
    light = { %s },
  },
  mark = %s, -- for windows narrower than min_width
  tagline = %s,
  min_width = %d,
}

-- Filled cells are painted as background color, so the rows join without seams.
function M.highlight()
  for i, color in ipairs(M.colors[vim.o.background == 'light' and 'light' or 'dark']) do
    vim.api.nvim_set_hl(0, 'ShoshinLogo' .. i, { bg = color })
  end
end

-- runs splits a row into runs of filled and empty cells: { { width, filled }, ... }.
function M.runs(row)
  local runs, i = {}, 1
  while i <= #row do
    local j = i
    while row:sub(j, j) == row:sub(i, i) do j = j + 1 end
    runs[#runs + 1] = { j - i, row:sub(i, i) == '#' }
    i = j
  end
  return runs
end
`, quote(LogoColors(true)), quote(LogoColors(false)), LuaString(LogoMark), LuaString("A fresh mind. A focused workspace."), len(LogoPixels()[0])+4)
	if start == "snacks" {
		b.WriteString(`
-- sections is the snacks.nvim dashboard header for a window this wide.
function M.sections(width)
  local tagline = { text = { M.tagline, hl = 'SnacksDashboardHeader' }, align = 'center', padding = 2 }
  if width < M.min_width then
    return { { text = { M.mark, hl = 'SnacksDashboardHeader' }, align = 'center', padding = 1 }, tagline }
  end
  local sections = {}
  for y, row in ipairs(M.pixels) do
    local text = {}
    for _, run in ipairs(M.runs(row)) do
      text[#text + 1] = { (' '):rep(run[1]), hl = run[2] and ('ShoshinLogo' .. y) or nil }
    end
    sections[#sections + 1] = { text = text, align = 'center', padding = y == #M.pixels and 1 or 0 }
  end
  sections[#sections + 1] = tagline
  return sections
end
`)
	} else {
		b.WriteString(`
-- header is mini.starter's header: blank wordmark rows, which paint() fills in.
function M.header()
  M.wide = vim.api.nvim_win_get_width(0) >= M.min_width
  if not M.wide then return M.mark .. '\n\n' .. M.tagline end
  local lines = {}
  for _, row in ipairs(M.pixels) do lines[#lines + 1] = (' '):rep(#row) end
  return table.concat(lines, '\n') .. '\n\n' .. M.tagline
end

-- paint is a mini.starter content hook that colors the wordmark rows.
function M.paint(content)
  local y = 0
  for _, line in ipairs(content) do
    if M.wide and y < #M.pixels and #line == 1 and line[1].type == 'header' then
      y = y + 1
      local units = {}
      for _, run in ipairs(M.runs(M.pixels[y])) do
        units[#units + 1] = { string = (' '):rep(run[1]), type = 'header', hl = run[2] and ('ShoshinLogo' .. y) or 'MiniStarterHeader' }
      end
      for k in pairs(line) do line[k] = nil end
      for k, unit in ipairs(units) do line[k] = unit end
    end
  end
  return content
end
`)
	}
	b.WriteString(`
if not vim.g.shoshinvim_probe then
  M.highlight()
  local group = vim.api.nvim_create_augroup('ShoshinLogo', { clear = true })
  vim.api.nvim_create_autocmd('ColorScheme', { group = group, callback = M.highlight })
  vim.api.nvim_create_autocmd('OptionSet', { group = group, pattern = 'background', callback = M.highlight })
end
return M
`)
	return b.String()
}
