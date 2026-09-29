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
	return fmt.Sprintf(`local ok, lint = false, nil
if not vim.g.shoshinvim_probe then ok, lint = pcall(require, 'lint') end
if not ok then return end
local by_filetype = { %s }
local needs_project = { eslint_d = { %s } }
local function run(buf)
  local names = {}
  for _, name in ipairs(by_filetype[vim.bo[buf].filetype] or {}) do
    local markers = needs_project[name]
    if vim.fn.executable(name) == 1 and (not markers or vim.fs.root(buf, markers)) then names[#names + 1] = name end
  end
  if #names > 0 then lint.try_lint(names) end
end
vim.api.nvim_create_autocmd({ 'BufReadPost', 'BufWritePost', 'InsertLeave' }, {
  group = vim.api.nvim_create_augroup('ShoshinLint', { clear = true }),
  callback = function(event) run(event.buf) end,
})
vim.api.nvim_create_user_command('ShoshinLint', function() run(vim.api.nvim_get_current_buf()) end, { desc = 'Lint this file' })
`, strings.Join(byFiletype, ", "), quoted(eslintConfigs))
}

// debugLua sets up nvim-dap for JavaScript and TypeScript with js-debug-adapter (the
// VS Code debugger), and the chosen debugger view with icons that need no Nerd Font.
func debugLua(s State) string {
	if !debugging(s) {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, `local ok, dap = false, nil
if not vim.g.shoshinvim_probe then ok, dap = pcall(require, 'dap') end
if not ok then return end
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
	switch Value(s, "coding.debug", "none") {
	case "nvim-dap-ui":
		// kickstart's debug.lua settings: icons that work in every terminal.
		b.WriteString(`local ui_ok, dapui = pcall(require, 'dapui')
if ui_ok then
  dapui.setup({
    icons = { expanded = '▾', collapsed = '▸', current_frame = '*' },
    controls = { icons = { pause = '⏸', play = '▶', step_into = '⏎', step_over = '⏭', step_out = '⏮', step_back = 'b', run_last = '▶▶', terminate = '⏹', disconnect = '⏏' } },
  })
  dap.listeners.after.event_initialized.shoshinvim = function() dapui.open() end
  dap.listeners.before.event_terminated.shoshinvim = function() dapui.close() end
  dap.listeners.before.event_exited.shoshinvim = function() dapui.close() end
  vim.api.nvim_create_user_command('ShoshinDebugView', function() dapui.toggle() end, { desc = 'Toggle the debugger view' })
end
`)
	case "nvim-dap-view":
		b.WriteString(`local view_ok, view = pcall(require, 'dap-view')
if view_ok then
  view.setup({
    auto_toggle = true,
    icons = { collapsed = '▸ ', expanded = '▾ ', disabled = '○', enabled = '●', disconnect = '⏏', filter = 'f', negate = '! ',
      pause = '⏸', play = '▶', run_last = '▶▶', step_back = 'b', step_into = '⏎', step_out = '⏮', step_over = '⏭', terminate = '⏹' },
  })
  vim.api.nvim_create_user_command('ShoshinDebugView', function() view.toggle() end, { desc = 'Toggle the debugger view' })
end
`)
	}
	return b.String()
}
