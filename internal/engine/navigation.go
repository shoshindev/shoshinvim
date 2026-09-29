package engine

import (
	"fmt"
	"strings"
)

// finderCommands are the searches every finder provides as Shoshin* commands, so
// shortcuts work the same whichever finder is chosen. Order is the output order.
var finderCommands = []struct{ name, desc string }{
	{"ShoshinFiles", "Find files"},
	{"ShoshinGrep", "Search text in the project"},
	{"ShoshinGrepWord", "Search the word under the cursor"},
	{"ShoshinRecent", "Recent files"},
	{"ShoshinBuffers", "Open buffers"},
	{"ShoshinDiagnostics", "Project diagnostics"},
	{"ShoshinHelpPages", "Search Neovim help"},
	{"ShoshinKeymaps", "Search keymaps"},
	{"ShoshinCommands", "Search commands"},
	{"ShoshinResume", "Reopen the last search"},
	{"ShoshinGitStatus", "Changed files (Git)"},
	{"ShoshinBufferLines", "Search in this file"},
	{"ShoshinConfigFiles", "Find Neovim config files"},
}

// Implementations per finder: module to require, then a Lua call on it named `m`.
// An empty entry falls back to Neovim's built-in behavior.
var finders = map[string]struct {
	module string
	calls  map[string]string
}{
	"snacks": {"snacks", map[string]string{
		"ShoshinFiles": "m.picker.files()", "ShoshinGrep": "m.picker.grep()", "ShoshinGrepWord": "m.picker.grep_word()",
		"ShoshinRecent": "m.picker.recent()", "ShoshinBuffers": "m.picker.buffers()", "ShoshinDiagnostics": "m.picker.diagnostics()",
		"ShoshinHelpPages": "m.picker.help()", "ShoshinKeymaps": "m.picker.keymaps()", "ShoshinCommands": "m.picker.commands()",
		"ShoshinResume": "m.picker.resume()", "ShoshinGitStatus": "m.picker.git_status()", "ShoshinBufferLines": "m.picker.lines()",
		"ShoshinConfigFiles": "m.picker.files({ cwd = config_root() })"}},
	"telescope": {"telescope.builtin", map[string]string{
		"ShoshinFiles": "m.find_files()", "ShoshinGrep": "m.live_grep()", "ShoshinGrepWord": "m.grep_string()",
		"ShoshinRecent": "m.oldfiles()", "ShoshinBuffers": "m.buffers()", "ShoshinDiagnostics": "m.diagnostics()",
		"ShoshinHelpPages": "m.help_tags()", "ShoshinKeymaps": "m.keymaps()", "ShoshinCommands": "m.commands()",
		"ShoshinResume": "m.resume()", "ShoshinGitStatus": "m.git_status()", "ShoshinBufferLines": "m.current_buffer_fuzzy_find()",
		"ShoshinConfigFiles": "m.find_files({ cwd = config_root() })"}},
	"fzf-lua": {"fzf-lua", map[string]string{
		"ShoshinFiles": "m.files()", "ShoshinGrep": "m.live_grep()", "ShoshinGrepWord": "m.grep_cword()",
		"ShoshinRecent": "m.oldfiles()", "ShoshinBuffers": "m.buffers()", "ShoshinDiagnostics": "m.diagnostics_workspace()",
		"ShoshinHelpPages": "m.helptags()", "ShoshinKeymaps": "m.keymaps()", "ShoshinCommands": "m.commands()",
		"ShoshinResume": "m.resume()", "ShoshinGitStatus": "m.git_status()", "ShoshinBufferLines": "m.blines()",
		"ShoshinConfigFiles": "m.files({ cwd = config_root() })"}},
	"mini.pick": {"mini.pick", map[string]string{
		"ShoshinFiles": "m.builtin.files()", "ShoshinGrep": "m.builtin.grep_live()",
		"ShoshinGrepWord": "m.builtin.grep({ pattern = vim.fn.expand('<cword>') })", "ShoshinBuffers": "m.builtin.buffers()",
		"ShoshinHelpPages": "m.builtin.help()", "ShoshinResume": "m.builtin.resume()",
		"ShoshinConfigFiles": "m.builtin.files(nil, { source = { cwd = config_root() } })"}},
}

