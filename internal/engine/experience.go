package engine

import (
	"fmt"
	"strings"
)

func appearanceLua(s State) string {
	var b strings.Builder
	b.WriteString(colorschemeLua(s))
	nerd := nerdIcons(s)
	b.WriteString(noiceLua(s))
	if Value(s, "appearance.statusline", "native") == "lualine" {
		b.WriteString(lualineLua(s))
	}
	if Value(s, "appearance.statusline", "native") == "mini.statusline" {
		b.WriteString(onLoad("mini.statusline", restoreLaststatus+"\nrequire('mini.statusline').setup({ use_icons = "+Lua(nerd)+" })"))
	}
	if Value(s, "appearance.buffers", "native") == "bufferline" {
		markers := "" // bufferline's arrows and close buttons are Nerd Font glyphs by default.
		if !nerd {
			markers = "\n  left_trunc_marker = '<', right_trunc_marker = '>', buffer_close_icon = 'x', close_icon = 'x',"
		}
		gap := "''"
		if nerd {
			gap = "' '"
		}
		themeHighlights := "" // Some themes color bufferline themselves.
		if h := bufferlineHighlights(s); h != "" {
			themeHighlights = " highlights = " + h + ","
		}
		// LazyVim's settings: error and warning counts on each tab, and closing a tab
		// keeps the window layout.
		b.WriteString(onLoad("bufferline", fmt.Sprintf(`local icons = require('user_config.icons').diagnostics
local function close(n)
  local has_snacks, snacks = pcall(require, 'snacks')
  if has_snacks then snacks.bufdelete(n) else vim.cmd.bdelete({ tostring(n), bang = true }) end
end
require('bufferline').setup({%[4]s options = {
  close_command = close, right_mouse_command = close,
  diagnostics = 'nvim_lsp',
  diagnostics_indicator = function(_, _, diag)
    return vim.trim((diag.error and icons.Error .. %[3]s .. diag.error .. ' ' or '') .. (diag.warning and icons.Warn .. %[3]s .. diag.warning or ''))
  end,
  separator_style = 'thin', show_buffer_icons = %[1]s,
  always_show_bufferline = false,
  show_buffer_close_icons = false, show_close_icon = false,%[2]s
  offsets = {
    { filetype = 'snacks_layout_box', text = 'PROJECT', text_align = 'left', highlight = 'Directory', separator = true },
    { filetype = 'neo-tree', text = 'PROJECT', text_align = 'left', highlight = 'Directory', separator = true },
    { filetype = 'NvimTree', text = 'PROJECT', text_align = 'left', highlight = 'Directory', separator = true },
  },
} })`, Lua(nerd), markers, gap, themeHighlights)))
	}
	if Value(s, "appearance.buffers", "native") == "barbar" {
		b.WriteString(onLoad("barbar", fmt.Sprintf(`require('barbar').setup({
  auto_hide = 1,
  icons = { filetype = { enabled = %s }, button = '', separator = { left = '▎', right = '' } },
  sidebar_filetypes = { NvimTree = true, ['neo-tree'] = true, snacks_layout_box = true },
})`, Lua(nerd))))
	}
	if Value(s, "appearance.buffers", "native") == "mini.tabline" {
		b.WriteString(onLoad("mini.tabline", "require('mini.tabline').setup({ show_icons = "+Lua(nerd)+" })"))
	}
	groups := shortcutGroups(s)
	switch Value(s, "navigation.shortcuts", "native") {
	case "which-key":
		// With a Nerd Font, which-key's own icons, chosen by the group or shortcut name;
		// otherwise key names and plain symbols in place of its glyphs.
		win := "" // which-key frames its popup itself, rounded unless told otherwise.
		if border := borderStyle(s); border != "" {
			win = ", win = { border = " + LuaString(border) + " }"
		}
		setup := "wk.setup({ preset = 'modern', delay = 300" + win + " })"
		if !nerd {
			setup = `local key_names = {}
for _, key in ipairs({ 'Up', 'Down', 'Left', 'Right', 'C', 'M', 'D', 'S', 'CR', 'Esc', 'ScrollWheelDown', 'ScrollWheelUp', 'NL', 'BS', 'Space', 'Tab', 'F1', 'F2', 'F3', 'F4', 'F5', 'F6', 'F7', 'F8', 'F9', 'F10', 'F11', 'F12' }) do key_names[key] = key end
wk.setup({ preset = 'modern', delay = 300` + win + `, icons = { mappings = false, keys = key_names, breadcrumb = '>', separator = '->', group = '+' } })`
		}
		b.WriteString(onLoad("which-key", "local wk = require('which-key')\n"+setup+"\nwk.add({ "+strings.Join(groups.whichKey, ", ")+" })"))
	case "mini.clue":
		b.WriteString(onLoad("mini.clue", fmt.Sprintf(`local clue = require('mini.clue')
clue.setup({
  triggers = {
    { mode = 'n', keys = '<Leader>' }, { mode = 'x', keys = '<Leader>' }, { mode = 'n', keys = 'g' }, { mode = 'x', keys = 'g' },
    { mode = 'n', keys = '[' }, { mode = 'n', keys = ']' }, { mode = 'n', keys = '<C-w>' }, { mode = 'n', keys = 'z' },
    { mode = 'n', keys = "'" }, { mode = 'n', keys = '"' }, { mode = 'i', keys = '<C-r>' },
  },
  clues = { clue.gen_clues.g(), clue.gen_clues.marks(), clue.gen_clues.registers(), clue.gen_clues.windows(), clue.gen_clues.z(), %s },
  window = { delay = 300 },
})`, strings.Join(groups.miniClue, ", "))))
	}
	return b.String()
}

