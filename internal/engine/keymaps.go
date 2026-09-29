package engine

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// KeyAction is something a shortcut can do. RHS is either a key sequence/command
// (quoted when generated) or, when Lua is set, a Lua expression such as a function.
type KeyAction struct {
	ID, Label, Group, RHS string
	Lua                   bool
	Needs                 string // Setting that must not be native/none for the action to exist.
}

// KeyActions is the catalog of shortcut actions, in the order shown to users.
var KeyActions = []KeyAction{
	{ID: "find.files", Label: "Find files", Group: "Find", RHS: "<cmd>ShoshinFiles<CR>"},
	{ID: "find.grep", Label: "Search text in the project", Group: "Find", RHS: "<cmd>ShoshinGrep<CR>"},
	{ID: "find.word", Label: "Search the word under the cursor", Group: "Find", RHS: "<cmd>ShoshinGrepWord<CR>"},
	{ID: "find.recent", Label: "Recent files", Group: "Find", RHS: "<cmd>ShoshinRecent<CR>"},
	{ID: "find.buffers", Label: "Open buffers", Group: "Find", RHS: "<cmd>ShoshinBuffers<CR>"},
	{ID: "find.buffer_lines", Label: "Search in this file", Group: "Find", RHS: "<cmd>ShoshinBufferLines<CR>"},
	{ID: "find.help", Label: "Search Neovim help", Group: "Find", RHS: "<cmd>ShoshinHelpPages<CR>"},
	{ID: "find.keymaps", Label: "Search keymaps", Group: "Find", RHS: "<cmd>ShoshinKeymaps<CR>"},
	{ID: "find.commands", Label: "Search commands", Group: "Find", RHS: "<cmd>ShoshinCommands<CR>"},
	{ID: "find.diagnostics", Label: "Search diagnostics", Group: "Find", RHS: "<cmd>ShoshinDiagnostics<CR>"},
	{ID: "find.resume", Label: "Reopen the last search", Group: "Find", RHS: "<cmd>ShoshinResume<CR>"},
	{ID: "find.config", Label: "Find Neovim config files", Group: "Find", RHS: "<cmd>ShoshinConfigFiles<CR>"},
	{ID: "explorer.toggle", Label: "File explorer", Group: "Files", RHS: "<cmd>ShoshinExplore<CR>"},
	{ID: "file.save", Label: "Save file", Group: "Files", RHS: "<cmd>write<CR><Esc>"},
	{ID: "file.new", Label: "New file", Group: "Files", RHS: "<cmd>enew<CR>"},
	{ID: "code.format", Label: "Format file", Group: "Code", RHS: "<cmd>ShoshinFormat<CR>"},
	{ID: "code.action", Label: "Code action", Group: "Code", RHS: "vim.lsp.buf.code_action", Lua: true},
	{ID: "code.rename", Label: "Rename symbol", Group: "Code", RHS: "vim.lsp.buf.rename", Lua: true},
	{ID: "code.definition", Label: "Go to definition", Group: "Code", RHS: "vim.lsp.buf.definition", Lua: true},
	{ID: "code.references", Label: "Find references", Group: "Code", RHS: "vim.lsp.buf.references", Lua: true},
	{ID: "code.implementation", Label: "Go to implementation", Group: "Code", RHS: "vim.lsp.buf.implementation", Lua: true},
	{ID: "code.type_definition", Label: "Go to type definition", Group: "Code", RHS: "vim.lsp.buf.type_definition", Lua: true},
	{ID: "code.declaration", Label: "Go to declaration", Group: "Code", RHS: "vim.lsp.buf.declaration", Lua: true},
	{ID: "code.hover", Label: "Documentation", Group: "Code", RHS: "vim.lsp.buf.hover", Lua: true},
	{ID: "code.signature", Label: "Signature help", Group: "Code", RHS: "vim.lsp.buf.signature_help", Lua: true},
	{ID: "code.line_diagnostics", Label: "Line diagnostics", Group: "Code", RHS: "vim.diagnostic.open_float", Lua: true},
	{ID: "code.diagnostics_list", Label: "Diagnostics list", Group: "Code", RHS: "vim.diagnostic.setloclist", Lua: true},
	{ID: "code.next_diagnostic", Label: "Next diagnostic", Group: "Code", RHS: "function() vim.diagnostic.jump({ count = 1, float = true }) end", Lua: true},
	{ID: "code.prev_diagnostic", Label: "Previous diagnostic", Group: "Code", RHS: "function() vim.diagnostic.jump({ count = -1, float = true }) end", Lua: true},
	{ID: "code.inlay_hints", Label: "Toggle inlay hints", Group: "Code", RHS: "function() vim.lsp.inlay_hint.enable(not vim.lsp.inlay_hint.is_enabled()) end", Lua: true},
	{ID: "git.next_hunk", Label: "Next Git change", Group: "Git", RHS: "<cmd>ShoshinHunkNext<CR>", Needs: "git.hunks"},
	{ID: "git.prev_hunk", Label: "Previous Git change", Group: "Git", RHS: "<cmd>ShoshinHunkPrev<CR>", Needs: "git.hunks"},
	{ID: "git.preview_hunk", Label: "Preview Git change", Group: "Git", RHS: "<cmd>ShoshinHunkPreview<CR>", Needs: "git.hunks"},
	{ID: "git.stage_hunk", Label: "Stage Git change", Group: "Git", RHS: "<cmd>ShoshinHunkStage<CR>", Needs: "git.hunks"},
	{ID: "git.reset_hunk", Label: "Undo Git change", Group: "Git", RHS: "<cmd>ShoshinHunkReset<CR>", Needs: "git.hunks"},
	{ID: "git.blame_line", Label: "Blame this line", Group: "Git", RHS: "<cmd>ShoshinBlameLine<CR>", Needs: "git.hunks"},
	{ID: "git.diff", Label: "Diff this file", Group: "Git", RHS: "<cmd>ShoshinGitDiff<CR>", Needs: "git.hunks"},
	{ID: "git.status", Label: "Changed files", Group: "Git", RHS: "<cmd>ShoshinGitStatus<CR>"},
	{ID: "buffer.next", Label: "Next buffer", Group: "Buffers", RHS: "<cmd>bnext<CR>"},
	{ID: "buffer.prev", Label: "Previous buffer", Group: "Buffers", RHS: "<cmd>bprevious<CR>"},
	{ID: "buffer.other", Label: "Switch to the other buffer", Group: "Buffers", RHS: "<cmd>e #<CR>"},
	{ID: "buffer.delete", Label: "Close buffer", Group: "Buffers", RHS: "<cmd>ShoshinBufferDelete<CR>"},
	{ID: "window.left", Label: "Focus window left", Group: "Windows", RHS: "<C-w>h"},
	{ID: "window.down", Label: "Focus window below", Group: "Windows", RHS: "<C-w>j"},
	{ID: "window.up", Label: "Focus window above", Group: "Windows", RHS: "<C-w>k"},
	{ID: "window.right", Label: "Focus window right", Group: "Windows", RHS: "<C-w>l"},
	{ID: "window.split_below", Label: "Split window below", Group: "Windows", RHS: "<C-w>s"},
	{ID: "window.split_right", Label: "Split window right", Group: "Windows", RHS: "<C-w>v"},
	{ID: "window.close", Label: "Close window", Group: "Windows", RHS: "<C-w>c"},
	{ID: "search.clear", Label: "Clear search highlight", Group: "Editor", RHS: "<cmd>nohlsearch<CR>"},
	{ID: "view.wrap", Label: "Toggle line wrapping", Group: "Editor", RHS: "<cmd>set wrap!<CR>"},
	{ID: "terminal.toggle", Label: "Terminal", Group: "Editor", RHS: "<cmd>ShoshinTerminal<CR>"},
	{ID: "terminal.normal", Label: "Leave terminal mode", Group: "Editor", RHS: "<C-\\><C-n>"},
	{ID: "quit.all", Label: "Quit all", Group: "Editor", RHS: "<cmd>qa<CR>"},
	{ID: "guide", Label: "Shortcut guide", Group: "Editor", RHS: "<cmd>ShoshinHelp<CR>"},
}