// Built-in behavior, used without a finder plugin and where a finder lacks a feature.
var nativeCommands = map[string]string{
	"ShoshinFiles":       "vim.ui.input({ prompt = 'Find file: ', completion = 'file' }, function(path) if path and path ~= '' then vim.cmd.edit(vim.fn.fnameescape(path)) end end)",
	"ShoshinGrep":        "vim.ui.input({ prompt = 'Search text: ' }, function(text) if text and text ~= '' then vim.cmd('silent grep! ' .. vim.fn.shellescape(text)); vim.cmd.copen() end end)",
	"ShoshinGrepWord":    "vim.cmd('silent grep! ' .. vim.fn.shellescape(vim.fn.expand('<cword>'))); vim.cmd.copen()",
	"ShoshinRecent":      "vim.cmd('browse oldfiles')",
	"ShoshinBuffers":     "vim.cmd('ls'); vim.ui.input({ prompt = 'Buffer: ' }, function(b) if b and b ~= '' then vim.cmd.buffer(b) end end)",
	"ShoshinDiagnostics": "vim.diagnostic.setqflist()",
	"ShoshinHelpPages":   "vim.ui.input({ prompt = 'Help topic: ', completion = 'help' }, function(t) if t and t ~= '' then vim.cmd.help(t) end end)",
	"ShoshinKeymaps":     "vim.cmd('map')",
	"ShoshinCommands":    "vim.api.nvim_feedkeys(':', 'n', false)",
	"ShoshinResume":      "vim.notify('Resume needs a finder plugin', vim.log.levels.INFO)",
	"ShoshinGitStatus":   "vim.cmd('!git status --short')",
	"ShoshinBufferLines": "vim.api.nvim_feedkeys('/', 'n', false)",
	"ShoshinConfigFiles": "vim.cmd.edit(vim.fn.fnameescape(config_root()))",
}

// Explorers: how to open each one, and setup to run once it is loaded.
var explorers = map[string]struct{ module, open, setup string }{
	"snacks":     {"snacks", "m.explorer()", ""},
	"neo-tree":   {"neo-tree", "vim.cmd('Neotree toggle reveal')", "m.setup({ default_component_configs = { icon = { folder_closed = '+', folder_open = '-', folder_empty = '-', default = ' ' }, git_status = { symbols = { added = '+', modified = '~', deleted = 'x', renamed = '>', untracked = '?', ignored = '.', unstaged = '*', staged = '+', conflict = '!' } } }, filesystem = { follow_current_file = { enabled = true } } })"},
	"nvim-tree":  {"nvim-tree", "require('nvim-tree.api').tree.toggle({ find_file = true })", "m.setup({ renderer = { icons = { show = { file = false }, glyphs = { folder = { arrow_closed = '>', arrow_open = 'v', default = '+', open = '-', empty = '+', empty_open = '-', symlink = '@', symlink_open = '@' }, git = { unstaged = '*', staged = '+', unmerged = '!', renamed = '>', untracked = '?', deleted = 'x', ignored = '.' } } } } })"},
	"oil":        {"oil", "m.open()", ""},
	"mini.files": {"mini.files", "local path = vim.api.nvim_buf_get_name(0); m.open(path ~= '' and path or nil)", ""},
}

