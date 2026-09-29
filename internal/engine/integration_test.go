package engine

import (
	"context"
	"encoding/json"
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

	swatches := map[string][]string{}
	swatchLua := `local out = {}
for _, g in ipairs({ 'Normal', 'Keyword', 'Function', 'String', 'Type', 'Constant' }) do
  local hl = vim.api.nvim_get_hl(0, { name = g, link = false })
  local c = g == 'Normal' and (hl.bg or hl.fg) or hl.fg
  if c then out[#out + 1] = string.format('#%06x', c) end
end
io.stdout:write('SWATCH ' .. vim.json.encode(out) .. '\n')`
	cat := LoadCatalog()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := NewState()
			for id, v := range c.values {
				if err := cat.Set(&s, id, v, true); err != nil {
					t.Fatal(err)
				}
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