// legacyKeyActions maps action IDs from earlier states to the current catalog.
var legacyKeyActions = map[string]string{"clear_search": "search.clear", "save": "file.save", "files": "find.files",
	"buffers": "find.buffers", "diagnostics": "code.diagnostics_list", "format": "code.format", "terminal": "terminal.toggle"}

// CommandAction prefixes a user shortcut that runs an Ex command.
const CommandAction = "command:"

// NoAction in a user shortcut turns off a key from the chosen style.
const NoAction = "none"

func keyAction(id string) (KeyAction, bool) {
	if legacy, ok := legacyKeyActions[id]; ok {
		id = legacy
	}
	for _, a := range KeyActions {
		if a.ID == id {
			return a, true
		}
	}
	if cmd, ok := strings.CutPrefix(id, CommandAction); ok {
		return KeyAction{ID: id, Label: ":" + cmd, Group: "Yours", RHS: "<cmd>" + cmd + "<CR>"}, true
	}
	return KeyAction{}, false
}

// KeyActionFor exposes a catalog entry (including custom commands) to the interface.
func KeyActionFor(id string) (KeyAction, bool) { return keyAction(id) }

type presetKey struct{ Mode, Key, Action string }

// KeymapStyle is a community keymap set, reproduced from its source.
type KeymapStyle struct {
	ID, Label, Source, URL string
	Groups                 [][2]string // which-key group labels: prefix, name
	keys                   []presetKey
}