// noiceLua sets noice.nvim up as LazyVim does: commands in a popup near the top,
// searches at the bottom, long messages in a split, and written-lines and undo
// messages in a small corner view. Each job stays with the plugin chosen for it:
// fidget keeps language server progress and blink.cmp keeps signature help.
func noiceLua(s State) string {
	if Value(s, "appearance.cmdline", "native") != "noice" {
		return ""
	}
	fidget := Value(s, "appearance.notifications", "none") == "fidget"
	blink := Value(s, "completion.engine", "native") == "blink.cmp"
	nvimCmp := Value(s, "completion.engine", "native") == "nvim-cmp"
	var extra []string
	if b := borderStyle(s); b != "" {
		extra = append(extra, "views = { hover = { border = { style = "+LuaString(b)+" }, position = { row = 2, col = 2 } } }, -- Framed documentation popups.")
	}
	if !nerdIcons(s) {
		extra = append(extra, `cmdline = { format = { cmdline = { icon = '>' }, search_down = { icon = '/' }, search_up = { icon = '?' }, filter = { icon = '$' },
    lua = { icon = 'lua' }, help = { icon = '?' }, calculator = { icon = '=' }, input = { icon = '>' } } },
  format = { level = { icons = { error = 'E ', warn = 'W ', info = 'I ' } } },
  popupmenu = { kind_icons = false },`)
	}
	tail := ""
	if len(extra) > 0 {
		tail = "\n  " + strings.Join(extra, "\n  ")
	}
	return onLoad("noice", fmt.Sprintf(`require('noice').setup({
  lsp = {
    override = {
      ['vim.lsp.util.convert_input_to_markdown_lines'] = true,
      ['vim.lsp.util.stylize_markdown'] = true,
      ['cmp.entry.get_documentation'] = %s,
    },
    progress = { enabled = %s },
    signature = { enabled = %s },
  },
  routes = {
    { filter = { event = 'msg_show', any = { { find = '%%d+L, %%d+B' }, { find = '; after #%%d+' }, { find = '; before #%%d+' } } }, view = 'mini' },
  },
  presets = { bottom_search = true, command_palette = true, long_message_to_split = true },%s
})`, Lua(nvimCmp), Lua(!fidget), Lua(!blink), tail))
}

// restoreLaststatus brings back the status line its plugin's spec hid until it loaded.
const restoreLaststatus = "vim.o.laststatus = vim.g.shoshinvim_laststatus or vim.o.laststatus"

