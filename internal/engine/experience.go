package engine

import "strings"

func appearanceLua(s State) string {
	var b strings.Builder
	if theme, ok := s.Values["appearance.colorscheme"]; ok {
		if theme == "tokyonight" {
			b.WriteString(`vim.cmd.colorscheme('habamax')
if not vim.g.shoshinvim_probe then
  local ok, theme = pcall(require, 'tokyonight')
  if ok then theme.setup({ style = 'night', styles = { sidebars = 'dark', floats = 'dark' } }); vim.cmd.colorscheme('tokyonight') end
end
`)
		} else {
			b.WriteString("vim.cmd.colorscheme(" + Lua(theme) + ")\n")
		}
	}
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
    extensions = { 'quickfix', 'oil' },
  }) end
end
`)
	}
	if Value(s, "appearance.buffers", "native") == "bufferline" {
		b.WriteString(`if not vim.g.shoshinvim_probe then
  local ok, line = pcall(require, 'bufferline')
  if ok then line.setup({ options = {
    diagnostics = 'nvim_lsp', separator_style = 'thin', show_buffer_icons = false,
    show_buffer_close_icons = false, show_close_icon = false,
    offsets = { { filetype = 'snacks_layout_box', text = 'PROJECT', text_align = 'left' } },
  } }) end
end
`)
	}
	if Value(s, "navigation.shortcuts", "native") == "which-key" {
		b.WriteString(`if not vim.g.shoshinvim_probe then
  local ok, wk = pcall(require, 'which-key')
  if ok then
    local key_names = {}
    for _, key in ipairs({ 'Up', 'Down', 'Left', 'Right', 'C', 'M', 'D', 'S', 'CR', 'Esc', 'ScrollWheelDown', 'ScrollWheelUp', 'NL', 'BS', 'Space', 'Tab', 'F1', 'F2', 'F3', 'F4', 'F5', 'F6', 'F7', 'F8', 'F9', 'F10', 'F11', 'F12' }) do key_names[key] = key end
    wk.setup({ preset = 'modern', delay = 300, icons = { mappings = false, keys = key_names, breadcrumb = '>', separator = '->', group = '+' } })
    wk.add({ { '<leader>f', group = 'Find' }, { '<leader>b', group = 'Buffers' }, { '<leader>c', group = 'Code' }, { '<leader>g', group = 'Git' }, { '<leader>w', group = 'Windows' }, { '<leader>u', group = 'View' } })
  end
end
`)
	}
	return b.String()
}

func snacksOptions(s State) string {
	if Value(s, "experience.layout", "native") != "rich" {
		return "{ picker = { enabled = true }, explorer = { enabled = " + Lua(Value(s, "navigation.explorer", "native") == "snacks") + " } }"
	}
	return `{ picker = { enabled = true, prompt = '> ', icons = { files = { enabled = false } } },
 explorer = { enabled = true }, input = { enabled = true }, indent = { enabled = true },
 notifier = { enabled = true, timeout = 3000 },
 dashboard = { enabled = true, preset = {
   header = 'S H O S H I N\n\nA fresh mind. A focused workspace.',
   keys = {
     { icon = '', key = 'f', desc = 'Find a file', action = ':ShoshinFiles' },
     { icon = '', key = 'e', desc = 'Explore this project', action = ':ShoshinExplore' },
     { icon = '', key = 'r', desc = 'Recent files', action = function() Snacks.picker.recent() end },
     { icon = '', key = 'n', desc = 'New file', action = ':ene | startinsert' },
     { icon = '', key = '?', desc = 'Shortcut guide', action = ':ShoshinHelp' },
     { icon = '', key = 'q', desc = 'Quit', action = ':qa' },
   },
 }, sections = { { section = 'header' }, { section = 'keys', gap = 1, padding = 1 }, { text = 'Press Space in a file to discover commands.', align = 'center', padding = 1 } } },
 }`
}

func workspaceLua(s State) string {
	if Value(s, "experience.layout", "native") != "rich" {
		return ""
	}
	return `-- Task-oriented shortcuts. User mappings load after these defaults.
local map = function(key, action, description) vim.keymap.set('n', key, action, { desc = description, silent = true }) end
local snacks = function(action, fallback)
  return function() local ok, ui = pcall(require, 'snacks'); if ok then action(ui) elseif fallback then fallback() else vim.notify('This feature is pending installation. Open ShoshinVim and /retry.') end end