// Styles are copied from the named projects' own configuration (read 2026-09-29).
// Only keys whose actions ShoshinVim supports are included; Neovim's built-in LSP keys
// (grn, gra, grr, gri, gO, K, [d, ]d) stay available in every style.
var KeymapStyles = []KeymapStyle{
	{ID: "kickstart", Label: "kickstart.nvim", Source: "kickstart.nvim", URL: "https://github.com/nvim-lua/kickstart.nvim/blob/master/init.lua",
		Groups: [][2]string{{"<leader>s", "Search"}, {"<leader>t", "Toggle"}, {"<leader>h", "Git hunk"}},
		keys: []presetKey{
			{"n", "<Esc>", "search.clear"}, {"n", "<leader>q", "code.diagnostics_list"}, {"t", "<Esc><Esc>", "terminal.normal"},
			{"n", "<C-h>", "window.left"}, {"n", "<C-l>", "window.right"}, {"n", "<C-j>", "window.down"}, {"n", "<C-k>", "window.up"},
			{"n", "]c", "git.next_hunk"}, {"n", "[c", "git.prev_hunk"}, {"n", "<leader>hs", "git.stage_hunk"}, {"n", "<leader>hr", "git.reset_hunk"},
			{"n", "<leader>hp", "git.preview_hunk"}, {"n", "<leader>hb", "git.blame_line"}, {"n", "<leader>hd", "git.diff"},
			{"n", "<leader>sh", "find.help"}, {"n", "<leader>sk", "find.keymaps"}, {"n", "<leader>sf", "find.files"}, {"n", "<leader>sw", "find.word"},
			{"n", "<leader>sg", "find.grep"}, {"n", "<leader>sd", "find.diagnostics"}, {"n", "<leader>sr", "find.resume"}, {"n", "<leader>s.", "find.recent"},
			{"n", "<leader>sc", "find.commands"}, {"n", "<leader><leader>", "find.buffers"}, {"n", "<leader>/", "find.buffer_lines"}, {"n", "<leader>sn", "find.config"},
			{"n", "grd", "code.definition"}, {"n", "grD", "code.declaration"}, {"n", "grt", "code.type_definition"}, {"n", "<leader>th", "code.inlay_hints"},
			{"nv", "<leader>f", "code.format"}, {"n", "\\", "explorer.toggle"},
		}},
	{ID: "lazyvim", Label: "LazyVim", Source: "LazyVim", URL: "https://www.lazyvim.org/keymaps",
		Groups: [][2]string{{"<leader>b", "Buffer"}, {"<leader>c", "Code"}, {"<leader>f", "File/find"}, {"<leader>g", "Git"}, {"<leader>gh", "Hunks"}, {"<leader>q", "Quit"}, {"<leader>s", "Search"}, {"<leader>u", "UI"}, {"<leader>w", "Windows"}},
		keys: []presetKey{
			{"n", "<C-h>", "window.left"}, {"n", "<C-j>", "window.down"}, {"n", "<C-k>", "window.up"}, {"n", "<C-l>", "window.right"},
			{"n", "<S-h>", "buffer.prev"}, {"n", "<S-l>", "buffer.next"}, {"n", "[b", "buffer.prev"}, {"n", "]b", "buffer.next"},
			{"n", "<leader>bb", "buffer.other"}, {"n", "<leader>bd", "buffer.delete"}, {"n", "<Esc>", "search.clear"}, {"nixs", "<C-s>", "file.save"},
			{"n", "<leader>fn", "file.new"}, {"n", "<leader>cd", "code.line_diagnostics"}, {"n", "]d", "code.next_diagnostic"}, {"n", "[d", "code.prev_diagnostic"},
			{"n", "<leader>qq", "quit.all"}, {"n", "<leader>-", "window.split_below"}, {"n", "<leader>|", "window.split_right"}, {"n", "<leader>wd", "window.close"},
			{"n", "<leader>uw", "view.wrap"}, {"n", "<leader>uh", "code.inlay_hints"}, {"n", "<leader>ft", "terminal.toggle"}, {"nt", "<C-/>", "terminal.toggle"},
			{"n", "<leader>,", "find.buffers"}, {"n", "<leader>/", "find.grep"}, {"n", "<leader><space>", "find.files"}, {"n", "<leader>fb", "find.buffers"},
			{"n", "<leader>fc", "find.config"}, {"n", "<leader>ff", "find.files"}, {"n", "<leader>fr", "find.recent"}, {"n", "<leader>gs", "git.status"},
			{"n", "<leader>gd", "git.diff"}, {"n", "<leader>sb", "find.buffer_lines"}, {"n", "<leader>sg", "find.grep"}, {"n", "<leader>sw", "find.word"},
			{"n", "<leader>sd", "find.diagnostics"}, {"n", "<leader>sh", "find.help"}, {"n", "<leader>sk", "find.keymaps"}, {"n", "<leader>sR", "find.resume"},
			{"n", "<leader>sC", "find.commands"}, {"n", "<leader>e", "explorer.toggle"},
			{"n", "gd", "code.definition"}, {"n", "gr", "code.references"}, {"n", "gI", "code.implementation"}, {"n", "gy", "code.type_definition"},
			{"n", "gD", "code.declaration"}, {"n", "K", "code.hover"}, {"n", "gK", "code.signature"}, {"i", "<C-k>", "code.signature"},
			{"nx", "<leader>ca", "code.action"}, {"n", "<leader>cr", "code.rename"}, {"nx", "<leader>cf", "code.format"},
			{"n", "]h", "git.next_hunk"}, {"n", "[h", "git.prev_hunk"}, {"n", "<leader>ghs", "git.stage_hunk"}, {"n", "<leader>ghr", "git.reset_hunk"},
			{"n", "<leader>ghp", "git.preview_hunk"}, {"n", "<leader>ghb", "git.blame_line"}, {"n", "<leader>ghd", "git.diff"},
		}},
}

