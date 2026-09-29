package engine

// completionEngineLua configures a completion plugin; the native engine is generated
// separately and needs no plugin.
func completionEngineLua(s State) string {
	switch Value(s, "completion.engine", "native") {
	case "blink.cmp":
		// Pure-Lua matching needs no prebuilt binary; kinds are shown as text, not icons.
		return `if not vim.g.shoshinvim_probe then
  local ok, blink = pcall(require, 'blink.cmp')
  if ok then blink.setup({
    keymap = { preset = 'default' },
    appearance = { nerd_font_variant = 'mono' },
    completion = {
      documentation = { auto_show = true, auto_show_delay_ms = 300 },
      menu = { draw = { columns = { { 'label', 'label_description', gap = 1 }, { 'kind' } } } },
    },
    sources = { default = { 'lsp', 'path', 'snippets', 'buffer' } },
    fuzzy = { implementation = 'lua' },
    signature = { enabled = true },
  }) end
end
`
	case "nvim-cmp":
		return `if not vim.g.shoshinvim_probe then
  local ok, cmp = pcall(require, 'cmp')
  if ok then cmp.setup({
    snippet = { expand = function(args) vim.snippet.expand(args.body) end },
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

func anyLanguage(s State) bool {
	for _, l := range languages {
		if Enabled(s, "language."+l.ID+".lsp") {
			return true
		}
	}
	return false
}
