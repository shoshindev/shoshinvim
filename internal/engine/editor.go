package engine

import "strings"

// editorAutocmds adds the editing conveniences chosen in the Editor section.
func editorAutocmds(s State) string {
	var b strings.Builder
	group := "local group = vim.api.nvim_create_augroup('ShoshinEditor', { clear = true })\n"
	add := func(lua string) {
		b.WriteString(group)
		group = ""
		b.WriteString(lua)
	}
	if Enabled(s, "editor.highlight_yank") {
		add("vim.api.nvim_create_autocmd('TextYankPost', { group = group, desc = 'Highlight copied text', callback = function() vim.hl.on_yank() end })\n")
	}
	if Enabled(s, "editor.trim_whitespace") {
		// Markdown and diffs use trailing spaces on purpose.
		add(`vim.api.nvim_create_autocmd('BufWritePre', { group = group, desc = 'Remove trailing spaces', callback = function(args)
  if vim.tbl_contains({ 'markdown', 'diff' }, vim.bo[args.buf].filetype) or not vim.bo[args.buf].modifiable then return end
  local view = vim.fn.winsaveview()
  vim.cmd([[keeppatterns silent! %s/\s\+$//e]])
  vim.fn.winrestview(view)
end })
`)
	}
	if Enabled(s, "editor.restore_cursor") {
		add(`vim.api.nvim_create_autocmd('BufReadPost', { group = group, desc = 'Reopen at the last position', callback = function(args)
  if vim.tbl_contains({ 'gitcommit', 'gitrebase' }, vim.bo[args.buf].filetype) then return end
  local mark = vim.api.nvim_buf_get_mark(args.buf, '"')
  if mark[1] > 0 and mark[1] <= vim.api.nvim_buf_line_count(args.buf) then pcall(vim.api.nvim_win_set_cursor, 0, mark) end
end })
`)
	}
	if Enabled(s, "editor.spell_prose") {
		add("vim.api.nvim_create_autocmd('FileType', { group = group, desc = 'Spell check prose', pattern = { 'markdown', 'text', 'gitcommit' }, callback = function() vim.opt_local.spell = true end })\n")
	}
	return b.String()
}