// shoshinKeys are ShoshinVim's own additions, labeled as such.
var shoshinKeys = []presetKey{{"n", "<leader>?", "guide"}}

func keymapStyle(id string) (KeymapStyle, bool) {
	for _, st := range KeymapStyles {
		if st.ID == id {
			return st, true
		}
	}
	return KeymapStyle{}, false
}

// KeymapStyleFor exposes a style's description to the interface.
func KeymapStyleFor(id string) (KeymapStyle, bool) { return keymapStyle(id) }

// Keymap is one effective shortcut with where it came from.
type EffectiveKeymap struct {
	Mode, Key, Action, Label, Group, Source string
}

func actionAvailable(s State, a KeyAction) bool {
	if a.Needs == "" {
		return true
	}
	v, _ := Value(s, a.Needs, "native").(string)
	return v != "native" && v != "none" && v != ""
}

// EffectiveKeymaps combines the chosen style with the user's own shortcuts. A user
// entry for the same mode and key replaces the style's key, or removes it (NoAction).
func EffectiveKeymaps(s State) []EffectiveKeymap {
	s = withLegacyWorkspace(s)
	overrides := map[string]bool{}
	for _, k := range s.Keymaps {
		overrides[k.Mode+"\x00"+k.Key] = true
	}
	var out []EffectiveKeymap
	add := func(k presetKey, source string) {
		if overrides[k.Mode+"\x00"+k.Key] {
			return
		}
		if a, ok := keyAction(k.Action); ok && actionAvailable(s, a) {
			out = append(out, EffectiveKeymap{k.Mode, k.Key, a.ID, a.Label, a.Group, source})
		}
	}
	if st, ok := keymapStyle(Value(s, "keymaps.style", "none").(string)); ok {
		for _, k := range st.keys {
			add(k, st.Source)
		}
		for _, k := range shoshinKeys {
			add(k, "ShoshinVim")
		}
	}
	for _, k := range s.Keymaps {
		if k.Action == NoAction {
			continue
		}
		if a, ok := keyAction(k.Action); ok {
			out = append(out, EffectiveKeymap{k.Mode, k.Key, a.ID, a.Label, a.Group, "yours"})
		}
	}
	return out
}