end
map('<leader><space>', '<cmd>ShoshinFiles<CR>', 'Find file')
map('<leader>ff', '<cmd>ShoshinFiles<CR>', 'Find file')
map('<leader>fg', snacks(function(ui) ui.picker.grep() end), 'Search project text')
map('<leader>fr', snacks(function(ui) ui.picker.recent() end), 'Recent files')
map('<leader>fb', snacks(function(ui) ui.picker.buffers() end, function() vim.cmd.ls() end), 'Find buffer')
map('<leader>e', '<cmd>ShoshinExplore<CR>', 'Project explorer')
map('<S-h>', '<cmd>bprevious<CR>', 'Previous buffer')
map('<S-l>', '<cmd>bnext<CR>', 'Next buffer')
map('<leader>bd', snacks(function(ui) ui.bufdelete() end, function() vim.cmd.bdelete() end), 'Close buffer')
map('<leader>cf', '<cmd>ShoshinFormat<CR>', 'Format file')
map('<leader>ca', vim.lsp.buf.code_action, 'Code action')
map('<leader>cr', vim.lsp.buf.rename, 'Rename symbol')
map('<leader>cd', vim.diagnostic.open_float, 'Line diagnostics')
map('<leader>cl', snacks(function(ui) ui.picker.diagnostics() end, vim.diagnostic.setqflist), 'Project diagnostics')
map('gd', vim.lsp.buf.definition, 'Go to definition')
map('gr', vim.lsp.buf.references, 'Find references')
map('K', vim.lsp.buf.hover, 'Documentation')
map('<leader>gg', snacks(function(ui) ui.picker.git_status() end), 'Git status')
map('<leader>gd', snacks(function(ui) ui.picker.git_diff() end), 'Git diff')
map('<leader>gp', function() local ok, git = pcall(require, 'gitsigns'); if ok then git.preview_hunk() end end, 'Preview Git hunk')
map('<leader>ws', '<cmd>split<CR>', 'Split below')
map('<leader>wv', '<cmd>vsplit<CR>', 'Split right')
map('<leader>wd', '<cmd>close<CR>', 'Close window')
map('<C-h>', '<C-w>h', 'Focus left window')
map('<C-j>', '<C-w>j', 'Focus window below')
map('<C-k>', '<C-w>k', 'Focus window above')
map('<C-l>', '<C-w>l', 'Focus right window')
map('<leader>uw', function() vim.wo.wrap = not vim.wo.wrap end, 'Toggle line wrapping')
map('<leader>t', snacks(function(ui) ui.terminal.toggle() end, function() vim.cmd.terminal() end), 'Toggle terminal')
map('<Esc>', '<cmd>nohlsearch<CR>', 'Clear search highlight')
vim.keymap.set('i', '<C-space>', function() vim.lsp.completion.get() end, { desc = 'Complete from language server' })
vim.keymap.set('i', '<C-j>', function() return vim.fn.pumvisible() == 1 and '<C-n>' or '<C-j>' end, { expr = true, desc = 'Next completion' })
vim.keymap.set('i', '<C-k>', function() return vim.fn.pumvisible() == 1 and '<C-p>' or '<C-k>' end, { expr = true, desc = 'Previous completion' })
vim.keymap.set('s', '<Tab>', function() if vim.snippet.active({ direction = 1 }) then vim.snippet.jump(1) end end, { desc = 'Next snippet field' })
vim.keymap.set('t', '<Esc><Esc>', '<C-\\><C-n>', { desc = 'Leave terminal mode' })
vim.api.nvim_create_user_command('ShoshinHelp', function()
  local lines = {
    'SHOSHIN  /  WORKSPACE GUIDE', '',
    'Space opens your command groups. Pause to discover shortcuts.', '',
    'FIND      Space Space   files       Space fg   project text',
    '          Space e       explorer    Space fr   recent files',
    'CODE      gd            definition  K          documentation',
    '          Space ca      actions     Space cr   rename symbol',
    '          Space cf      format      Space cd   diagnostics',
    'BUFFERS   Shift H / L   switch      Space bd   close file',
    'WINDOWS   Ctrl h/j/k/l  focus       Space wv   split right',
    'GIT       Space gg      status      Space gp   preview hunk',
    'TERMINAL  Space t       toggle      Esc Esc    normal mode', '',
    'COMPLETION  Ctrl Space  request     Ctrl n/p   choose',
    '            Ctrl y      accept     Ctrl e     dismiss', '',
    'Customize: run shoshinvim. Manual Lua belongs in user.lua.',
    'This Neovim configuration works without ShoshinVim installed.', '',
    'Press q to close this guide.',
  }
  local buf = vim.api.nvim_create_buf(false, true)
  vim.api.nvim_buf_set_lines(buf, 0, -1, false, lines)
  vim.bo[buf].modifiable = false
  local width = math.min(74, vim.o.columns - 4)
  local height = math.min(#lines, vim.o.lines - 4)
  vim.api.nvim_open_win(buf, true, { relative = 'editor', width = width, height = height, row = math.max(0, math.floor((vim.o.lines-height)/2)-1), col = math.max(0, math.floor((vim.o.columns-width)/2)), border = 'rounded', title = ' Workspace guide ', title_pos = 'center' })
  vim.keymap.set('n', 'q', '<cmd>close<CR>', { buffer = buf, silent = true })
end, {})
map('<leader>?', '<cmd>ShoshinHelp<CR>', 'Workspace guide')
if not vim.g.shoshinvim_probe then
  vim.api.nvim_create_autocmd('VimEnter', { once = true, callback = function()
    if #vim.api.nvim_list_uis() > 0 and vim.fn.argc() > 0 and vim.o.columns >= 100 then
      vim.schedule(function() local ok, ui = pcall(require, 'snacks'); if ok then ui.explorer({ focus = false }) end end)
    end
  end })
end
`
}
