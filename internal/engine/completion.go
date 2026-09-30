package engine

// completionEngineLua configures a completion plugin; the native engine is generated
// separately and needs no plugin.
func completionEngineLua(s State) string {
	switch Value(s, "completion.engine", "native") {
	case "blink.cmp":
		// Pure-Lua matching needs no prebuilt binary. Labels are colored like code, as in
		// LazyVim. Without a Nerd Font, kinds are shown as text instead of blink's icon
		// column. blink's windows follow the popup border choice by themselves.
		menu := "\n    menu = { draw = { treesitter = { 'lsp' }, columns = { { 'label', 'label_description', gap = 1 }, { 'kind' } } } },"
		if nerdIcons(s) {
			menu = "\n    menu = { draw = { treesitter = { 'lsp' } } },"
		}
		cmdline := "" // Under noice's command popup, : commands get LazyVim's dropdown.
		if Value(s, "appearance.cmdline", "native") == "noice" {
			cmdline = "\n  cmdline = { completion = { list = { selection = { preselect = false } }, menu = { auto_show = function() return vim.fn.getcmdtype() == ':' end }, ghost_text = { enabled = true } } },"
		}
		return onLoad("blink.cmp", `require('blink.cmp').setup({
  keymap = { preset = 'default' },
  appearance = { nerd_font_variant = 'mono' },
  completion = {
    documentation = { auto_show = true, auto_show_delay_ms = 300 },`+menu+`
  },
  sources = { default = { 'lsp', 'path', 'snippets', 'buffer' } },
  fuzzy = { implementation = 'lua' },
  signature = { enabled = true },`+cmdline+`
})`)
	case "nvim-cmp":
		extra := ""
		if nerdIcons(s) {
			extra = `
  formatting = { format = function(_, item)
    item.kind = (require('user_config.icons').kinds[item.kind] or '') .. ' ' .. item.kind
    return item
  end },`
		}
		if b := borderStyle(s); b != "" {
			extra += "\n  window = { completion = cmp.config.window.bordered({ border = " + LuaString(b) + " }), documentation = cmp.config.window.bordered({ border = " + LuaString(b) + " }) },"
		}
		return onLoad("nvim-cmp", `local cmp = require('cmp')
cmp.setup({
  snippet = { expand = function(args) vim.snippet.expand(args.body) end },`+extra+`
  mapping = cmp.mapping.preset.insert({
    ['<C-Space>'] = cmp.mapping.complete(),
    ['<C-y>'] = cmp.mapping.confirm({ select = true }),
    ['<C-e>'] = cmp.mapping.abort(),
    ['<C-b>'] = cmp.mapping.scroll_docs(-4),
    ['<C-f>'] = cmp.mapping.scroll_docs(4),
  }),
  sources = cmp.config.sources({ { name = 'nvim_lsp' }, { name = 'path' } }, { { name = 'buffer' } }),
})`)
	}
	return ""
}

// completionCapabilities tells language servers what the completion plugin supports.
// The plugin loads when a file opens, before a language server starts for it.
func completionCapabilities(s State) string {
	if !anyLanguage(s) {
		return ""
	}
	switch Value(s, "completion.engine", "native") {
	case "blink.cmp":
		return onLoad("blink.cmp", "vim.lsp.config('*', { capabilities = require('blink.cmp').get_lsp_capabilities() })")
	case "nvim-cmp":
		return onLoad("cmp-nvim-lsp", "vim.lsp.config('*', { capabilities = require('cmp_nvim_lsp').default_capabilities() })")
	}
	return ""
}
