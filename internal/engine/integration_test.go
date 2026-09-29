package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
		choice{"git gitsigns", map[string]any{"git.hunks": "gitsigns"}, "assert(package.loaded['gitsigns'], 'gitsigns not loaded')"},
		choice{"statusline lualine", map[string]any{"appearance.statusline": "lualine"}, "assert(package.loaded['lualine'] and vim.o.statusline:find('lualine'), vim.o.statusline)"},
		choice{"statusline mini", map[string]any{"appearance.statusline": "mini.statusline"}, "assert(_G.MiniStatusline and vim.o.statusline:find('MiniStatusline'), vim.o.statusline)"},
		choice{"tabs bufferline", map[string]any{"appearance.buffers": "bufferline"}, "assert(package.loaded['bufferline'] and vim.o.showtabline == 2)"},
		choice{"tabs barbar", map[string]any{"appearance.buffers": "barbar"}, "assert(package.loaded['barbar'] and vim.o.showtabline == 2, 'showtabline ' .. vim.o.showtabline)"},
		choice{"tabs mini", map[string]any{"appearance.buffers": "mini.tabline"}, "assert(_G.MiniTabline and vim.o.tabline:find('MiniTabline'), vim.o.tabline)"},
		choice{"completion blink", map[string]any{"completion.engine": "blink.cmp", "language.lua.lsp": true},
			"assert(package.loaded['blink.cmp']); local c = vim.lsp.config['*']; assert(c and c.capabilities and c.capabilities.textDocument.completion.completionItem.snippetSupport, 'capabilities')"},
		choice{"completion nvim-cmp", map[string]any{"completion.engine": "nvim-cmp", "language.lua.lsp": true},
			"local cmp = require('cmp'); assert(#cmp.get_config().sources >= 3, 'sources'); local c = vim.lsp.config['*']; assert(c and c.capabilities and c.capabilities.textDocument.completion.completionItem.snippetSupport, 'capabilities')"},
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
		choice{"shortcut command", map[string]any{"keymaps.style": "none"}, "assert(vim.fn.maparg('<leader>L', 'n'):find('Lazy'), 'custom command shortcut')"},
		choice{"start snacks", map[string]any{"workspace.start_screen": "snacks"}, "assert(require('snacks').config.dashboard.enabled, 'dashboard off')"},
		choice{"start mini.starter", map[string]any{"workspace.start_screen": "mini.starter"}, "assert(_G.MiniStarter, 'mini.starter not set up')"},
		choice{"indent blankline", map[string]any{"appearance.indent": "indent-blankline"}, "assert(package.loaded['ibl'], 'ibl not loaded')"},
		choice{"indent snacks", map[string]any{"appearance.indent": "snacks"}, "assert(require('snacks').config.indent.enabled, 'snacks indent off')"},
		choice{"indent mini", map[string]any{"appearance.indent": "mini.indentscope"}, "assert(_G.MiniIndentscope, 'mini.indentscope not set up')"},
		choice{"notify snacks", map[string]any{"appearance.notifications": "snacks"}, "assert(require('snacks').config.notifier.enabled, 'notifier off')"},
		choice{"notify fidget", map[string]any{"appearance.notifications": "fidget"}, "assert(package.loaded['fidget'], 'fidget not loaded')"},
		choice{"hints mini.clue", map[string]any{"navigation.shortcuts": "mini.clue", "keymaps.style": "kickstart"}, "assert(_G.MiniClue, 'mini.clue not set up')"},
		choice{"hints which-key", map[string]any{"navigation.shortcuts": "which-key", "keymaps.style": "lazyvim"}, "assert(package.loaded['which-key'], 'which-key not loaded')"},
		choice{"detect indent", map[string]any{"editor.detect_indent": "guess-indent"}, "assert(package.loaded['guess-indent'], 'guess-indent not loaded')"},
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
	)
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
	languageCheck := func(server string, conform bool, smp sample) string {
		var writes []string
		names := map[string]bool{}
		for name := range smp.files {
			names[name] = true
		}
		for _, name := range sortedKeys(names) {
			writes = append(writes, fmt.Sprintf("vim.fn.mkdir(vim.fs.dirname(dir .. '/' .. %[1]s), 'p'); vim.fn.writefile(vim.split(%[2]s, '\\n'), dir .. '/' .. %[1]s)", LuaString(name), LuaString(smp.files[name])))
		}
		formatter := "vim.lsp.buf.format({ bufnr = buf, async = false, timeout_ms = 20000 })"
		if conform {
			formatter = "require('conform').format({ bufnr = buf, async = false, timeout_ms = 20000 })"
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
pcall(function() vim.treesitter.get_parser(buf):parse(true) end)
assert(vim.treesitter.highlighter.active[buf], 'Tree-sitter did not start for ' .. vim.bo.filetype)
local want = %[4]s
if want ~= '' then
  vim.wait(3000, function() return false end)
  local before = table.concat(vim.api.nvim_buf_get_lines(buf, 0, -1, false), '\n')
  %[5]s
  local got = table.concat(vim.api.nvim_buf_get_lines(buf, 0, -1, false), '\n')
  if want == '*' then
    assert(got ~= before, 'the server did not format: ' .. vim.inspect(got))
  else
    assert(got == want, 'formatted to ' .. vim.inspect(got))
  end
end`, strings.Join(writes, "\n"), LuaString(smp.open), LuaString(server), LuaString(smp.format), formatter)
	}
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
			for name, content := range files {
				p := filepath.Join(target, filepath.FromSlash(name))
				if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			actions, err := Actions(s, target)
			if err != nil {
				t.Fatal(err)
			}
			base, _ := Canonical(DataDirFor(target))
			for _, a := range actions {
				if a.Kind == "mason" && !strings.HasPrefix(c.name, "language ") && !strings.HasPrefix(c.name, "basic server") && a.Package != "tree-sitter-cli" && a.Package != "markdownlint" && a.Package != "js-debug-adapter" {
					continue // Only the language cases and the tools a case exercises are installed.
				}
				if !Installed(a) {
					if err := acquire(context.Background(), base, nvim, a); err != nil {
						t.Fatalf("download %s: %v", a.ID, err)
					}
				}
			}
			tmp := t.TempDir()
			script := "local ok, err = pcall(function()\n" + c.check + "\nend)\n" + swatchLua + `
local messages = vim.api.nvim_exec2('messages', { output = true }).output
if ok and (messages:find('Error') or messages:find('E%d+:')) then ok, err = false, messages end
io.stdout:write(ok and 'RESULT ok\n' or ('RESULT fail: ' .. tostring(err) .. '\n'))
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