var (
	keyModes   = regexp.MustCompile(`^[nvxsoict]{1,6}$`)
	keyPattern = regexp.MustCompile(`^[^\s\x00]{1,40}$`)
)

func validateKeymaps(s State) error {
	seen := map[string]bool{}
	for _, k := range s.Keymaps {
		if !keyModes.MatchString(k.Mode) {
			return fmt.Errorf("keymap mode %q must be letters from nvxsoict", k.Mode)
		}
		if !keyPattern.MatchString(k.Key) {
			return fmt.Errorf("invalid keymap key %q", k.Key)
		}
		if k.Action != NoAction {
			if cmd, ok := strings.CutPrefix(k.Action, CommandAction); ok {
				if cmd == "" || len(cmd) > 200 || strings.ContainsAny(cmd, "\n\r") {
					return fmt.Errorf("invalid command for %s", k.Key)
				}
			} else if _, ok := keyAction(k.Action); !ok {
				return fmt.Errorf("unsupported mapping action %q", k.Action)
			}
		}
		id := k.Mode + ":" + k.Key
		if seen[id] {
			return fmt.Errorf("conflicting mapping %s", id)
		}
		seen[id] = true
	}
	return nil
}

// KeymapConflicts reports keys that make another key wait: with "gr" mapped, typing
// "grn" pauses first. Styles that do this on purpose use nowait and are not reported.
func KeymapConflicts(s State) []string {
	keys := EffectiveKeymaps(s)
	var out []string
	for _, a := range keys {
		for _, b := range keys {
			if a.Key != b.Key && strings.HasPrefix(b.Key, a.Key) && sharesMode(a.Mode, b.Mode) && !nowait(a) {
				out = append(out, fmt.Sprintf("%s (%s) waits because %s also starts with it", KeyLabel(s, a.Key), a.Label, KeyLabel(s, b.Key)))
			}
		}
	}
	sort.Strings(out)
	return out
}

func sharesMode(a, b string) bool { return strings.ContainsAny(a, b) }

// nowait marks LazyVim's gr, which it maps with nowait so Neovim's gr* keys do not delay it.
func nowait(k EffectiveKeymap) bool { return k.Source == "LazyVim" && k.Key == "gr" }

func leaderKey(s State) string {
	switch Value(s, "keymaps.leader", "space") {
	case "comma":
		return ","
	case "backslash":
		return "\\"
	}
	return " "
}

// withLegacyWorkspace maps the earlier all-in-one "rich" workspace onto today's
// separate choices, so configurations saved before them keep their behavior.
func withLegacyWorkspace(s State) State {
	if Value(s, "experience.layout", "native") != "rich" {
		return s
	}
	next := Clone(s)
	for id, v := range map[string]any{"keymaps.style": "lazyvim", "keymaps.leader": "space", "workspace.start_screen": "snacks",
		"appearance.indent": "snacks", "appearance.notifications": "snacks"} {
		if _, ok := next.Values[id]; !ok {
			next.Values[id] = v
		}
	}
	return next
}

func keymapsLua(s State) string {
	keys := EffectiveKeymaps(s)
	if len(keys) == 0 {
		return ""
	}
	var b strings.Builder
	style, _ := keymapStyle(Value(withLegacyWorkspace(s), "keymaps.style", "none").(string))
	if style.ID != "" {
		fmt.Fprintf(&b, "-- Shortcuts in the %s style (%s), then your own.\n", style.Label, style.URL)
	}
	b.WriteString("local function map(modes, key, action, desc, opts)\n  vim.keymap.set(vim.split(modes, ''), key, action, vim.tbl_extend('force', { desc = desc, silent = true }, opts or {}))\nend\n")
	source := ""
	for _, k := range keys {
		if k.Source != source {
			fmt.Fprintf(&b, "-- %s\n", sourceHeading(k.Source))
			source = k.Source
		}
		a, _ := keyAction(k.Action)
		rhs := LuaString(a.RHS)
		if a.Lua {
			rhs = a.RHS
		}
		extra := ""
		if nowait(k) {
			extra = ", { nowait = true }"
		}
		fmt.Fprintf(&b, "map(%s, %s, %s, %s%s)\n", LuaString(k.Mode), LuaString(k.Key), rhs, LuaString(k.Label), extra)
	}
	b.WriteString(guideLua(s, keys))
	return b.String()
}