// lualineLua lays the status line out as LazyVim does: mode · branch · diagnostics and
// file · language server progress and Git changes · position · time.
func lualineLua(s State) string {
	nerd := nerdIcons(s)
	// A space follows each Nerd Font icon; plain letters read as E:2 W:1.
	gap, count := "''", "':'"
	fileType, language, clock := "", ", 'filetype'", "function() return os.date('%R') end"
	if nerd {
		gap, count = "' '", "' '"
		// The file type shows as its icon before the name instead of as a word.
		fileType = "\n      { 'filetype', icon_only = true, separator = '', padding = { left = 1, right = 0 } },"
		language, clock = "", "function() return icons.ui.clock .. ' ' .. os.date('%R') end"
	}
	c := "{ 'diagnostics', symbols = { error = icons.diagnostics.Error .. %[3]s, warn = icons.diagnostics.Warn .. %[3]s, info = icons.diagnostics.Info .. %[3]s, hint = icons.diagnostics.Hint .. %[3]s } }," +
		fileType + "\n      { 'filename', path = 1, symbols = { modified = icons.ui.modified, readonly = icons.ui.readonly } }"
	// noice shows language server progress itself; the bar shows its mode instead, such
	// as "recording @q".
	progress := "function() return vim.lsp.status() end,"
	if Value(s, "appearance.cmdline", "native") == "noice" {
		progress = "{ function() return require('noice').api.status.mode.get() end, cond = function() return package.loaded['noice'] and require('noice').api.status.mode.has() end, color = 'Constant' },"
	}
	// Git change counts come live from the chosen Git plugin.
	source := ""
	switch Value(s, "git.hunks", "native") {
	case "gitsigns":
		source = ",\n        source = function() local g = vim.b.gitsigns_status_dict; if g then return { added = g.added, modified = g.changed, removed = g.removed } end end"
	case "mini.diff":
		source = ",\n        source = function() local d = vim.b.minidiff_summary; if d then return { added = d.add, modified = d.change, removed = d.delete } end end"
	}
	return onLoad("lualine", fmt.Sprintf(restoreLaststatus+`
local icons = require('user_config.icons')
require('lualine').setup({
  options = { theme = 'auto', icons_enabled = %[1]s, globalstatus = true,
    component_separators = { left = icons.ui.component_left, right = icons.ui.component_right },
    section_separators = { left = icons.ui.section_left, right = icons.ui.section_right },
    ignore_focus = { 'snacks_picker_list', 'snacks_picker_input' }, disabled_filetypes = { statusline = { 'snacks_dashboard', 'ministarter' } } },
  sections = {
    lualine_a = { 'mode' },
    lualine_b = { 'branch' },
    lualine_c = {
      `+c+`,
    },
    lualine_x = {
      `+progress+`
      { 'diff', symbols = { added = icons.git.added .. %[2]s, modified = icons.git.modified .. %[2]s, removed = icons.git.removed .. %[2]s }%[4]s }%[5]s,
    },
    lualine_y = { { 'progress', separator = ' ', padding = { left = 1, right = 0 } }, { 'location', padding = { left = 0, right = 1 } } },
    lualine_z = { %[6]s },
  },
  extensions = { 'quickfix', 'oil', 'neo-tree', 'nvim-tree', 'lazy', 'mason', 'fzf', 'nvim-dap-ui', 'man' },
})`, Lua(nerd), gap, count, source, language, clock))
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
	if notify == "true" {
		b.WriteString(" styles = { notification = { wo = { wrap = true } } }, -- Long messages wrap instead of being cut off.\n")
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
 }, sections = { function(self) return require('user_config.logo').sections(self:size().width) end, { section = 'keys', gap = 1, padding = 1 }, { text = 'Press your leader key in a file to discover shortcuts.', align = 'center', padding = 1 }, { section = 'startup' } } },
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

// guidelessFiletypes are the start screens, sidebars and plugin windows that indent
// guides leave alone.
var guidelessFiletypes = []string{"help", "lazy", "mason", "notify", "snacks_dashboard", "snacks_notif", "ministarter", "neo-tree", "NvimTree", "oil", "minifiles", "noice", "dap-view", "dapui_scopes"}

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
	setup := func(plugin, module, call string) {
		b.WriteString(onLoad(plugin, "local m = require("+LuaString(module)+")\n"+call))
	}
	if Value(s, "workspace.start_screen", "none") == "mini.starter" {
		var items []string
		for _, item := range startItems(s) {
			items = append(items, fmt.Sprintf("{ name = %s, action = %s, section = 'Start' }", LuaString(item.label), LuaString(item.command)))
		}
		setup("mini.starter", "mini.starter", "local logo = require('user_config.logo')\nm.setup({ header = logo.header, items = { "+strings.Join(items, ", ")+" }, footer = 'Press your leader key in a file to discover shortcuts.',\n  content_hooks = { logo.paint, m.gen_hook.adding_bullet(), m.gen_hook.aligning('center', 'center') } })")
	}
	// Guides stay out of start screens, sidebars and plugin windows.
	quiet := luaList(guidelessFiletypes)
	switch Value(s, "appearance.indent", "none") {
	case "indent-blankline":
		// The scope around the cursor is highlighted where Tree-sitter reads the code.
		scope := Lua(Value(s, "syntax.engine", "native") == "nvim-treesitter")
		setup("indent-blankline", "ibl", "m.setup({ indent = { char = '│', tab_char = '│' }, scope = { enabled = "+scope+", show_start = false, show_end = false },\n  exclude = { filetypes = { "+quiet+" } } })")
	case "mini.indentscope":
		// Set from startup, so windows opened before the guides load are left alone too.
		b.WriteString("vim.api.nvim_create_autocmd('FileType', { group = vim.api.nvim_create_augroup('ShoshinIndentscope', { clear = true }),\n  pattern = { " + quiet + " }, callback = function() vim.b.miniindentscope_disable = true end })\n")
		setup("mini.indentscope", "mini.indentscope", "m.setup({ symbol = '│', options = { try_as_border = true } })")
	}
	if Value(s, "appearance.notifications", "none") == "fidget" {
		setup("fidget", "fidget", "m.setup({})")
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
