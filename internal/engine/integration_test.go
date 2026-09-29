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
	cat := LoadCatalog()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := NewState()
			for id, v := range c.values {
				if err := cat.Set(&s, id, v, true); err != nil {
					t.Fatal(err)
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