func navigationLua(s State) string {
	var b strings.Builder
	picker, _ := Value(s, "navigation.picker", "native").(string)
	explorer, _ := Value(s, "navigation.explorer", "native").(string)
	finder, hasFinder := finders[picker]
	b.WriteString(`-- Search commands. Shortcuts call these, so they work with any finder.
local function command(name, run, desc) vim.api.nvim_create_user_command(name, run, { desc = desc }) end
local function config_root() return vim.g.shoshinvim_config_root or vim.fn.stdpath('config') end
local function with(module, run)
  return function()
    local ok, m = pcall(require, module)
    if not ok then return vim.notify(module .. ' is not installed yet. Open ShoshinVim and download plugins.', vim.log.levels.WARN) end
    local done, err = pcall(run, m) -- For example Git status outside a Git repository.
    if not done then vim.notify(tostring(err), vim.log.levels.WARN) end
  end
end
`)
	if picker == "native" || picker == "mini.pick" {
		b.WriteString("if vim.fn.executable('rg') == 1 then vim.o.grepprg = 'rg --vimgrep --smart-case' end\n")
	}
	for _, c := range finderCommands {
		if call, ok := finder.calls[c.name]; hasFinder && ok {
			fmt.Fprintf(&b, "command(%s, with(%s, function(m) %s end), %s)\n", LuaString(c.name), LuaString(finder.module), call, LuaString(c.desc))
		} else {
			fmt.Fprintf(&b, "command(%s, function() %s end, %s)\n", LuaString(c.name), nativeCommands[c.name], LuaString(c.desc))
		}
	}
	if e, ok := explorers[explorer]; ok {
		if explorer == "nvim-tree" {
			b.WriteString("vim.g.loaded_netrw, vim.g.loaded_netrwPlugin = 1, 1 -- nvim-tree replaces netrw\n")
		}
		if e.setup != "" {
			fmt.Fprintf(&b, "if not vim.g.shoshinvim_probe then\n  local ok, m = pcall(require, %s)\n  if ok then %s end\nend\n", LuaString(e.module), e.setup)
		}
		fmt.Fprintf(&b, "command(\"ShoshinExplore\", with(%s, function(m) %s end), \"Project explorer\")\n", LuaString(e.module), e.open)
	} else {
		b.WriteString("command(\"ShoshinExplore\", function() vim.cmd.Explore() end, \"Project explorer\")\n")
	}
	b.WriteString(`command("ShoshinTerminal", function()
  local ok, snacks = pcall(require, 'snacks')
  if ok then snacks.terminal.toggle() else vim.cmd('botright 12split | terminal') end
end, "Terminal")
command("ShoshinBufferDelete", function()
  local ok, snacks = pcall(require, 'snacks')
  if ok then snacks.bufdelete() else vim.cmd.bdelete() end
end, "Close buffer")
`)
	return b.String()
}

// gitLua gives every Git choice the same hunk commands, so shortcuts work with either.
func gitLua(s State) string {
	var calls map[string]string
	module := ""
	switch Value(s, "git.hunks", "native") {
	case "gitsigns":
		module = "gitsigns"
		calls = map[string]string{
			"ShoshinHunkNext":    "if vim.wo.diff then vim.cmd.normal({ ']c', bang = true }) else m.nav_hunk('next') end",
			"ShoshinHunkPrev":    "if vim.wo.diff then vim.cmd.normal({ '[c', bang = true }) else m.nav_hunk('prev') end",
			"ShoshinHunkPreview": "m.preview_hunk()", "ShoshinHunkStage": "m.stage_hunk()", "ShoshinHunkReset": "m.reset_hunk()",
			"ShoshinBlameLine": "m.blame_line({ full = true })", "ShoshinGitDiff": "m.diffthis()",
		}
	case "mini.diff":
		module = "mini.diff"
		calls = map[string]string{
			"ShoshinHunkNext": "m.goto_hunk('next')", "ShoshinHunkPrev": "m.goto_hunk('prev')",
			"ShoshinHunkPreview": "m.toggle_overlay()", "ShoshinGitDiff": "m.toggle_overlay()",
			"ShoshinHunkStage": "local l = vim.fn.line('.'); m.do_hunks(0, 'apply', { line_start = l, line_end = l })",
			"ShoshinHunkReset": "local l = vim.fn.line('.'); m.do_hunks(0, 'reset', { line_start = l, line_end = l })",
			"ShoshinBlameLine": "vim.notify('Blame needs gitsigns.nvim', vim.log.levels.INFO)",
		}
	default:
		return ""
	}
	var b strings.Builder
	b.WriteString("-- Git change commands; shortcuts call these, so they work with gitsigns or mini.diff.\n")
	b.WriteString("local function command(name, run, desc) vim.api.nvim_create_user_command(name, run, { desc = desc }) end\n")
	for _, c := range []struct{ name, desc string }{{"ShoshinHunkNext", "Next Git change"}, {"ShoshinHunkPrev", "Previous Git change"}, {"ShoshinHunkPreview", "Preview Git change"},
		{"ShoshinHunkStage", "Stage Git change"}, {"ShoshinHunkReset", "Undo Git change"}, {"ShoshinBlameLine", "Blame this line"}, {"ShoshinGitDiff", "Diff this file"}} {
		fmt.Fprintf(&b, "command(%s, function()\n  local ok, m = pcall(require, %s)\n  if ok then %s end\nend, %s)\n", LuaString(c.name), LuaString(module), calls[c.name], LuaString(c.desc))
	}
	return b.String()
}
