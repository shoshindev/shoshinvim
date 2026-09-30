package engine

// completionEngineLua configures a completion plugin; the native engine is generated
// separately and needs no plugin.
func completionEngineLua(s State) string {
	switch Value(s, "completion.engine", "native") {
	case "blink.cmp":
		// Pure-Lua matching needs no prebuilt binary. Labels are colored like code, as in
		// LazyVim. Without a Nerd Font, kinds are shown as text instead of blink's icon
		// column. blink's windows follow the popup border choice by themselves.
		menu := "\n      menu = { draw = { treesitter = { 'lsp' }, columns = { { 'label', 'label_description', gap = 1 }, { 'kind' } } } },"
		if nerdIcons(s) {
			menu = "\n      menu = { draw = { treesitter = { 'lsp' } } },"
		}
		return `if not vim.g.shoshinvim_probe then
  local ok, blink = pcall(require, 'blink.cmp')
  if ok then blink.setup({
    keymap = { preset = 'default' },
    appearance = { nerd_font_variant = 'mono' },
    completion = {
      documentation = { auto_show = true, auto_show_delay_ms = 300 },` + menu + `
    },
    sources = { default = { 'lsp', 'path', 'snippets', 'buffer' } },
    fuzzy = { implementation = 'lua' },
    signature = { enabled = true },
  }) end
end
`
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
			extra += "\n    window = { completion = cmp.config.window.bordered({ border = " + LuaString(b) + " }), documentation = cmp.config.window.bordered({ border = " + LuaString(b) + " }) },"
		}
		return `if not vim.g.shoshinvim_probe then
  local ok, cmp = pcall(require, 'cmp')
  if ok then cmp.setup({
    snippet = { expand = function(args) vim.snippet.expand(args.body) end },` + extra + `
    mapping = cmp.mapping.preset.insert({
      ['<C-Space>'] = cmp.mapping.complete(),
      ['<C-y>'] = cmp.mapping.confirm({ select = true }),
      ['<C-e>'] = cmp.mapping.abort(),
      ['<C-b>'] = cmp.mapping.scroll_docs(-4),
      ['<C-f>'] = cmp.mapping.scroll_docs(4),
    }),
    sources = cmp.config.sources({ { name = 'nvim_lsp' }, { name = 'path' } }, { { name = 'buffer' } }),
  }) end
end
`
	}
	return ""
}

// completionCapabilities tells language servers what the completion plugin supports.
func completionCapabilities(s State) string {
	if !anyLanguage(s) {
		return ""
	}
	switch Value(s, "completion.engine", "native") {
	case "blink.cmp":
		return "if not vim.g.shoshinvim_probe then\n  local ok, blink = pcall(require, 'blink.cmp')\n  if ok then vim.lsp.config('*', { capabilities = blink.get_lsp_capabilities() }) end\nend\n"
	case "nvim-cmp":
		return "if not vim.g.shoshinvim_probe then\n  local ok, source = pcall(require, 'cmp_nvim_lsp')\n  if ok then vim.lsp.config('*', { capabilities = source.default_capabilities() }) end\nend\n"
	}
	return ""
}
