package engine

import (
	"fmt"
	"strings"
)

// eslintConfigs are the files that mark a project as using ESLint.
var eslintConfigs = []string{"eslint.config.js", "eslint.config.mjs", "eslint.config.cjs", "eslint.config.ts", "eslint.config.mts", "eslint.config.cts",
	".eslintrc", ".eslintrc.js", ".eslintrc.cjs", ".eslintrc.json", ".eslintrc.yml", ".eslintrc.yaml"}

var scriptFiletypes = []string{"javascript", "javascriptreact", "typescript", "typescriptreact"}

func linting(s State) bool { return Value(s, "coding.lint", "none") == "nvim-lint" }

func debugging(s State) bool { return Value(s, "coding.debug", "none") != "none" }

// linters names the programs nvim-lint runs: markdownlint for Markdown (kickstart's
// example) and, with TypeScript chosen, ESLint through eslint_d.
func linters(s State) []string {
	if !linting(s) {
		return nil
	}
	out := []string{"markdownlint"}
	if Enabled(s, "language.typescript.lsp") {
		out = append(out, "eslint_d")
	}
	return out
}

func quoted(items []string) string {
	var out []string
	for _, item := range items {
		out = append(out, LuaString(item))
	}
	return strings.Join(out, ", ")
}

// lintLua runs each linter only where it applies, and only when it is installed, so a
// project without an ESLint configuration never shows ESLint errors.
func lintLua(s State) string {
	if !linting(s) {
		return ""
	}
	var byFiletype []string
	byFiletype = append(byFiletype, "markdown = { 'markdownlint' }")
	if Enabled(s, "language.typescript.lsp") {
		for _, ft := range scriptFiletypes {
			byFiletype = append(byFiletype, ft+" = { 'eslint_d' }")
		}
	}
	return fmt.Sprintf(`if vim.g.shoshinvim_probe then return end
local by_filetype = { %s }
local needs_project = { eslint_d = { %s } }
-- nvim-lint loads the first time a file it lints is opened.
local function run(buf)
  local names = {}
  for _, name in ipairs(by_filetype[vim.bo[buf].filetype] or {}) do
    local markers = needs_project[name]
    if vim.fn.executable(name) == 1 and (not markers or vim.fs.root(buf, markers)) then names[#names + 1] = name end
  end
  if #names == 0 then return end
  local ok, lint = pcall(require, 'lint')
  if ok then lint.try_lint(names) end
end
vim.api.nvim_create_autocmd({ 'BufReadPost', 'BufWritePost', 'InsertLeave' }, {
  group = vim.api.nvim_create_augroup('ShoshinLint', { clear = true }),
  callback = function(event) run(event.buf) end,
})
vim.api.nvim_create_user_command('ShoshinLint', function() run(vim.api.nvim_get_current_buf()) end, { desc = 'Lint this file' })
`, strings.Join(byFiletype, ", "), quoted(eslintConfigs))
}

// debugLua sets up nvim-dap for JavaScript and TypeScript with js-debug-adapter (the
// VS Code debugger), and the chosen debugger view.
func debugLua(s State) string {
	if !debugging(s) {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, `local dap = require('dap')
-- js-debug-adapter is the VS Code JavaScript debugger, run as a server.
dap.adapters['pwa-node'] = {
  type = 'server', host = 'localhost', port = '${port}',
  executable = { command = 'js-debug-adapter', args = { '${port}' } },
}
for _, ft in ipairs({ %s }) do
  dap.configurations[ft] = {
    { type = 'pwa-node', request = 'launch', name = 'Launch this file with Node.js', program = '${file}', cwd = '${workspaceFolder}' },
    { type = 'pwa-node', request = 'attach', name = 'Attach to a running Node.js process', processId = require('dap.utils').pick_process, cwd = '${workspaceFolder}' },
  }
end
`, quoted(scriptFiletypes))
	nerd := nerdIcons(s)
	if nerd {
		// LazyVim's breakpoint signs.
		b.WriteString(`local sign_hl = { Stopped = 'DiagnosticWarn', BreakpointRejected = 'DiagnosticError' }
vim.api.nvim_set_hl(0, 'DapStoppedLine', { default = true, link = 'Visual' })
for name, text in pairs(require('user_config.icons').dap) do
  local line = name == 'Stopped' and 'DapStoppedLine' or nil
  vim.fn.sign_define('Dap' .. name, { text = text, texthl = sign_hl[name] or 'DiagnosticInfo', linehl = line, numhl = line })
end
`)
	}
	toggle := ""
	switch Value(s, "coding.debug", "none") {
	case "nvim-dap-ui":
		// kickstart's debug.lua settings: icons that work in every terminal, or the
		// plugin's own with a Nerd Font.
		var opts []string
		if !nerd {
			opts = append(opts, "icons = { expanded = '▾', collapsed = '▸', current_frame = '*' }",
				"controls = { icons = { pause = '⏸', play = '▶', step_into = '⏎', step_over = '⏭', step_out = '⏮', step_back = 'b', run_last = '▶▶', terminate = '⏹', disconnect = '⏏' } }")
		}
		if border := borderStyle(s); border != "" {
			opts = append(opts, "floating = { border = "+LuaString(border)+" }")
		}
		setup := "{}"
		if len(opts) > 0 {
			setup = "{\n  " + strings.Join(opts, ",\n  ") + ",\n}"
		}
		b.WriteString(`local dapui = require('dapui')
dapui.setup(` + setup + `)
dap.listeners.after.event_initialized.shoshinvim = function() dapui.open() end
dap.listeners.before.event_terminated.shoshinvim = function() dapui.close() end
dap.listeners.before.event_exited.shoshinvim = function() dapui.close() end
`)
		toggle = "require('dapui').toggle()"
	case "nvim-dap-view":
		icons := `
  icons = { collapsed = '▸ ', expanded = '▾ ', disabled = '○', enabled = '●', disconnect = '⏏', filter = 'f', negate = '! ',
    pause = '⏸', play = '▶', run_last = '▶▶', step_back = 'b', step_into = '⏎', step_out = '⏮', step_over = '⏭', terminate = '⏹' },`
		if nerd {
			icons = ""
		}
		b.WriteString(`require('dap-view').setup({
  auto_toggle = true,` + icons + `
})
`)
		toggle = "require('dap-view').toggle()"
	}
	// The debugger loads with its first shortcut or command; its view is set up with it.
	return onLoad("nvim-dap", b.String()) + `vim.api.nvim_create_user_command('ShoshinDebugView', function()
  if not pcall(require, 'dap') then -- Loads the debugger, and so sets its view up, first.
    return vim.notify('nvim-dap is not installed yet. Open ShoshinVim and download plugins.', vim.log.levels.WARN)
  end
  ` + toggle + `
end, { desc = 'Toggle the debugger view' })
`
}
