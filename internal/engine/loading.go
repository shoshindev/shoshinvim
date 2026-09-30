package engine

import (
	"fmt"
	"strings"
)

// Plugins load when first needed, as in LazyVim, so Neovim starts with only what its
// first screen shows. Each rule follows the plugin's own README or LazyVim's spec.
type loadRule struct {
	startup bool     // Loaded when Neovim starts.
	events  []string // Autocommand events that load it.
	cmds    []string // Commands that load it.
	// A plugin with none of these loads the first time Lua code requires it.
}

// fileEvents are LazyVim's LazyFile: a file is read, created or about to be written.
var fileEvents = []string{"BufReadPost", "BufNewFile", "BufWritePre"}

// completionEvents load a completion plugin when a file opens, so language servers
// started for it learn what completion supports, or on first typing.
var completionEvents = append(append([]string{}, fileEvents...), "InsertEnter", "CmdlineEnter")

var startup = loadRule{startup: true}
var veryLazy = loadRule{events: []string{"VeryLazy"}}
var onFile = loadRule{events: fileEvents}
var onRequire = loadRule{}

var loadRules = map[string]loadRule{
	// Their READMEs ask not to lazy-load them.
	"snacks": startup, "mason": startup, "nvim-treesitter": startup, "oil": startup,
	"neo-tree": startup, // "neo-tree will lazily load itself"
	// Server recipes that Neovim reads when a language server starts.
	"lspconfig": startup,
	// The chosen explorer opens folders given on the command line, in place of netrw.
	"nvim-tree": startup, "mini.files": startup,
	// The start screen is the first screen.
	"mini.starter": startup,
	// barbar hides its row by watching buffers open, so it must see the first one.
	"barbar": startup,

	// The status line, tab row, shortcut hints, command line, editing helpers and
	// mini.diff (as in LazyVim) load right after the first screen is drawn.
	"lualine": veryLazy, "mini.statusline": veryLazy, "bufferline": veryLazy, "mini.tabline": veryLazy,
	"which-key": veryLazy, "mini.clue": veryLazy, "noice": veryLazy,
	"mini.ai": veryLazy, "mini.surround": veryLazy, "nvim-surround": veryLazy, "mini.pairs": veryLazy, "mini.diff": veryLazy,

	// Helpers for the text of a file load with the first file.
	"gitsigns": onFile, "indent-blankline": onFile, "mini.indentscope": onFile, "mini.hipatterns": onFile,
	"todo-comments": onFile, "guess-indent": onFile,

	"blink.cmp":    {events: completionEvents},
	"cmp-nvim-lsp": {events: completionEvents},
	"nvim-cmp":     {events: []string{"InsertEnter"}}, "cmp-buffer": {events: []string{"InsertEnter"}}, "cmp-path": {events: []string{"InsertEnter"}},
	"nvim-autopairs": {events: []string{"InsertEnter"}},
	"fidget":         {events: []string{"LspAttach"}},
	"conform":        {events: []string{"BufWritePre"}, cmds: []string{"ConformInfo"}},

	// Finders load when one is opened, through a Shoshin command or their own.
	"telescope": {cmds: []string{"Telescope"}}, "fzf-lua": {cmds: []string{"FzfLua"}}, "mini.pick": {cmds: []string{"Pick"}},
	"nvim-dap": {cmds: []string{"DapSetLogLevel", "DapShowLog", "DapContinue", "DapToggleBreakpoint", "DapClearBreakpoints", "DapToggleRepl",
		"DapStepOver", "DapStepInto", "DapStepOut", "DapPause", "DapTerminate", "DapDisconnect", "DapRestartFrame", "DapNew", "DapEval"}},

	// Loaded by the code that uses them: the debugger views by nvim-dap's setup, nvim-lint
	// by the file types it lints, and libraries by their plugins.
	"nvim-dap-ui": onRequire, "nvim-dap-view": onRequire, "nvim-lint": onRequire,
	"mini.icons": onRequire, "plenary": onRequire, "nui": onRequire, "nvim-nio": onRequire,
}

// loadFields is a lazy.nvim spec's loading fields for a plugin of the state. Themes
// load at startup to color the first screen.
func loadFields(s State, id string) string {
	rule, ok := loadRules[id]
	if id == themePlugin(s) || !ok {
		rule = startup
	}
	var fields []string
	if rule.startup {
		fields = append(fields, "lazy = false")
	}
	if len(rule.events) > 0 {
		fields = append(fields, "event = { "+quoted(rule.events)+" }")
	}
	if len(rule.cmds) > 0 {
		fields = append(fields, "cmd = { "+quoted(rule.cmds)+" }")
	}
	if len(fields) == 0 {
		fields = append(fields, "lazy = true")
	}
	return strings.Join(fields, ", ")
}

// onLoad is Lua that runs body (written unindented) once lazy.nvim has loaded the
// plugin: at startup if it already has, otherwise when the plugin first loads.
// Validation probes skip it.
func onLoad(plugin, body string) string {
	lines := strings.Split(strings.TrimRight(body, "\n"), "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = "    " + line
		}
	}
	return fmt.Sprintf("if not vim.g.shoshinvim_probe then\n  require('user_config.plugins').on_load(%s, function()\n%s\n  end)\nend\n",
		LuaString(plugin), strings.Join(lines, "\n"))
}