func sourceHeading(source string) string {
	switch source {
	case "yours":
		return "Your shortcuts"
	case "ShoshinVim":
		return "ShoshinVim additions"
	}
	return "From " + source
}

// guideLua builds :ShoshinHelp from the shortcuts actually configured.
func guideLua(s State, keys []EffectiveKeymap) string {
	groups := map[string][]string{}
	var order []string
	for _, k := range keys {
		if _, ok := groups[k.Group]; !ok {
			order = append(order, k.Group)
		}
		label := k.Label
		if mode := ModeLabel(k.Mode); mode != "" {
			label += " (" + mode + ")"
		}
		groups[k.Group] = append(groups[k.Group], fmt.Sprintf("  %-18s %s", KeyLabel(s, k.Key), label))
	}
	var lines []string
	lines = append(lines, "SHORTCUT GUIDE", "Press q to close.", "")
	for _, g := range order {
		lines = append(lines, strings.ToUpper(g))
		lines = append(lines, groups[g]...)
		lines = append(lines, "")
	}
	var quoted []string
	for _, l := range lines {
		quoted = append(quoted, LuaString(l))
	}
	return `vim.api.nvim_create_user_command('ShoshinHelp', function()
  local lines = { ` + strings.Join(quoted, ", ") + ` }
  local buf = vim.api.nvim_create_buf(false, true)
  vim.api.nvim_buf_set_lines(buf, 0, -1, false, lines)
  vim.bo[buf].modifiable = false
  local width = math.min(72, vim.o.columns - 4)
  local height = math.min(#lines, vim.o.lines - 4)
  vim.api.nvim_open_win(buf, true, { relative = 'editor', width = width, height = height, row = math.max(0, math.floor((vim.o.lines - height) / 2) - 1), col = math.max(0, math.floor((vim.o.columns - width) / 2)), border = 'rounded', title = ' Shortcuts ', title_pos = 'center' })
  vim.keymap.set('n', 'q', '<cmd>close<CR>', { buffer = buf, silent = true })
end, { desc = 'Shortcut guide' })
`
}

var specialKey = regexp.MustCompile(`<[^<>]+>`)

// KeyLabel spells a key in words with the chosen leader, for example "<leader>sf"
// becomes "Space sf" and "<C-h>" becomes "Ctrl+h".
func KeyLabel(s State, key string) string {
	var parts []string
	for key != "" {
		loc := specialKey.FindStringIndex(key)
		if loc == nil {
			parts = append(parts, key)
			break
		}
		if loc[0] > 0 {
			parts = append(parts, key[:loc[0]])
		}
		parts = append(parts, specialKeyName(s, key[loc[0]+1:loc[1]-1]))
		key = key[loc[1]:]
	}
	return strings.Join(parts, " ")
}

func specialKeyName(s State, name string) string {
	switch strings.ToLower(name) {
	case "leader":
		return map[string]string{" ": "Space", ",": "Comma", "\\": "Backslash"}[leaderKey(s)]
	case "localleader":
		return "LocalLeader"
	case "esc":
		return "Esc"
	case "cr", "enter", "return":
		return "Enter"
	case "tab":
		return "Tab"
	case "bs":
		return "Backspace"
	case "space":
		return "Space"
	}
	modifiers := map[string]string{"c": "Ctrl", "s": "Shift", "m": "Alt", "a": "Alt", "d": "Cmd"}
	if len(name) > 2 && name[1] == '-' {
		if mod, ok := modifiers[strings.ToLower(name[:1])]; ok {
			return mod + "+" + specialKeyName(s, name[2:])
		}
	}
	return name
}

// ModeLabel names the modes of a shortcut other than plain normal mode, such as
// "normal & visual"; it is empty for normal mode alone.
func ModeLabel(modes string) string {
	if modes == "n" {
		return ""
	}
	names := map[rune]string{'n': "normal", 'v': "visual", 'x': "visual", 's': "select", 'o': "operator-pending", 'i': "insert", 'c': "command line", 't': "terminal"}
	var out []string
	seen := map[string]bool{}
	for _, m := range modes {
		if n := names[m]; n != "" && !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return strings.Join(out, " & ")
}

// ValidShortcutKey reports whether a key can be used in a shortcut, such as <leader>ff.
func ValidShortcutKey(key string) bool { return keyPattern.MatchString(key) }
