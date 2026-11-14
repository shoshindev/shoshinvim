package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
)

// TestChoicesInRealNeovim downloads every plugin a choice needs at its pinned revision
// and checks the generated configuration in real Neovim. It needs the network, git and
// Neovim, so it only runs when SHOSHINVIM_INTEGRATION=1. UPDATE_THEME_SWATCHES=1 also
// records each theme's real colors into data/theme-swatches.json.
// activeColorscheme accepts the exact name or the family name some themes report
// (kanagawa-wave reports kanagawa), and requires a loaded theme, not the fallback.
func activeColorscheme(name string) string {
	return "local active = tostring(vim.g.colors_name); assert(active == " + LuaString(name) + " or (" + LuaString(name) + "):sub(1, #active) == active and active ~= 'habamax', 'active colorscheme: ' .. active)"
}

func TestChoicesInRealNeovim(t *testing.T) {
	if os.Getenv("SHOSHINVIM_INTEGRATION") != "1" {
		t.Skip("set SHOSHINVIM_INTEGRATION=1 to download plugins and test every choice")
	}
	nvim, _, err := Neovim(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))

	type choice struct {
		name   string
		values map[string]any
		check  string // Lua that errors on failure
	}
	var cases []choice
	for _, theme := range Themes() {
		if theme.Hidden {
			continue
		}
		for _, v := range theme.Variants {
			cases = append(cases, choice{"theme " + v.ID, map[string]any{"appearance.colorscheme": v.ID}, activeColorscheme(v.Colorscheme)})
		}
	}
	custom, _ := PluginInfo("kanagawa") // Any real repository works; this one is known.
	cases = append(cases,
		choice{"custom theme", map[string]any{"appearance.colorscheme": CustomTheme, "appearance.custom_theme.repository": custom.Repository,
			"appearance.custom_theme.revision": custom.Revision, "appearance.custom_theme.colorscheme": "kanagawa-dragon"},
			activeColorscheme("kanagawa-dragon")},
		choice{"git mini.diff", map[string]any{"git.hunks": "mini.diff"}, "assert(_G.MiniDiff, 'mini.diff not set up')"},
		choice{"git gitsigns", map[string]any{"git.hunks": "gitsigns"}, openFile + "assert(package.loaded['gitsigns'], 'gitsigns not loaded')"},
		choice{"statusline lualine", map[string]any{"appearance.statusline": "lualine"}, "assert(package.loaded['lualine'] and vim.o.statusline:find('lualine'), vim.o.statusline)"},
		choice{"statusline mini", map[string]any{"appearance.statusline": "mini.statusline"}, "assert(_G.MiniStatusline and vim.o.statusline:find('MiniStatusline'), vim.o.statusline)"},
		// Until it loads, the start screen has no status line; then it comes back,
		choice{"startup status line", map[string]any{"appearance.statusline": "mini.statusline"}, `assert(vim.o.laststatus == 0, 'a status line before mini.statusline loaded')
` + firstScreenDrawn + `
assert(_G.MiniStatusline and vim.o.laststatus == 2, 'laststatus is ' .. vim.o.laststatus .. ' once mini.statusline loaded')`},
		// unless user.lua has chosen one meanwhile.
		choice{"status line from user.lua", map[string]any{"appearance.statusline": "mini.statusline"}, "assert(_G.MiniStatusline and vim.o.laststatus == 3, 'user.lua chose 3, now ' .. vim.o.laststatus)"},
		// A plugin module that fails leaves the other modules working, as init.lua promises.
		choice{"broken plugin module", map[string]any{"navigation.picker": "telescope", "navigation.explorer": "neo-tree", "completion.engine": "blink.cmp",
			"language.lua.lsp": true, "formatting.provider": "conform"}, `for _, name in ipairs({ 'ShoshinFiles', 'ShoshinExplore', 'ShoshinFormat' }) do
  assert(vim.fn.exists(':' .. name) == 2, name .. ' went missing with the plugin module')
end
local ok, attach = pcall(vim.api.nvim_get_autocmds, { group = 'ShoshinLspAttach' })
assert(ok and #attach > 0, 'language servers went missing with the plugin module')`},
		choice{"tabs bufferline", map[string]any{"appearance.buffers": "bufferline"}, "assert(package.loaded['bufferline'], 'bufferline not loaded')\n" + tabsCheck},
		choice{"tabs barbar", map[string]any{"appearance.buffers": "barbar"}, "assert(package.loaded['barbar'], 'barbar not loaded')\n" + tabsCheck},
		choice{"tabs mini", map[string]any{"appearance.buffers": "mini.tabline"}, "assert(_G.MiniTabline and vim.o.tabline:find('MiniTabline'), vim.o.tabline)"},
		choice{"completion blink", map[string]any{"completion.engine": "blink.cmp", "language.lua.lsp": true},
			openFile + "assert(package.loaded['blink.cmp']); local c = vim.lsp.config['*']; assert(c and c.capabilities and c.capabilities.textDocument.completion.completionItem.snippetSupport, 'capabilities')"},
		choice{"completion nvim-cmp", map[string]any{"completion.engine": "nvim-cmp", "language.lua.lsp": true},
			openFile + "local cmp = require('cmp'); assert(#cmp.get_config().sources >= 3, 'sources'); local c = vim.lsp.config['*']; assert(c and c.capabilities and c.capabilities.textDocument.completion.completionItem.snippetSupport, 'capabilities')"},
	)
	for _, picker := range []string{"native", "snacks", "telescope", "fzf-lua", "mini.pick"} {
		var commands []string
		for _, c := range finderCommands {
			switch c.name {
			case "ShoshinFiles", "ShoshinGrep", "ShoshinBuffers", "ShoshinHelpPages", "ShoshinRecent", "ShoshinCommands":
				if picker == "native" {
					continue // These wait for typed input, which a headless test cannot give.
				}
			}
			commands = append(commands, LuaString(c.name))
		}
		opens, prepare := "true", ""
		switch picker {
		case "native":
		case "mini.pick":
			// mini.pick reads keys in its own loop; queue Esc so each picker opens and closes.
			prepare = "vim.api.nvim_feedkeys(vim.keycode('<Esc>'), 't', false)"
		default:
			opens = "vim.wait(4000, function() return #vim.api.nvim_list_wins() > before end, 50)"
		}
		cases = append(cases, choice{"finder " + picker, map[string]any{"navigation.picker": picker},
			`for _, name in ipairs({ ` + strings.Join(commands, ", ") + ` }) do
  local before = #vim.api.nvim_list_wins()
  ` + prepare + `
  local ok, err = pcall(vim.cmd, name)
  assert(ok, name .. ': ' .. tostring(err))
  if name == 'ShoshinFiles' then assert(` + opens + `, 'no picker window opened') end
  vim.wait(300, function() return false end)
  for _, w in ipairs(vim.api.nvim_list_wins()) do
    local ok, cfg = pcall(vim.api.nvim_win_get_config, w) -- Pickers close their own windows too.
    if ok and cfg.relative ~= '' then pcall(vim.api.nvim_win_close, w, true) end
  end
  vim.cmd('stopinsert')
end`})
	}
	for explorer, filetype := range map[string]string{"snacks": "snacks_picker_list", "neo-tree": "neo-tree", "nvim-tree": "NvimTree", "oil": "oil", "mini.files": "minifiles", "native": "netrw"} {
		cases = append(cases, choice{"explorer " + explorer, map[string]any{"navigation.explorer": explorer},
			`vim.cmd('ShoshinExplore')
local found = vim.wait(4000, function()
  for _, w in ipairs(vim.api.nvim_list_wins()) do
    if vim.bo[vim.api.nvim_win_get_buf(w)].filetype == ` + LuaString(filetype) + ` then return true end
  end
end, 50)
assert(found, 'no ` + filetype + ` window')`})
	}

	// Every shortcut of each style exists, and the commands it calls are defined.
	for _, style := range []map[string]any{
		{"keymaps.style": "kickstart", "navigation.picker": "telescope", "navigation.explorer": "neo-tree", "git.hunks": "gitsigns"},
		{"keymaps.style": "lazyvim", "navigation.picker": "snacks", "navigation.explorer": "snacks", "git.hunks": "mini.diff", "keymaps.leader": "comma"},
	} {
		s := NewState()
		for id, v := range style {
			s.Values[id] = v
		}
		var checks []string
		for _, k := range EffectiveKeymaps(s) {
			a, _ := keyAction(k.Action)
			for _, mode := range k.Mode {
				checks = append(checks, fmt.Sprintf("assert(vim.fn.maparg(%s, %s) ~= '', %s)", LuaString(k.Key), LuaString(string(mode)), LuaString("missing "+string(mode)+" "+k.Key)))
			}
			if cmd, ok := strings.CutPrefix(a.RHS, "<cmd>"); ok && strings.HasPrefix(cmd, "Shoshin") {
				name := strings.TrimSuffix(cmd, "<CR>")
				checks = append(checks, fmt.Sprintf("assert(vim.fn.exists(':%s') == 2, 'missing command %s')", name, name))
			}
		}
		leader := map[any]string{"comma": ","}[style["keymaps.leader"]]
		if leader == "" {
			leader = " "
		}
		checks = append(checks, "assert(vim.g.mapleader == "+LuaString(leader)+", 'leader')", "assert(vim.fn.exists(':ShoshinHelp') == 2, 'guide')")
		cases = append(cases, choice{"shortcuts " + style["keymaps.style"].(string), style, strings.Join(checks, "\n")})
	}
	cases = append(cases,
		// Each plugin manager case has a data folder of its own, so lazy.nvim finds no other
		// case's plugins there.
		choice{"plugin manager", map[string]any{"git.hunks": "gitsigns"}, lazyCheck},
		// Earlier versions downloaded lazy.nvim to plugins/lazy; downloading again moves it,
		choice{"plugin manager moved from earlier folder", map[string]any{"git.hunks": "gitsigns"}, lazyCheck},
		// and until then plugins still load from there.
		choice{"plugin manager in earlier folder", map[string]any{"git.hunks": "gitsigns"}, openFile + "assert(package.loaded['gitsigns'], 'gitsigns not loaded')"},
		// Files an earlier version generated, such as ones restored from history, find it
		// through the link the move leaves.
		choice{"plugin manager earlier files", map[string]any{"git.hunks": "gitsigns"}, lazyCheck},
		choice{"shortcut command", map[string]any{"keymaps.style": "none"}, "assert(vim.fn.maparg('<leader>L', 'n'):find('Lazy'), 'custom command shortcut')"},
		choice{"start snacks", map[string]any{"workspace.start_screen": "snacks"}, "assert(require('snacks').config.dashboard.enabled, 'dashboard off')\n" +
			"Snacks.dashboard.open({ win = vim.api.nvim_get_current_win(), buf = vim.api.nvim_get_current_buf() })\n" + logoCheck},
		choice{"start mini.starter", map[string]any{"workspace.start_screen": "mini.starter"}, "assert(_G.MiniStarter, 'mini.starter not set up')\nMiniStarter.open()\n" + logoCheck},
		choice{"indent blankline", map[string]any{"appearance.indent": "indent-blankline"}, openFile + "assert(package.loaded['ibl'], 'ibl not loaded')"},
		choice{"indent snacks", map[string]any{"appearance.indent": "snacks"}, "assert(require('snacks').config.indent.enabled, 'snacks indent off')"},
		choice{"indent mini", map[string]any{"appearance.indent": "mini.indentscope"}, openFile + "assert(_G.MiniIndentscope, 'mini.indentscope not set up')"},
		choice{"notify snacks", map[string]any{"appearance.notifications": "snacks"}, "assert(require('snacks').config.notifier.enabled, 'notifier off')"},
		// fidget loads when a language server attaches; the event stands in for one.
		choice{"notify fidget", map[string]any{"appearance.notifications": "fidget"}, `assert(not package.loaded['fidget'], 'fidget loaded before a language server attached')
vim.api.nvim_exec_autocmds('LspAttach', { buffer = 0, data = { client_id = 0 } })
assert(package.loaded['fidget'], 'fidget not loaded')`},
		choice{"hints mini.clue", map[string]any{"navigation.shortcuts": "mini.clue", "keymaps.style": "kickstart"}, "assert(_G.MiniClue, 'mini.clue not set up')"},
		choice{"hints which-key", map[string]any{"navigation.shortcuts": "which-key", "keymaps.style": "lazyvim"}, "assert(package.loaded['which-key'], 'which-key not loaded')"},
		choice{"icons nerd", map[string]any{"appearance.icons": "nerd", "appearance.statusline": "lualine", "appearance.buffers": "bufferline", "navigation.explorer": "neo-tree",
			"navigation.shortcuts": "which-key", "keymaps.style": "lazyvim", "workspace.start_screen": "snacks"}, nerdIconsCheck},
		choice{"icons plain", map[string]any{"appearance.icons": "plain", "appearance.cmdline": "noice", "appearance.statusline": "lualine", "appearance.buffers": "bufferline", "navigation.explorer": "neo-tree",
			"navigation.picker": "snacks", "appearance.notifications": "snacks", "navigation.shortcuts": "which-key", "keymaps.style": "lazyvim",
			"completion.engine": "blink.cmp", "language.lua.lsp": true}, plainIconsCheck},
		choice{"polish rounded", map[string]any{"options.winborder": "rounded", "options.fillchars": "eob: ", "navigation.shortcuts": "which-key", "keymaps.style": "lazyvim",
			"navigation.picker": "telescope", "navigation.explorer": "neo-tree", "appearance.statusline": "lualine", "appearance.indent": "indent-blankline",
			"language.lua.lsp": true, "diagnostics.virtual_text": true, "completion.engine": "blink.cmp"}, `assert(require('which-key.config').options.win.border == 'rounded', 'which-key border')
assert(require('lazy.core.config').options.ui.border == 'rounded', 'lazy.nvim border')
assert(vim.o.pumborder == 'rounded' and vim.o.smoothscroll and vim.o.splitkeep == 'screen' and not vim.o.ruler, 'refinements')
assert(vim.o.fillchars:find('eob: ', 1, true) and vim.o.fillchars:find('diff:', 1, true), 'fillchars ' .. vim.o.fillchars)
local inline = vim.diagnostic.config().virtual_text
assert(type(inline) == 'table' and inline.prefix == '●' and vim.diagnostic.config().signs.text, 'diagnostics')
assert(require('telescope.config').values.sorting_strategy == 'ascending', 'telescope layout')
assert(require('neo-tree').ensure_config().close_if_last_window, 'neo-tree')
assert(#vim.api.nvim_get_autocmds({ group = 'ShoshinLspAttach' }) == 1, 'no reference highlighting')`},
		choice{"polish single", map[string]any{"options.winborder": "single", "navigation.picker": "fzf-lua", "navigation.explorer": "nvim-tree",
			"completion.engine": "nvim-cmp", "language.lua.lsp": true, "diagnostics.virtual_text": false, "appearance.indent": "mini.indentscope"},
			`assert(require('fzf-lua.config').globals.winopts.border == 'single', 'fzf-lua border')
assert(require('cmp.config').get().window.completion.border == 'single', 'completion border')
assert(vim.o.pumborder == 'single', 'pumborder ' .. vim.o.pumborder)
assert(vim.diagnostic.config().virtual_text == false, 'signs only')
vim.cmd('ShoshinExplore')
assert(vim.wait(4000, function()
  for _, w in ipairs(vim.api.nvim_list_wins()) do
    if vim.bo[vim.api.nvim_win_get_buf(w)].filetype == 'NvimTree' then return vim.api.nvim_win_get_width(w) == 30 end
  end
end, 50), 'no 30 column file tree')`},
		choice{"polish folds", map[string]any{"syntax.engine": "nvim-treesitter", "tools.provider": "mason", "options.number": true}, `assert(vim.o.foldmethod == 'expr' and vim.o.foldlevel == 99, 'folds ' .. vim.o.foldmethod)
local file = vim.fn.tempname() .. '.lua'
vim.fn.writefile({ 'local function f()', '  return 1', 'end' }, file)
vim.cmd.edit(file)
vim.treesitter.get_parser(0):parse()
assert(vim.fn.foldlevel(2) > 0, 'a function is not a fold')`},
		choice{"noice snacks", map[string]any{"appearance.cmdline": "noice", "appearance.notifications": "snacks", "appearance.statusline": "lualine",
			"completion.engine": "blink.cmp", "language.lua.lsp": true}, `local config = require('noice.config')
assert(vim.wait(2000, config.is_running), 'noice is not running')
assert(config.options.lsp.signature.enabled == false, 'blink.cmp keeps signature help')
assert(config.options.lsp.progress.enabled, 'noice shows language server progress')
assert(require('blink.cmp.config').cmdline.completion.menu.auto_show, 'no command dropdown')
vim.notify('from noice', vim.log.levels.WARN)
assert(vim.wait(2000, function()
  for _, w in ipairs(vim.api.nvim_list_wins()) do
    if vim.bo[vim.api.nvim_win_get_buf(w)].filetype == 'snacks_notif' then return true end
  end
end, 50), 'noice did not hand the notification to snacks')`},
		choice{"noice fidget", map[string]any{"appearance.cmdline": "noice", "appearance.notifications": "fidget", "completion.engine": "nvim-cmp", "language.lua.lsp": true},
			`local config = require('noice.config')
assert(vim.wait(2000, config.is_running), 'noice is not running')
assert(config.options.lsp.progress.enabled == false, 'fidget keeps language server progress')
assert(config.options.lsp.signature.enabled, 'noice shows signature help')
assert(config.options.lsp.override['cmp.entry.get_documentation'], 'nvim-cmp documentation')`},
		choice{"detect indent", map[string]any{"editor.detect_indent": "guess-indent"}, openFile + "assert(package.loaded['guess-indent'], 'guess-indent not loaded')"},
		choice{"editor behaviors", map[string]any{"editor.highlight_yank": true, "editor.trim_whitespace": true, "editor.restore_cursor": true, "editor.spell_prose": true},
			"local events = {}; for _, a in ipairs(vim.api.nvim_get_autocmds({ group = 'ShoshinEditor' })) do events[a.event] = true end; for _, e in ipairs({ 'TextYankPost', 'BufWritePre', 'BufReadPost', 'FileType' }) do assert(events[e], 'missing ' .. e) end"},
	)
	swatches := map[string][]string{}
	swatchLua := `local out = {}
for _, g in ipairs({ 'Normal', 'Keyword', 'Function', 'String', 'Type', 'Constant' }) do
  local hl = vim.api.nvim_get_hl(0, { name = g, link = false })
  local c = g == 'Normal' and (hl.bg or hl.fg) or hl.fg
  if c then out[#out + 1] = string.format('#%06x', c) end
end
io.stdout:write('SWATCH ' .. vim.json.encode(out) .. '\n')`
	// Tree-sitter: every parser compiles with Mason's tree-sitter CLI, starts on its file
	// type, and matches the source the pinned nvim-treesitter names.
	syntax := map[string]any{"syntax.engine": "nvim-treesitter", "tools.provider": "mason",
		"language.typescript.lsp": true, "language.css.lsp": true, "language.json.lsp": true}
	var parserChecks []string
	for _, p := range Parsers(State{Values: syntax}) {
		parserChecks = append(parserChecks, fmt.Sprintf("{ %s, %s }", LuaString(p.Language), LuaString(p.Revision)))
	}
	cases = append(cases, choice{"syntax treesitter", syntax, `local pinned = require('nvim-treesitter.parsers')
for _, p in ipairs({ ` + strings.Join(parserChecks, ", ") + ` }) do
  local lang, revision = p[1], p[2]
  assert(pinned[lang].install_info.revision == revision, lang .. ' source differs from nvim-treesitter')
  local file = vim.api.nvim_get_runtime_file('parser/' .. lang .. '.so', false)[1]
  assert(file and file:find('shoshinvim/parsers', 1, true), lang .. ' parser not installed: ' .. tostring(file))
end
for ft, lang in pairs({ typescriptreact = 'tsx', typescript = 'typescript', markdown = 'markdown', sh = 'bash', scss = 'scss' }) do
  local buf = vim.api.nvim_create_buf(true, false)
  vim.api.nvim_set_current_buf(buf)
  vim.bo[buf].filetype = ft
  assert(vim.treesitter.highlighter.active[buf], ft .. ': Tree-sitter did not start')
  assert(vim.treesitter.get_parser(buf):lang() == lang, ft .. ': wrong parser')
end
vim.bo.filetype = 'typescript'
assert(vim.bo.indentexpr:find('nvim%-treesitter'), 'indentexpr ' .. vim.bo.indentexpr)`})
	// Coding helpers are checked by what they do to text, not only by loading.
	edit := func(lines, cursor, keys, want string) string {
		return fmt.Sprintf(`local buf = vim.api.nvim_create_buf(true, false)
vim.api.nvim_set_current_buf(buf)
vim.bo[buf].filetype = 'lua'
vim.api.nvim_buf_set_lines(buf, 0, -1, false, { %s })
vim.api.nvim_win_set_cursor(0, { 1, %s })
vim.api.nvim_feedkeys(vim.api.nvim_replace_termcodes(%s, true, false, true), 'mx', false)
local got = vim.api.nvim_buf_get_lines(buf, 0, 1, false)[1]
assert(got == %s, 'got ' .. vim.inspect(got))`, LuaString(lines), cursor, LuaString(keys), LuaString(want))
	}
	todoCheck := func(namespace string) string {
		return `local file = vim.fn.tempname() .. '.lua'
vim.fn.writefile({ '-- TODO: tidy this up', 'local x = 1' }, file)
vim.cmd.edit(file)
local buf = vim.api.nvim_get_current_buf()
-- A headless Neovim never redraws, so parse as a screen update would.
pcall(function() vim.treesitter.get_parser(buf):parse(true) end)
local function marks()
  local n = 0
  for name, id in pairs(vim.api.nvim_get_namespaces()) do
    if name:find(` + LuaString(namespace) + `, 1, true) then n = n + #vim.api.nvim_buf_get_extmarks(buf, id, 0, -1, {}) end
  end
  return n
end
assert(vim.wait(5000, function() return marks() > 0 end, 50), 'TODO is not highlighted')`
	}
	debugSession := `local dap = require('dap')
local file = vim.fn.tempname() .. '.js'
vim.fn.writefile({ 'let total = 0', 'for (let i = 0; i < 3; i++) {', '  total += i', '}', 'console.log(total)' }, file)
vim.cmd.edit(file)
vim.api.nvim_win_set_cursor(0, { 3, 0 })
dap.toggle_breakpoint()
local stopped
dap.listeners.after.event_stopped.test = function(_, body) stopped = body end
dap.run(dap.configurations.javascript[1])
assert(vim.wait(60000, function() return stopped ~= nil end, 100), 'the debugger did not stop at the breakpoint')
assert(stopped.reason == 'breakpoint', 'stopped for ' .. tostring(stopped.reason))
assert(vim.fn.exists(':ShoshinDebugView') == 2, 'no debugger view command')
dap.terminate()
vim.wait(5000, function() return dap.session() == nil end, 100)`
	cases = append(cases,
		choice{"pairs nvim-autopairs", map[string]any{"coding.pairs": "nvim-autopairs"}, "vim.cmd('doautocmd BufEnter')\n" + edit("", "0", "a(<Esc>", "()")},
		choice{"pairs mini.pairs", map[string]any{"coding.pairs": "mini.pairs"}, edit("", "0", "a(<Esc>", "()")},
		choice{"surround mini.surround", map[string]any{"coding.surround": "mini.surround"}, edit("word", "0", "saiw)", "(word)")},
		choice{"surround mini.surround lazyvim", map[string]any{"coding.surround": "mini.surround", "keymaps.style": "lazyvim"}, edit("word", "0", "gsaiw)", "(word)")},
		choice{"surround nvim-surround", map[string]any{"coding.surround": "nvim-surround"}, edit("word", "0", "ysiw)", "(word)")},
		choice{"todo todo-comments", map[string]any{"coding.todo": "todo-comments"}, todoCheck("todo-comments")},
		choice{"todo mini.hipatterns", map[string]any{"coding.todo": "mini.hipatterns"}, todoCheck("MiniHipatterns")},
		choice{"textobjects mini.ai", map[string]any{"coding.textobjects": "mini.ai"}, edit("f(a, b)", "2", "ciaX<Esc>", "f(X, b)")},
		choice{"lint nvim-lint", map[string]any{"coding.lint": "nvim-lint", "tools.provider": "mason", "language.typescript.lsp": true}, `local dir = vim.fn.tempname()
vim.fn.mkdir(dir, 'p')
vim.fn.writefile({ '# Title', '## Skipped a blank line' }, dir .. '/notes.md')
vim.cmd.edit(dir .. '/notes.md')
assert(vim.wait(20000, function() return #vim.diagnostic.get(0) > 0 end, 100), 'markdownlint reported nothing')
-- Without an ESLint configuration, ESLint must not run (and so cannot complain).
vim.fn.writefile({ 'const x = 1' }, dir .. '/a.ts')
vim.cmd.edit(dir .. '/a.ts')
vim.cmd('ShoshinLint')
vim.wait(1000)
assert(#require('lint').get_running() == 0 and #vim.diagnostic.get(0) == 0, 'ESLint ran without a project configuration')`},
		choice{"debug nvim-dap-ui", map[string]any{"coding.debug": "nvim-dap-ui", "tools.provider": "mason", "language.typescript.lsp": true, "keymaps.style": "kickstart"},
			debugSession + "\nassert(vim.fn.maparg('<F5>', 'n') ~= '', 'F5 is not mapped')"},
		choice{"debug nvim-dap-view", map[string]any{"coding.debug": "nvim-dap-view", "tools.provider": "mason", "language.typescript.lsp": true, "keymaps.style": "lazyvim"},
			debugSession + "\nassert(vim.fn.maparg('<leader>db', 'n') ~= '', 'Space db is not mapped')"},
		// A plugin's own commands load it before any file is open, set up first.
		choice{"todo command before a file", map[string]any{"coding.todo": "todo-comments"}, `assert(not package.loaded['todo-comments'], 'todo-comments loaded before a file was opened')
vim.cmd('TodoQuickFix')
assert(require('todo-comments.config').options.signs == false, 'TodoQuickFix ran before todo-comments was set up')`},
		choice{"debug view command", map[string]any{"coding.debug": "nvim-dap-view"}, `assert(not package.loaded['dap'], 'nvim-dap loaded before it was needed')
vim.cmd('DapViewToggle')
assert(require('dap-view.setup').config.auto_toggle, 'DapViewToggle opened the view before it was set up')`},
	)
	// Plugins load when first needed: at startup only those that must, then with the first
	// screen, the first file, the first typing and the first search.
	full := map[string]any{"appearance.icons": "nerd", "appearance.statusline": "lualine", "appearance.buffers": "bufferline", "navigation.shortcuts": "which-key",
		"keymaps.style": "kickstart", "navigation.picker": "telescope", "navigation.explorer": "neo-tree", "git.hunks": "gitsigns", "workspace.start_screen": "snacks",
		"appearance.notifications": "snacks", "appearance.cmdline": "noice", "appearance.indent": "indent-blankline", "completion.engine": "blink.cmp",
		"language.lua.lsp": true, "lsp.provider": "lspconfig", "formatting.provider": "conform", "tools.provider": "mason", "syntax.engine": "nvim-treesitter",
		"coding.pairs": "nvim-autopairs", "coding.surround": "mini.surround", "coding.textobjects": "mini.ai", "coding.todo": "todo-comments",
		"editor.detect_indent": "guess-indent", "appearance.colorscheme": "tokyonight"}
	stages := map[string][]string{}
	fullState := NewState()
	for id, v := range full {
		fullState.Values[id] = v
	}
	for _, id := range Plugins(fullState) {
		rule := loadRules[id]
		switch {
		case rule.startup || id == themePlugin(fullState):
			stages["startup"] = append(stages["startup"], id)
		case slices.Contains(rule.events, "VeryLazy"):
			stages["screen"] = append(stages["screen"], id)
		case slices.Contains(rule.events, "BufNewFile"):
			stages["file"] = append(stages["file"], id)
		case slices.Contains(rule.events, "InsertEnter"):
			stages["insert"] = append(stages["insert"], id)
		}
	}
	stages["startup"] = append(stages["startup"], "lazy.nvim")
	slices.Sort(stages["startup"])
	cases = append(cases, choice{"startup loading", full, `local function loaded()
  local names = {}
  for _, p in ipairs(require('lazy').plugins()) do
    if p._.loaded then names[#names + 1] = p.name end
  end
  table.sort(names)
  return names
end
local function expect(when, names)
  local now = {}
  for _, name in ipairs(loaded()) do now[name] = true end
  for _, name in ipairs(names) do
    assert(now[name], name .. ' did not load ' .. when .. '; loaded: ' .. table.concat(loaded(), ' '))
  end
end
local start = table.concat(loaded(), ' ')
assert(start == ` + LuaString(strings.Join(stages["startup"], " ")) + `, 'loaded at startup: ' .. start)
` + firstScreenDrawn + `
expect('with the first screen', { ` + luaList(stages["screen"]) + ` })
` + openFile + `expect('with the first file', { ` + luaList(stages["file"]) + ` })
vim.api.nvim_feedkeys(vim.keycode('i<Esc>'), 'mx', false)
expect('when typing', { ` + luaList(stages["insert"]) + ` })
vim.cmd('ShoshinFiles')
expect('when searching', { 'telescope' })
-- Set up as it loads, so even the first picker has the chosen layout.
local prompt
assert(vim.wait(4000, function()
  prompt = vim.api.nvim_get_current_buf()
  return vim.bo[prompt].filetype == 'TelescopePrompt'
end, 50), 'no Telescope prompt')
local picker = require('telescope.actions.state').get_current_picker(prompt)
assert(picker.sorting_strategy == 'ascending', 'the first picker was opened before Telescope was set up')`})
	// Themes style the plugins chosen with them: each group below is set by the theme
	// itself (colors, not the plugin's default link).
	workspace := map[string]any{"git.hunks": "gitsigns", "navigation.picker": "telescope", "navigation.explorer": "neo-tree", "navigation.shortcuts": "which-key",
		"keymaps.style": "lazyvim", "appearance.indent": "indent-blankline", "appearance.buffers": "bufferline", "appearance.notifications": "snacks",
		"workspace.start_screen": "snacks", "appearance.cmdline": "noice", "completion.engine": "blink.cmp", "language.lua.lsp": true}
	for theme, groups := range map[string][]string{
		"tokyonight":           {"GitSignsAdd", "NeoTreeNormal", "TelescopeBorder", "WhichKeyGroup", "SnacksIndent", "BlinkCmpMenu"},
		"solarized-osaka/dark": {"GitSignsAdd", "NeoTreeNormal", "TelescopeBorder", "WhichKeyGroup", "SnacksIndent", "BlinkCmpMenu"},
		// catppuccin links some groups on purpose (TelescopeBorder to FloatBorder), so it is
		// checked by groups it colors.
		"catppuccin/mocha": {"GitSignsAdd", "NeoTreeNormal", "TelescopeSelection", "WhichKeyGroup", "SnacksNotifierInfo", "BlinkCmpKind", "NoiceCmdlinePopupBorder"},
		"material/darker":  {"GitSignsAdd", "WhichKeyGroup", "BlinkCmpMenu", "NoiceCmdlinePopupBorder"},
	} {
		values := maps.Clone(workspace)
		values["appearance.colorscheme"] = theme
		cases = append(cases, choice{"theme plugins " + theme, values, `for _, group in ipairs({ ` + luaList(groups) + ` }) do
  local hl = vim.api.nvim_get_hl(0, { name = group })
  assert(hl.link == nil and (hl.fg or hl.bg), group .. ' is not styled by the theme: ' .. vim.inspect(hl))
end`})
	}
	// sainnhe's themes draw popups a shade darker than the text (their light variants
	// already do).
	dimPopups := `local function lightness(c) return math.floor(c / 65536) + math.floor(c / 256) % 256 + c % 256 end
local text, popup = vim.api.nvim_get_hl(0, { name = 'Normal' }).bg, vim.api.nvim_get_hl(0, { name = 'NormalFloat', link = false }).bg
assert(text and popup and lightness(popup) < lightness(text), 'popups are not darker than the text')`
	cases = append(cases,
		choice{"theme kanagawa gutter", map[string]any{"appearance.colorscheme": "kanagawa/wave"}, "assert(vim.api.nvim_get_hl(0, { name = 'SignColumn' }).bg == nil, 'the sign column has its own background')"},
		choice{"theme everforest diagnostics", map[string]any{"appearance.colorscheme": "everforest/dark-hard"}, "assert(vim.g.everforest_diagnostic_virtual_text == 'colored', 'grey inline diagnostics')\n" + dimPopups},
		choice{"theme gruvbox-material popups", map[string]any{"appearance.colorscheme": "gruvbox-material/dark"}, dimPopups},
		choice{"theme catppuccin tabs", map[string]any{"appearance.colorscheme": "catppuccin/mocha", "appearance.buffers": "bufferline"}, "assert(vim.api.nvim_get_hl(0, { name = 'BufferLineBufferSelected' }).italic, 'bufferline lacks catppuccin colors')"},
		choice{"theme rose-pine tabs", map[string]any{"appearance.colorscheme": "rose-pine/main", "appearance.buffers": "bufferline"}, "assert(package.loaded['rose-pine.plugins.bufferline'], 'bufferline lacks rose-pine colors')"},
	)
	// Each interface choice also works with Nerd Font icons, which changes its setup.
	for _, c := range slices.Clone(cases) {
		for _, prefix := range []string{"statusline ", "tabs ", "finder ", "explorer ", "completion ", "start ", "hints ", "debug nvim-dap-view", "polish ", "noice "} {
			if strings.HasPrefix(c.name, prefix) {
				values := maps.Clone(c.values)
				values["appearance.icons"] = "nerd"
				check := c.check
				switch c.name {
				case "tabs bufferline", "tabs barbar":
					check += "\nassert(" + hasGlyph + "(vim.api.nvim_eval_statusline(vim.o.tabline, { use_tabline = true }).str), 'no file icons in the tab row')"
				case "debug nvim-dap-view":
					check += "\nassert(" + hasGlyph + "(vim.fn.sign_getdefined('DapBreakpoint')[1].text), 'breakpoint sign has no icon')"
				case "explorer nvim-tree", "explorer neo-tree", "explorer mini.files", "explorer oil":
					check += "\nvim.wait(500)\nlocal shown = ''\nfor _, w in ipairs(vim.api.nvim_list_wins()) do\n  local b = vim.api.nvim_win_get_buf(w)\n  shown = shown .. table.concat(vim.api.nvim_buf_get_lines(b, 0, -1, false), '\\n')\n  for _, m in ipairs(vim.api.nvim_buf_get_extmarks(b, -1, 0, -1, { details = true })) do\n    for _, chunk in ipairs(m[4].virt_text or {}) do shown = shown .. chunk[1] end\n  end\nend\nassert(" + hasGlyph + "(shown), 'no icons in the explorer: ' .. shown)"
				}
				cases = append(cases, choice{"nerd " + c.name, values, check})
			}
		}
	}
	// Every language: its server attaches in a small project, Tree-sitter parses the file,
	// and its formatter (when it has one) rewrites messy code as expected.
	type sample struct {
		files        map[string]string
		open, format string // format: the expected text after formatting; empty skips it
	}
	samples := map[string]sample{
		"typescript": {map[string]string{"package.json": "{}", "a.ts": "const x   =  1"}, "a.ts", "const x = 1;"},
		"html":       {map[string]string{"package.json": "{}", "index.html": "<div   class='a'>hi</div>"}, "index.html", `<div class="a">hi</div>`},
		"css":        {map[string]string{"package.json": "{}", "a.css": "a{color:red}"}, "a.css", "a {\n  color: red;\n}"},
		"json":       {map[string]string{"package.json": "{}", "a.json": `{"a":1}`}, "a.json", `{ "a": 1 }`},
		"svelte":     {map[string]string{"package.json": "{}", "App.svelte": "<script>let x = 1</script>\n<p>{x}</p>"}, "App.svelte", ""},
		"lua":        {map[string]string{".luarc.json": "{}", "a.lua": "local x  =  1"}, "a.lua", "local x = 1"},
		"python":     {map[string]string{"pyproject.toml": "", "a.py": "x  =  1"}, "a.py", "x = 1"},
		"ruby":       {map[string]string{".git/HEAD": "ref: refs/heads/main", "a.rb": "x = 1"}, "a.rb", ""}, // A Gemfile needs its lock file.
		"php":        {map[string]string{"composer.json": "{}", "a.php": "<?php echo 1;"}, "a.php", ""},
		"bash":       {map[string]string{"a.sh": "#!/bin/sh\necho   hi"}, "a.sh", "#!/bin/sh\necho hi"},
		"elixir":     {map[string]string{"mix.exs": "", "a.ex": "x = 1"}, "a.ex", ""},
		"go":         {map[string]string{"go.mod": "module demo\n\ngo 1.22", "main.go": "package main\nfunc main(){}"}, "main.go", "package main\n\nfunc main() {}"},
		"rust":       {map[string]string{"Cargo.toml": "[package]\nname = \"demo\"\nversion = \"0.1.0\"\nedition = \"2021\"", "src/main.rs": "fn main(){}"}, "src/main.rs", "fn main() {}"},
		"cpp":        {map[string]string{"compile_flags.txt": "", "a.cpp": "int main(){return 0;}"}, "a.cpp", "int main() { return 0; }"},
		"zig":        {map[string]string{"build.zig": "", "a.zig": "const x = 1;"}, "a.zig", ""},
		"swift":      {map[string]string{"Package.swift": "// swift-tools-version:5.9\nimport PackageDescription\nlet package = Package(name: \"demo\")", "main.swift": "let x = 1"}, "main.swift", ""},
		"java":       {map[string]string{"pom.xml": "<project/>", "A.java": "class A {}"}, "A.java", ""},
		"kotlin":     {map[string]string{"settings.gradle": "", "a.kt": "val x = 1"}, "a.kt", ""},
		"csharp":     {map[string]string{"demo.csproj": "<Project/>", "A.cs": "class A {}"}, "A.cs", ""},
		"yaml":       {map[string]string{"a.yaml": "a:   1"}, "a.yaml", "a: 1"},
		"toml":       {map[string]string{"taplo.toml": "", "a.toml": "a  =  1"}, "a.toml", "a = 1"},
		"markdown":   {map[string]string{".marksman.toml": "", "a.md": "#  Title"}, "a.md", "# Title"},
		"docker":     {map[string]string{"Dockerfile": "FROM alpine"}, "Dockerfile", ""},
		"terraform":  {map[string]string{"main.tf": "variable \"x\" {}"}, "main.tf", ""},
	}
	// openSample writes the sample's project to a new folder, opens its file as buf and
	// waits for the server to attach.
	openSample := func(server string, smp sample) string {
		var writes []string
		names := map[string]bool{}
		for name := range smp.files {
			names[name] = true
		}
		for _, name := range sortedKeys(names) {
			writes = append(writes, fmt.Sprintf("vim.fn.mkdir(vim.fs.dirname(dir .. '/' .. %[1]s), 'p'); vim.fn.writefile(vim.split(%[2]s, '\\n'), dir .. '/' .. %[1]s)", LuaString(name), LuaString(smp.files[name])))
		}
		return fmt.Sprintf(`local dir = vim.fn.resolve(vim.fn.tempname())
%s
vim.cmd.cd(dir)
vim.cmd.edit(dir .. '/' .. %s)
local buf = vim.api.nvim_get_current_buf()
if not vim.wait(60000, function() return #vim.lsp.get_clients({ bufnr = buf, name = %[3]s }) > 0 end, 200) then
  local log = table.concat(vim.fn.readfile(vim.lsp.log.get_filename()), '\n')
  error(%[3]s .. ' did not attach; LSP log: ' .. log:sub(-2500))
end
`, strings.Join(writes, "\n"), LuaString(smp.open), LuaString(server))
	}
	languageCheck := func(server string, conform bool, smp sample) string {
		formatter := "vim.lsp.buf.format({ bufnr = buf, async = false, timeout_ms = 20000 })"
		if conform {
			formatter = "require('conform').format({ bufnr = buf, async = false, timeout_ms = 20000 })"
		}
		return openSample(server, smp) + fmt.Sprintf(`pcall(function() vim.treesitter.get_parser(buf):parse(true) end)
assert(vim.treesitter.highlighter.active[buf], 'Tree-sitter did not start for ' .. vim.bo.filetype)
local want = %[1]s
if want ~= '' then
  vim.wait(3000, function() return false end)
  local before = table.concat(vim.api.nvim_buf_get_lines(buf, 0, -1, false), '\n')
  %[2]s
  local got = table.concat(vim.api.nvim_buf_get_lines(buf, 0, -1, false), '\n')
  if want == '*' then
    assert(got ~= before, 'the server did not format: ' .. vim.inspect(got))
  else
    assert(got == want, 'formatted to ' .. vim.inspect(got))
  end
end`, LuaString(smp.format), formatter)
	}
	// Formatting on save, the recommended choice: the first write of a session formats
	// the file on disk, though with conform that same write is what loads conform.
	saveCheck := func(server string, conform bool, smp sample) string {
		loaded := ""
		if conform {
			loaded = "assert(not package.loaded['conform'], 'conform loaded before the first write')\n"
		}
		return openSample(server, smp) + loaded + fmt.Sprintf(`vim.wait(3000, function() return false end)
local before = table.concat(vim.api.nvim_buf_get_lines(buf, 0, -1, false), '\n')
vim.cmd.write()
local got = table.concat(vim.api.nvim_buf_get_lines(buf, 0, -1, false), '\n')
local want = %s
if want == '*' then
  assert(got ~= before, 'saving did not format: ' .. vim.inspect(got))
else
  assert(got == want, 'saving formatted to ' .. vim.inspect(got))
end
assert(table.concat(vim.fn.readfile(vim.api.nvim_buf_get_name(buf)), '\n') == got, 'the file on disk is not the formatted text')`, LuaString(smp.format))
	}
	cases = append(cases,
		choice{"format on save conform", map[string]any{"language.lua.lsp": true, "tools.provider": "mason", "lsp.provider": "lspconfig",
			"formatting.provider": "conform", "formatting.on_save": true}, saveCheck("lua_ls", true, samples["lua"])},
		choice{"format on save server", map[string]any{"language.json.lsp": true, "tools.provider": "mason", "lsp.provider": "lspconfig",
			"formatting.provider": "native", "formatting.on_save": true}, saveCheck("jsonls", false, sample{samples["json"].files, samples["json"].open, "*"})},
	)
	for _, l := range Languages() {
		smp, ok := samples[l.ID]
		if !ok {
			t.Fatalf("no sample for %s", l.ID)
		}
		cases = append(cases, choice{"language " + l.ID, map[string]any{"language." + l.ID + ".lsp": true, "tools.provider": "mason", "lsp.provider": "lspconfig",
			"formatting.provider": "conform", "syntax.engine": "nvim-treesitter"}, languageCheck(l.Server.Name, true, smp)})
	}
	// Without nvim-lspconfig, ShoshinVim's basic recipes must still let servers format.
	for _, l := range Languages() {
		if l.ID == "json" || l.ID == "html" || l.ID == "css" {
			cases = append(cases, choice{"basic server formatting " + l.ID, map[string]any{"language." + l.ID + ".lsp": true, "tools.provider": "mason", "lsp.provider": "native",
				"formatting.provider": "native", "syntax.engine": "nvim-treesitter"}, languageCheck(l.Server.Name, false, sample{samples[l.ID].files, samples[l.ID].open, "*"})})
		}
	}
	// Every recipe resolves in Neovim, both ShoshinVim's basic ones and nvim-lspconfig's
	// with ShoshinVim's additions (Swift's server keeps to Swift files).
	for _, provider := range []string{"native", "lspconfig"} {
		values := map[string]any{"lsp.provider": provider}
		var checks []string
		for _, l := range Languages() {
			values["language."+l.ID+".lsp"] = true
			checks = append(checks, fmt.Sprintf("{ %s, %s }", LuaString(l.Server.Name), LuaString(l.Server.Command)))
		}
		cases = append(cases, choice{"all recipes " + provider, values, `for _, s in ipairs({ ` + strings.Join(checks, ", ") + ` }) do
  local config = vim.lsp.config[s[1]]
  assert(config and config.cmd and #config.filetypes > 0, s[1] .. ' has no recipe')
  if type(config.cmd) == 'table' then assert(config.cmd[1] == s[2], s[1] .. ' runs ' .. tostring(config.cmd[1])) end
end
assert(vim.deep_equal(vim.lsp.config.sourcekit.filetypes, { 'swift' }), 'sourcekit also serves ' .. table.concat(vim.lsp.config.sourcekit.filetypes, ','))
assert(vim.lsp.config.lua_ls.settings.Lua.workspace.library[1] == vim.env.VIMRUNTIME, 'lua_ls lost Neovim runtime')`})
	}
	cat := LoadCatalog()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := NewState()
			for id, v := range c.values {
				if err := cat.Set(&s, id, v, true); err != nil {
					t.Fatal(err)
				}
			}
			if lang, ok := strings.CutPrefix(c.name, "language "); ok {
				// CI installs every runtime and sets SHOSHINVIM_ALL_LANGUAGES=1, so a missing
				// one fails there instead of silently leaving a language untested.
				missing := t.Skipf
				if os.Getenv("SHOSHINVIM_ALL_LANGUAGES") == "1" {
					missing = t.Fatalf
				}
				for _, l := range Languages() {
					if l.ID != lang {
						continue
					}
					for _, r := range l.Server.Needs {
						if p, err := exec.LookPath(r); err != nil || !runtimeWorks(r, p) {
							missing("needs %s, which is not installed here", r)
						}
					}
					if l.Server.Package == "" {
						if _, err := exec.LookPath(l.Server.Command); err != nil {
							missing("%s is not installed here", l.Server.Command)
						}
					}
				}
			}
			if c.name == "shortcut command" {
				s.Keymaps = []Keymap{{Mode: "n", Key: "<leader>L", Action: CommandAction + "Lazy"}}
			}
			if len(Plugins(s)) > 0 {
				s.Values["plugins.manager"] = "lazy"
			}
			if err := cat.Validate(s); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(t.TempDir(), "nvim")
			files, err := Generate(s)
			if err != nil {
				t.Fatal(err)
			}
			if c.name == "plugin manager earlier files" {
				lua := files["lua/user_config/plugins.lua"]
				files["lua/user_config/plugins.lua"] = strings.Replace(lua, "'/plugins/lazy.nvim'", "'/plugins/lazy'", 1)
				if files["lua/user_config/plugins.lua"] == lua {
					t.Fatal("plugins.lua no longer names the lazy.nvim folder")
				}
			}
			if c.name == "broken plugin module" {
				lua := files["lua/user_config/plugins.lua"]
				files["lua/user_config/plugins.lua"] = strings.Replace(lua, "local specs = {}\n", "error('broken on purpose')\nlocal specs = {}\n", 1)
				if files["lua/user_config/plugins.lua"] == lua {
					t.Fatal("plugins.lua no longer starts its specs with local specs = {}")
				}
			}
			if c.name == "status line from user.lua" {
				files[userPath] = "vim.o.laststatus = 3\n"
			}
			for name, content := range files {
				p := filepath.Join(target, filepath.FromSlash(name))
				if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if strings.HasPrefix(c.name, "plugin manager") {
				t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
			}
			actions, err := Actions(s, target)
			if err != nil {
				t.Fatal(err)
			}
			base, _ := Canonical(DataDirFor(target))
			earlier := strings.Contains(c.name, "earlier")
			if earlier {
				a := actions[0]
				a.Destination = filepath.Join(base, "plugins", legacyLazyFolder)
				if _, err := acquire(context.Background(), base, nvim, a); err != nil {
					t.Fatalf("download lazy.nvim to its earlier folder: %v", err)
				}
			}
			for _, a := range actions {
				if a.Kind == "mason" && !strings.HasPrefix(c.name, "language ") && !strings.HasPrefix(c.name, "basic server") && !strings.HasPrefix(c.name, "format on save") && a.Package != "tree-sitter-cli" && a.Package != "markdownlint" && a.Package != "js-debug-adapter" {
					continue // Only the language cases and the tools a case exercises are installed.
				}
				if a.ID == "lazy" && c.name == "plugin manager in earlier folder" {
					continue // Saved without downloading: lazy.nvim stays where it was.
				}
				if !Installed(a) {
					note, err := acquire(context.Background(), base, nvim, a)
					if err != nil {
						t.Fatalf("download %s: %v", a.ID, err)
					}
					if a.ID == "lazy" && earlier && note == "" {
						t.Fatal("lazy.nvim was downloaded again instead of moved from its earlier folder")
					}
				}
			}
			tmp := t.TempDir()
			check := c.check
			if !strings.HasPrefix(c.name, "startup ") { // Checked before the first screen is drawn.
				check = firstScreenDrawn + "\n" + check
			}
			script := "local ok, err = pcall(function()\n" + check + "\nend)\n" + swatchLua + `
local has_plugins, plugins = pcall(require, 'user_config.plugins')
if ok and has_plugins and type(plugins) == 'table' and #plugins.errors > 0 then ok, err = false, table.concat(plugins.errors, '\n') end
local messages = vim.api.nvim_exec2('messages', { output = true }).output
if ok and (messages:find('Error') or messages:find('E%d+:')) then ok, err = false, messages end
io.stdout:write(ok and 'RESULT ok\n' or ('RESULT fail: ' .. tostring(err) .. '\n'))
-- Shut language servers down before quitting; some (jdtls) keep writing to their cache
-- after Neovim exits, which would leave the test's temporary folder in use.
for _, client in ipairs(vim.lsp.get_clients()) do client:stop() end
if not vim.wait(15000, function() return #vim.lsp.get_clients() == 0 end, 100) then
  for _, client in ipairs(vim.lsp.get_clients()) do client:stop(true) end
  vim.wait(5000, function() return #vim.lsp.get_clients() == 0 end, 100)
end
vim.cmd('qa!')`
			if err := os.WriteFile(filepath.Join(tmp, "check.lua"), []byte(script), 0600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, nvim, "--headless", "-i", "NONE", "-n", "-u", filepath.Join(target, "init.lua"),
				"+lua vim.defer_fn(function() dofile("+LuaString(filepath.Join(tmp, "check.lua"))+") end, 200)")
			cmd.Dir = tmp
			cmd.Env = append(os.Environ(), "XDG_STATE_HOME="+filepath.Join(tmp, "state"), "XDG_CACHE_HOME="+filepath.Join(tmp, "cache"), "NVIM_APPNAME=")
			// kotlin-language-server fails to start on the Java 25 that jdtls needs, so it
			// can be given its own JDK, as a user with both languages would through JAVA_HOME.
			if home := os.Getenv("SHOSHINVIM_KOTLIN_JAVA_HOME"); home != "" && c.name == "language kotlin" {
				cmd.Env = append(cmd.Env, "JAVA_HOME="+home, "PATH="+filepath.Join(home, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
			}
			out, _ := cmd.CombinedOutput()
			text := strings.ReplaceAll(string(out), "\r", "\n")
			if !strings.Contains(text, "RESULT ok") {
				t.Fatalf("%s", text)
			}
			if m := regexp.MustCompile(`SWATCH (\[.*\])`).FindStringSubmatch(text); m != nil {
				if id, ok := c.values["appearance.colorscheme"].(string); ok && id != CustomTheme {
					var colors []string
					if json.Unmarshal([]byte(m[1]), &colors) == nil {
						swatches[id] = colors
					}
				}
			}
		})
	}
	// A variant must actually change the colors, not just the name.
	for _, theme := range Themes() {
		if len(theme.Variants) < 2 || theme.Hidden {
			continue
		}
		seen := map[string]string{}
		for _, v := range theme.Variants {
			key := strings.Join(swatches[v.ID], ",")
			if other, dup := seen[key]; dup && key != "" {
				t.Errorf("%s and %s look identical; the variant is not applied", other, v.ID)
			}
			seen[key] = v.ID
		}
	}
	if os.Getenv("UPDATE_THEME_SWATCHES") == "1" && !t.Failed() {
		b, _ := json.MarshalIndent(swatches, "", "  ")
		if err := os.WriteFile(filepath.Join("data", "theme-swatches.json"), append(b, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

// openFile opens a new file, which loads the plugins that wait for the first file.
const openFile = "vim.cmd.edit(vim.fn.tempname() .. '.lua')\n"

// firstScreenDrawn does what lazy.nvim does once a UI has drawn Neovim's first screen: load the
// plugins waiting for VeryLazy. A headless Neovim has no UI, so it never happens there.
const firstScreenDrawn = `if package.loaded['lazy'] and not vim.g.did_very_lazy then
  vim.g.did_very_lazy = true
  vim.api.nvim_exec_autocmds('User', { pattern = 'VeryLazy', modeline = false })
end`

// lazyCheck: lazy.nvim counts itself installed and finds nothing to clean, so :Lazy lists
// no missing or unknown plugins.
const lazyCheck = `local config = require('lazy.core.config')
assert(config.plugins['lazy.nvim']._.installed, 'lazy.nvim does not count itself installed')
local unknown = {}
for _, p in ipairs(config.to_clean) do unknown[#unknown + 1] = p.name end
assert(#unknown == 0, 'lazy.nvim would clean ' .. table.concat(unknown, ', '))
` + openFile + `assert(package.loaded['gitsigns'], 'gitsigns not loaded')`

// tabsCheck: the tab row stays hidden until two files are open, so an empty start
// screen gets no blank bar.
const tabsCheck = `assert(vim.wait(1000, function() return vim.o.showtabline < 2 end), 'tab row shown without files')
vim.cmd('edit one.txt | edit two.txt')
assert(vim.wait(1000, function() return vim.o.showtabline == 2 end), 'tab row hidden with two files')`

// logoCheck: the open start screen paints the wordmark in its row colors.
const logoCheck = `local cells = 0
for _, m in ipairs(vim.api.nvim_buf_get_extmarks(0, -1, 0, -1, { details = true })) do
  if (m[4].hl_group or ''):match('^ShoshinLogo') then cells = cells + 1 end
end
assert(cells > 0, 'no wordmark on the start screen')
assert(vim.api.nvim_get_hl(0, { name = 'ShoshinLogo1' }).bg, 'wordmark has no color')`

// nerdIconsCheck: mini.icons answers for nvim-web-devicons, so the file tree, status
// line and start screen show Nerd Font icons.
const nerdIconsCheck = `local function has_glyph(text)
  for _, c in ipairs(vim.fn.str2list(text)) do
    if c >= 0xE000 and c <= 0xF8FF or c >= 0xF0000 then return true end
  end
end
local icon = require('nvim-web-devicons').get_icon('init.lua', 'lua')
assert(icon and has_glyph(icon), 'no file icon: ' .. tostring(icon))
assert(_G.MiniIcons, 'mini.icons not set up')
assert(require('lualine').get_config().options.icons_enabled, 'status line icons off')
Snacks.dashboard.open({ win = vim.api.nvim_get_current_win(), buf = vim.api.nvim_get_current_buf() })
local shown = table.concat(vim.api.nvim_buf_get_lines(0, 0, -1, false), '\n')
for _, m in ipairs(vim.api.nvim_buf_get_extmarks(0, -1, 0, -1, { details = true })) do
  for _, chunk in ipairs(m[4].virt_text or {}) do shown = shown .. chunk[1] end
end
assert(shown:find(require('user_config.icons').start.ShoshinFiles, 1, true), 'no icons on the start screen')
vim.fn.writefile({ 'return 1' }, 'init.lua')
vim.cmd('ShoshinExplore')
local tree
assert(vim.wait(4000, function()
  for _, w in ipairs(vim.api.nvim_list_wins()) do
    local buf = vim.api.nvim_win_get_buf(w)
    if vim.bo[buf].filetype == 'neo-tree' then
      tree = table.concat(vim.api.nvim_buf_get_lines(buf, 0, -1, false), '\n')
      return tree:find('init.lua', 1, true) ~= nil
    end
  end
end, 50), 'no file tree')
assert(has_glyph(tree), 'the file tree has no icons: ' .. tree)`

// plainIconsCheck: with plain symbols, no plugin keeps a Nerd Font glyph from its
// defaults in what it shows.
const plainIconsCheck = `local function glyphs(where, value, seen)
  seen = seen or {}
  if type(value) == 'string' then
    for _, c in ipairs(vim.fn.str2list(value)) do
      if c >= 0xE000 and c <= 0xF8FF or c >= 0xF0000 then error(where .. ' is a Nerd Font glyph: ' .. value) end
    end
  elseif type(value) == 'table' and not seen[value] then
    seen[value] = true
    for k, v in pairs(value) do glyphs(where .. '.' .. tostring(k), v, seen) end
  end
end
glyphs('snacks picker', require('snacks.picker.config').get({}).icons)
glyphs('lazy.nvim', require('lazy.core.config').options.ui.icons)
local wk = require('which-key.config').options.icons -- Its name rules apply only with icons on.
glyphs('which-key', { wk.keys, wk.breadcrumb, wk.separator, wk.group })
glyphs('neo-tree', require('neo-tree').ensure_config().default_component_configs)
glyphs('bufferline', require('bufferline.config').get().options)
assert(vim.wait(2000, require('noice.config').is_running), 'noice is not running')
local noice = require('noice.config').options
glyphs('noice', { noice.cmdline.format, noice.format.level.icons, noice.popupmenu.kind_icons })
local columns = require('blink.cmp.config').completion.menu.draw.columns
if type(columns) == 'function' then columns = columns() end -- blink picks columns by mode.
for _, column in ipairs(columns) do
  for _, c in ipairs(column) do assert(c ~= 'kind_icon', 'completion menu shows kind icons') end
end
vim.notify('plain', vim.log.levels.WARN)
assert(vim.wait(2000, function()
  for _, w in ipairs(vim.api.nvim_list_wins()) do
    if vim.bo[vim.api.nvim_win_get_buf(w)].filetype == 'snacks_notif' then
      glyphs('notification title', vim.api.nvim_win_get_config(w).title)
      return true
    end
  end
end, 50), 'no notification window')`

// hasGlyph is a Lua function expression: whether text holds a Nerd Font glyph.
const hasGlyph = `(function(text)
  for _, c in ipairs(vim.fn.str2list(text or '')) do
    if c >= 0xE000 and c <= 0xF8FF or c >= 0xF0000 then return true end
  end
  return false
end)`
