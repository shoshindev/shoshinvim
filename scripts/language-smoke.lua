-- Run against an isolated Rich IDE target with its approved tools installed.
-- SHOSHINVIM_TEST_ROOT is a disposable fixture directory. No acquisition occurs.
local ok, err = xpcall(function()
  local root = assert(vim.env.SHOSHINVIM_TEST_ROOT, 'set SHOSHINVIM_TEST_ROOT to a temporary directory')
  vim.fn.mkdir(root, 'p')
  vim.fn.writefile({ '{"name":"shoshin-language-smoke","private":true}' }, root .. '/package.json')
  vim.fn.writefile({ '{"compilerOptions":{"strict":true},"include":["*.ts"]}' }, root .. '/tsconfig.json')
  vim.cmd.cd(vim.fn.fnameescape(root))
  local fixtures = {
    { 'sample.ts', 'ts_ls', { 'const word: string = 123;', 'console.log(word);' } },
    { 'sample.html', 'html', { '<html><body><h1>Hello</h1></body></html>' } },
    { 'sample.css', 'cssls', { 'body { color: red; }' } },
    { 'sample.json', 'jsonls', { '{"ready":true}' } },
    { 'sample.lua', 'lua_ls', { 'local   value={one=1,two=2}', 'print(value.one)' } },
  }
  for _, fixture in ipairs(fixtures) do
    local path, name, lines = root .. '/' .. fixture[1], fixture[2], fixture[3]
    vim.fn.writefile(lines, path)
    vim.cmd.edit(vim.fn.fnameescape(path))
    local buf = vim.api.nvim_get_current_buf()
    local client
    assert(vim.wait(30000, function()
      for _, c in ipairs(vim.lsp.get_clients({ bufnr = buf })) do if c.name == name and c.initialized then client = c; return true end end
      return false
    end, 50), 'server did not attach: ' .. name)
    assert(client:supports_method('textDocument/completion'), name .. ' lacks completion')
    local params = { textDocument = { uri = vim.uri_from_bufnr(buf) }, position = { line = 0, character = name == 'ts_ls' and 7 or 1 } }
    local response = client:request_sync('textDocument/completion', params, 15000, buf)
    assert(response and not response.err, name .. ' completion request failed')
    if name == 'ts_ls' then
      assert(vim.wait(20000, function() return #vim.diagnostic.get(buf) > 0 end, 50), 'TypeScript diagnostics did not arrive')
      local hover = client:request_sync('textDocument/hover', params, 10000, buf)
      assert(hover and hover.result and hover.result.contents, 'TypeScript hover failed')
    end
    if name == 'ts_ls' or name == 'lua_ls' then
      local before = table.concat(vim.api.nvim_buf_get_lines(buf, 0, -1, false), '\n')
      local done, failure = false, nil
      require('conform').format({ bufnr = buf, async = true, timeout_ms = 15000, lsp_format = 'never' }, function(e) failure = e; done = true end)
      assert(vim.wait(20000, function() return done end, 50), 'formatter timed out')
      assert(not failure, tostring(failure))
      local after = table.concat(vim.api.nvim_buf_get_lines(buf, 0, -1, false), '\n')
      if name == 'lua_ls' then assert(before ~= after, 'StyLua made no expected whitespace change') end
      vim.cmd.write()
    end
    print('PASS: ' .. name .. ' attachment and completion' .. ((name == 'ts_ls' or name == 'lua_ls') and ', formatter' or ''))
  end
  assert(vim.v.errmsg == '', vim.v.errmsg)
end, debug.traceback)
for _, client in ipairs(vim.lsp.get_clients()) do client:stop(true) end
if not ok then io.stderr:write(tostring(err)); vim.cmd('cquit 1') end
vim.cmd('qa!')
