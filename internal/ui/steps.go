package ui

import (
	"fmt"
	"strings"

	"github.com/shoshindev/shoshinvim/internal/engine"
)

// An option sets one or more catalog values. Every option sets explicit values so a
// saved state always shows which choice was made.
type option struct {
	label, detail, short string
	values               map[string]any
	recommended          bool
	swatch               []string
	plugin               string                  // Registry ID shown for transparency; empty means built into Neovim.
	group                string                  // Heading shown above the first option of a group.
	match                func(engine.State) bool // Overrides exact value matching.
}

// name is how the choice reads inside a one-line summary.
func (o option) name() string {
	if o.short != "" {
		return o.short
	}
	return o.label
}

// A question is a single decision. In a multi question every option is an independent
// on/off choice; in a form section each question is one row cycled with left/right.
// An input question asks for text instead, stored in input.setting.
type question struct {
	id, title, help string
	options         []option
	dynamic         func(engine.State) []option
	input           *inputSpec
	multi           bool
	skip            func(engine.State) bool
}

type inputSpec struct {
	setting, placeholder string
	lookup               bool // Submitting looks the repository up on GitHub.
}

// resolve fills options that depend on earlier answers.
func (q question) resolve(s engine.State) question {
	if q.dynamic != nil {
		q.options = q.dynamic(s)
	}
	return q
}

type section struct {
	id, title, intro string
	form             bool
	questions        []question
}

func on(id string) map[string]any  { return map[string]any{id: true} }
func off(id string) map[string]any { return map[string]any{id: false} }
func is(id string, v any) map[string]any {
	return map[string]any{id: v}
}
func toggle(id, title, help string, recommendOn bool, onLabel, offLabel string) question {
	return question{id: id, title: title, help: help, options: []option{
		{label: onLabel, values: on(id), recommended: recommendOn},
		{label: offLabel, values: off(id), recommended: !recommendOn},
	}}
}

var languageIDs = []string{"typescript", "html", "css", "json", "lua"}

// noLanguages is true once the user has decided on languages and selected none.
func noLanguages(s engine.State) bool {
	decided := false
	for _, id := range languageIDs {
		v, ok := s.Values["language."+id+".lsp"]
		if v == true {
			return false
		}
		decided = decided || ok
	}
	return decided
}

var sections = []section{
	{id: "editor", title: "Editor", form: true,
		intro: "How text looks and behaves. Every row starts at the recommended value.",
		questions: []question{
			toggle("options.number", "Line numbers", "Show absolute line numbers in the left margin.", true, "on", "off"),
			toggle("options.relativenumber", "Relative numbers", "Show each line's distance from the cursor, handy for jumps like 5j.", false, "on", "off"),
			toggle("options.wrap", "Wrap long lines", "Off keeps code shaped as written; long lines scroll sideways.", false, "on", "off"),
			{id: "indent", title: "Indent with", help: "What the Tab key and automatic indentation insert.", options: []option{
				{label: "spaces", values: on("options.expandtab"), recommended: true},
				{label: "tabs", values: off("options.expandtab")},
			}},
			{id: "options.tabstop", title: "Tab width", help: "How many columns a tab character occupies.", options: []option{
				{label: "2", values: is("options.tabstop", 2.0), recommended: true},
				{label: "4", values: is("options.tabstop", 4.0)},
				{label: "8", values: is("options.tabstop", 8.0)},
			}},
			{id: "options.shiftwidth", title: "Indent width", help: "How many columns one indentation step uses.", options: []option{
				{label: "2", values: is("options.shiftwidth", 2.0), recommended: true},
				{label: "4", values: is("options.shiftwidth", 4.0)},
				{label: "8", values: is("options.shiftwidth", 8.0)},
			}},
			{id: "search-case", title: "Search case", help: "Smart case ignores case until you type a capital letter.", options: []option{
				{label: "smart", values: map[string]any{"options.ignorecase": true, "options.smartcase": true}, recommended: true},
				{label: "ignore", values: map[string]any{"options.ignorecase": true, "options.smartcase": false}},
				{label: "exact", values: map[string]any{"options.ignorecase": false, "options.smartcase": false}},
			}},
			{id: "options.inccommand", title: "Replace preview", help: "Preview :substitute changes while you type; split also lists matches outside the window.", options: []option{
				{label: "split", values: is("options.inccommand", "split"), recommended: true},
				{label: "inline", values: is("options.inccommand", "nosplit")},
				{label: "off", values: is("options.inccommand", "")},
			}},
			{id: "options.scrolloff", title: "Context lines", help: "Lines kept visible above and below the cursor while scrolling.", options: []option{
				{label: "8", values: is("options.scrolloff", 8.0), recommended: true},
				{label: "4", values: is("options.scrolloff", 4.0)},
				{label: "0", values: is("options.scrolloff", 0.0)},
			}},
			{id: "options.sidescrolloff", title: "Context columns", help: "Columns kept visible beside the cursor when lines do not wrap.", options: []option{
				{label: "8", values: is("options.sidescrolloff", 8.0), recommended: true},
				{label: "0", values: is("options.sidescrolloff", 0.0)},
			}},
			toggle("options.undofile", "Persistent undo", "Undo history survives closing and reopening a file.", true, "on", "off"),
			{id: "options.confirm", title: "Unsaved changes", help: "When quitting with unsaved changes, ask what to do instead of failing.", options: []option{
				{label: "ask", values: on("options.confirm"), recommended: true},
				{label: "refuse", values: off("options.confirm")},
			}},
			{id: "splits", title: "New splits open", help: "Where :split and :vsplit place the new window.", options: []option{
				{label: "right & below", values: map[string]any{"options.splitright": true, "options.splitbelow": true}, recommended: true},
				{label: "left & above", values: map[string]any{"options.splitright": false, "options.splitbelow": false}},
			}},
			{id: "options.mouse", title: "Mouse", help: "Click, scroll and select with the mouse.", options: []option{
				{label: "all modes", values: is("options.mouse", "a"), recommended: true},
				{label: "normal & visual", values: is("options.mouse", "nv")},
				{label: "off", values: is("options.mouse", "")},
			}},
			{id: "options.clipboard", title: "Clipboard", help: "Shared means yank and paste use the system clipboard.", options: []option{
				{label: "shared", values: is("options.clipboard", "unnamedplus"), recommended: true},
				{label: "Neovim only", values: is("options.clipboard", "")},
			}},
			toggle("options.cursorline", "Cursor line", "Highlight the line the cursor is on.", true, "highlight", "off"),
			{id: "invisibles", title: "Invisible characters", help: "Reveal trailing spaces (and optionally tabs) that are otherwise hard to see.", options: []option{
				{label: "trailing spaces", values: map[string]any{"options.list": true, "options.listchars": "tab:  ,trail:·,nbsp:␣"}, recommended: true},
				{label: "tabs & spaces", values: map[string]any{"options.list": true, "options.listchars": "tab:» ,trail:·,nbsp:␣"}},
				{label: "hidden", values: off("options.list")},
			}},
			{id: "options.signcolumn", title: "Sign column", help: "Always keeps room for Git and diagnostic signs so text never shifts.", options: []option{
				{label: "always", values: is("options.signcolumn", "yes"), recommended: true},
				{label: "when needed", values: is("options.signcolumn", "auto")},
			}},
			{id: "timing", title: "Key timing", help: "Fast shows shortcut hints sooner and refreshes diagnostics and Git signs quickly.", options: []option{
				{label: "fast", values: map[string]any{"options.timeoutlen": 400.0, "options.updatetime": 250.0}, recommended: true},
				{label: "Neovim defaults", values: map[string]any{"options.timeoutlen": 1000.0, "options.updatetime": 4000.0}},
			}},
		}},
	{id: "appearance", title: "Appearance", intro: "The look of the editor.", questions: []question{
		{id: "appearance.colorscheme", title: "Color scheme", help: "The colors of the whole editor. Swatches show each theme's real colors.", dynamic: themeOptions},
		{id: "appearance.variant", title: "Variant", help: "Themes come in several versions; light versions are marked.", dynamic: variantOptions, skip: func(s engine.State) bool {
			t, ok := currentTheme(s)
			return !ok || len(t.Variants) < 2
		}},
		{id: "custom-theme-repository", title: "Theme repository", help: "Any GitHub repository with a colors folder, as owner/name. It is pinned to its current commit; ShoshinVim has not reviewed its code.",
			input: &inputSpec{setting: "appearance.custom_theme.repository", placeholder: "owner/name", lookup: true}, skip: notCustomTheme},
		{id: "custom-theme-colorscheme", title: "Which colorscheme?", help: "The colorschemes this repository provides at the pinned commit.", dynamic: customSchemeOptions,
			skip: func(s engine.State) bool { return notCustomTheme(s) || len(customSchemeOptions(s)) == 0 }},
		{id: "custom-theme-name", title: "Colorscheme name", help: "GitHub did not list this repository's colorschemes; type the name used with :colorscheme.",
			input: &inputSpec{setting: "appearance.custom_theme.colorscheme", placeholder: "name"}, skip: func(s engine.State) bool { return notCustomTheme(s) || len(customSchemeOptions(s)) > 0 }},
		{id: "appearance.statusline", title: "Status line", help: "The bar at the bottom of the screen.", options: []option{
			{label: "lualine.nvim", short: "lualine", plugin: "lualine", detail: "Mode, Git branch, diagnostics and file details in one configurable bar", values: is("appearance.statusline", "lualine"), recommended: true},
			{label: "mini.statusline", short: "mini.statusline", plugin: "mini.statusline", detail: "A small, fast statusline from mini.nvim; kickstart's choice", values: is("appearance.statusline", "mini.statusline")},
			{label: "Built in", short: "built-in status line", detail: "Neovim's standard status line", values: is("appearance.statusline", "native")},
		}},
		{id: "appearance.buffers", title: "Open files", help: "How open files (buffers) are shown across the top.", options: []option{
			{label: "bufferline.nvim", short: "bufferline", plugin: "bufferline", detail: "Tabs for open files with diagnostics; Shift+H and Shift+L switch", values: is("appearance.buffers", "bufferline"), recommended: true},
			{label: "barbar.nvim", short: "barbar", plugin: "barbar", detail: "Tabs for open files with reordering and a buffer picker", values: is("appearance.buffers", "barbar")},
			{label: "mini.tabline", short: "mini.tabline", plugin: "mini.tabline", detail: "A minimal tab line from mini.nvim", values: is("appearance.buffers", "mini.tabline")},
			{label: "No tabs", short: "no buffer tabs", detail: "Switch files with the finder or :bnext; kickstart's default", values: is("appearance.buffers", "native")},
		}},
		{id: "options.winborder", title: "Popup borders", help: "Borders around hover docs, signature help and other floating windows.", options: []option{
			{label: "Rounded", short: "rounded borders", detail: "Soft corners that separate popups from code", values: is("options.winborder", "rounded"), recommended: true},
			{label: "Square", short: "square borders", detail: "Thin straight lines", values: is("options.winborder", "single")},
			{label: "None", short: "no borders", detail: "Popups blend into the background", values: is("options.winborder", "")},
		}},
	}},
	{id: "workspace", title: "Workspace", intro: "Shortcuts and discoverability.", questions: []question{
		{id: "experience.layout", title: "Keyboard shortcuts", help: "Space is the leader key: it starts every workspace shortcut.", options: []option{
			{label: "ShoshinVim Space shortcuts", short: "Space shortcuts", plugin: "snacks", detail: "ShoshinVim's own set, modeled on LazyVim: groups for files, code, Git and windows, a start screen and a guide on Space ?", values: is("experience.layout", "rich"), recommended: true},
			{label: "Neovim defaults", short: "Neovim default keys", detail: "Only Neovim's built-in keys", values: is("experience.layout", "native")},
		}},
		{id: "navigation.shortcuts", title: "Shortcut hints", help: "Help while you learn the keys.", options: []option{
			{label: "which-key.nvim", short: "shortcut hints", plugin: "which-key", detail: "Pause after Space (or any prefix) to see what comes next", values: is("navigation.shortcuts", "which-key"), recommended: true},
			{label: "No hints", short: "no hints", detail: "Keys work silently", values: is("navigation.shortcuts", "native")},
		}},
	}},
	{id: "navigation", title: "Navigation", intro: "Moving around a project.", questions: []question{
		{id: "navigation.picker", title: "Find files and text", help: "The search window for files, text, buffers, help and more. Text search uses ripgrep.", options: []option{
			{label: "Telescope", short: "Telescope", plugin: "telescope", detail: "The most widely used finder, with many extensions", values: is("navigation.picker", "telescope"), recommended: true},
			{label: "Snacks picker", short: "Snacks picker", plugin: "snacks", detail: "Fast finder from snacks.nvim; LazyVim's default", values: is("navigation.picker", "snacks")},
			{label: "fzf-lua", short: "fzf-lua", plugin: "fzf-lua", detail: "Uses the fzf program for very fast matching on large projects", values: is("navigation.picker", "fzf-lua")},
			{label: "mini.pick", short: "mini.pick", plugin: "mini.pick", detail: "A compact finder from mini.nvim", values: is("navigation.picker", "mini.pick")},
			{label: "Built in", short: "built-in finder", detail: "Type a path with Tab completion; :grep for text", values: is("navigation.picker", "native")},
		}},
		{id: "navigation.explorer", title: "File explorer", help: "Browse and manage files in the project (Space e).", options: []option{
			{label: "neo-tree.nvim", short: "neo-tree", plugin: "neo-tree", detail: "A file tree sidebar with Git status", values: is("navigation.explorer", "neo-tree"), recommended: true},
			{label: "nvim-tree.lua", short: "nvim-tree", plugin: "nvim-tree", detail: "A file tree sidebar; NvChad's choice", values: is("navigation.explorer", "nvim-tree")},
			{label: "Snacks explorer", short: "Snacks explorer", plugin: "snacks", detail: "A sidebar from snacks.nvim; LazyVim's default", values: is("navigation.explorer", "snacks")},
			{label: "oil.nvim", short: "oil", plugin: "oil", detail: "Edit folders like text: rename and move files by editing lines", values: is("navigation.explorer", "oil")},
			{label: "mini.files", short: "mini.files", plugin: "mini.files", detail: "Column view of folders from mini.nvim", values: is("navigation.explorer", "mini.files")},
			{label: "netrw", short: "netrw", detail: "Neovim's built-in directory browser", values: is("navigation.explorer", "native")},
		}},
	}},
	{id: "languages", title: "Languages", intro: "Code intelligence for the languages you use.", questions: []question{
		{id: "languages", title: "Which languages do you use?", multi: true, help: "Language servers add completion, go to definition, hover docs and diagnostics.", options: []option{
			{label: "TypeScript & JavaScript", short: "TypeScript", detail: "typescript-language-server", values: on("language.typescript.lsp"), recommended: true},
			{label: "HTML", short: "HTML", detail: "vscode-html-language-server", values: on("language.html.lsp"), recommended: true},
			{label: "CSS, SCSS & Less", short: "CSS", detail: "vscode-css-language-server", values: on("language.css.lsp"), recommended: true},
			{label: "JSON", short: "JSON", detail: "vscode-json-language-server", values: on("language.json.lsp"), recommended: true},
			{label: "Lua & Neovim config", short: "Lua", detail: "lua-language-server", values: on("language.lua.lsp"), recommended: true},
		}},
		{id: "completion", title: "Completion", help: "Suggestions while you type.", skip: noLanguages, options: []option{
			{label: "blink.cmp", short: "blink.cmp", plugin: "blink.cmp", detail: "Fast completion with docs, snippets and signature help; kickstart's and LazyVim's choice", values: map[string]any{"completion.engine": "blink.cmp", "completion.native": false}, recommended: true},
			{label: "nvim-cmp", short: "nvim-cmp", plugin: "nvim-cmp", detail: "The long-standing completion engine; NvChad's choice", values: map[string]any{"completion.engine": "nvim-cmp", "completion.native": false}},
			{label: "Built in", short: "built-in completion", detail: "Neovim's native language server completion; Ctrl+Y accepts", values: map[string]any{"completion.engine": "native", "completion.native": true}},
			{label: "Off", short: "no auto-completion", detail: "No automatic suggestions", values: map[string]any{"completion.engine": "native", "completion.native": false}},
		}},
		{id: "diagnostics.virtual_text", title: "Errors and warnings", help: "How language server diagnostics are shown.", skip: noLanguages, options: []option{
			{label: "Inline", short: "inline diagnostics", detail: "Messages appear at the end of the affected line", values: on("diagnostics.virtual_text"), recommended: true},
			{label: "Signs only", short: "diagnostic signs", detail: "A mark in the sign column; open details on demand", values: off("diagnostics.virtual_text")},
		}},
		{id: "lsp.provider", title: "Server settings", help: "Where each language server's configuration comes from.", skip: noLanguages, options: []option{
			{label: "Written by ShoshinVim", short: "built-in server settings", detail: "Readable server settings in your config; nothing extra to download", values: is("lsp.provider", "native"), recommended: true},
			{label: "nvim-lspconfig", short: "nvim-lspconfig", plugin: "lspconfig", detail: "Community-maintained defaults on top", values: is("lsp.provider", "lspconfig")},
		}},
	}},
	{id: "formatting", title: "Formatting", intro: "Keeping code tidy.", questions: []question{
		{id: "formatting.provider", title: "Formatter", help: "The tool that rewrites code into a consistent style.", skip: noLanguages, options: []option{
			{label: "conform.nvim", short: "conform (Prettier & StyLua)", plugin: "conform", detail: "Runs Prettier for web files and StyLua for Lua; respects project config files", values: is("formatting.provider", "conform"), recommended: true},
			{label: "Language server", short: "server formatting", detail: "Each server's own formatting, when it has one", values: is("formatting.provider", "native")},
		}},
		{id: "formatting.on_save", title: "When to format", help: "Formatting on request avoids surprise changes in shared projects.", skip: noLanguages, options: []option{
			{label: "When I ask", short: "format on request", detail: "Run :ShoshinFormat (Space c f with Space shortcuts)", values: off("formatting.on_save"), recommended: true},
			{label: "Every save", short: "format on save", detail: "Format automatically before writing the file", values: on("formatting.on_save")},
		}},
	}},
	{id: "tools", title: "Git & tools", intro: "Version control and installing language tools.", questions: []question{
		{id: "git.hunks", title: "Git changes", help: "See what changed since the last commit.", options: []option{
			{label: "gitsigns.nvim", short: "gitsigns", plugin: "gitsigns", detail: "Markers for added, changed and deleted lines; preview, stage and blame", values: is("git.hunks", "gitsigns"), recommended: true},
			{label: "mini.diff", short: "mini.diff", plugin: "mini.diff", detail: "Lightweight change markers with an inline diff overlay, from mini.nvim", values: is("git.hunks", "mini.diff")},
			{label: "Off", short: "no Git markers", detail: "Use Git from the terminal", values: is("git.hunks", "native")},
		}},
		{id: "tools.provider", title: "Language tools", help: "Language servers and formatters are separate programs.", skip: noLanguages, options: []option{
			{label: "mason.nvim", short: "Mason installs tools", plugin: "mason", detail: "After you review, downloads the selected servers and formatters (web tools need Node.js)", values: is("tools.provider", "mason"), recommended: true},
			{label: "I'll install them", short: "tools from PATH", detail: "Use what is already on your PATH", values: is("tools.provider", "system")},
		}},
	}},
}

// ---- themes -------------------------------------------------------------------

func currentTheme(s engine.State) (engine.Theme, bool) {
	value, _ := s.Values["appearance.colorscheme"].(string)
	t, _, ok := engine.ThemeFor(value)
	return t, ok
}

func notCustomTheme(s engine.State) bool {
	return engine.Value(s, "appearance.colorscheme", "") != engine.CustomTheme
}

func themeOptions(s engine.State) []option {
	current, _ := s.Values["appearance.colorscheme"].(string)
	var out []option
	for _, t := range engine.Themes() {
		if t.Hidden {
			continue
		}
		value, swatch := t.Variants[0].ID, t.Variants[0].Swatch
		for _, v := range t.Variants {
			if v.ID == current {
				value, swatch = v.ID, v.Swatch // Keep the chosen variant when revisiting.
			}
		}
		group, detail := "Theme plugins · downloaded and pinned", t.Detail
		if t.Plugin == "" {
			group = "Built into Neovim · no download"
		}
		if n := len(t.Variants); n > 1 {
			detail += fmt.Sprintf(" · %d variants", n)
		}
		family := t
		out = append(out, option{label: t.Label, short: t.Label, detail: detail, plugin: t.Plugin, group: group, swatch: swatch,
			values: is("appearance.colorscheme", value), recommended: t.ID == "tokyonight",
			match: func(s engine.State) bool {
				value, _ := s.Values["appearance.colorscheme"].(string)
				for _, v := range family.Variants {
					if v.ID == value {
						return true
					}
				}
				return false
			}})
	}
	return append(out, option{label: "Another theme from GitHub…", short: "custom theme", group: "Your own",
		detail: "Name any repository; it is pinned to its current commit and marked as not reviewed", values: is("appearance.colorscheme", engine.CustomTheme)})
}

func variantOptions(s engine.State) []option {
	t, ok := currentTheme(s)
	if !ok {
		return nil
	}
	var out []option
	for i, v := range t.Variants {
		detail := "Dark"
		if v.Background == "light" {
			detail = "Light"
		}
		out = append(out, option{label: v.Label, short: v.Label + " variant", detail: detail + " · :colorscheme " + v.Colorscheme, swatch: v.Swatch,
			values: is("appearance.colorscheme", v.ID), recommended: i == 0})
	}
	return out
}

// lookupTheme is replaced in tests so the custom theme flow runs without the network.
var lookupTheme = engine.LookupTheme

// themeLookups remembers repositories looked up in this session, keyed by repository@revision.
var themeLookups = map[string]engine.ThemeLookup{}

func customSchemeOptions(s engine.State) []option {
	repo, _ := s.Values["appearance.custom_theme.repository"].(string)
	rev, _ := s.Values["appearance.custom_theme.revision"].(string)
	names := themeLookups[repo+"@"+rev].Colorschemes
	if name, _ := s.Values["appearance.custom_theme.colorscheme"].(string); len(names) == 0 && name != "" {
		names = []string{name}
	}
	var out []option
	for i, name := range names {
		out = append(out, option{label: name, short: name, detail: ":colorscheme " + name,
			values: is("appearance.custom_theme.colorscheme", name), recommended: i == 0})
	}
	return out
}

func sameValue(a, b any) bool {
	if x, ok := a.(int); ok {
		a = float64(x)
	}
	if x, ok := b.(int); ok {
		b = float64(x)
	}
	return a == b
}

// matches reports whether every value of an option is present in the state.
func matches(s engine.State, o option) bool {
	if o.match != nil {
		return o.match(s)
	}
	for id, v := range o.values {
		current, ok := s.Values[id]
		if !ok || !sameValue(current, v) {
			return false
		}
	}
	return true
}

// answered reports whether the state already holds a value for this question.
func answered(s engine.State, q question) bool {
	if q.input != nil {
		_, ok := s.Values[q.input.setting]
		return ok
	}
	for _, o := range q.resolve(s).options {
		for id := range o.values {
			if _, ok := s.Values[id]; ok {
				return true
			}
		}
	}
	return false
}

func recommendedIndex(q question) int {
	for i, o := range q.options {
		if o.recommended {
			return i
		}
	}
	return 0
}

// currentIndex is the option matching the state, the recommendation for unanswered
// questions, or -1 for a value set outside the guided choices.
func currentIndex(s engine.State, q question) int {
	q = q.resolve(s)
	if !answered(s, q) {
		return recommendedIndex(q)
	}
	for i, o := range q.options {
		if matches(s, o) {
			return i
		}
	}
	return -1
}

// checked returns the selection of a multi question.
func checked(s engine.State, q question) []bool {
	q = q.resolve(s)
	out := make([]bool, len(q.options))
	for i, o := range q.options {
		if answered(s, q) {
			out[i] = matches(s, o)
		} else {
			out[i] = o.recommended
		}
	}
	return out
}

func (q question) skipped(s engine.State) bool { return q.skip != nil && q.skip(s) }

func (sec section) skipped(s engine.State) bool {
	for _, q := range sec.questions {
		if !q.skipped(s) {
			return false
		}
	}
	return true
}

func unset(s *engine.State, id string) {
	delete(s.Values, id)
	delete(s.Explicit, id)
}

func setValues(c engine.Catalog, s *engine.State, values map[string]any, explicit bool) error {
	for _, id := range sortedKeys(values) {
		if err := c.Set(s, id, values[id], explicit); err != nil {
			return err
		}
	}
	return nil
}

// choose applies an option and the settings that follow from it.
func choose(c engine.Catalog, s engine.State, o option) (engine.State, error) {
	next := engine.Clone(s)
	if err := setValues(c, &next, o.values, true); err != nil {
		return s, err
	}
	return next, derive(c, &next)
}

// chooseMany applies a multi question: selected options on, the rest off.
func chooseMany(c engine.Catalog, s engine.State, q question, selected []bool) (engine.State, error) {
	next := engine.Clone(s)
	for i, o := range q.options {
		for id := range o.values {
			if err := c.Set(&next, id, selected[i], true); err != nil {
				return s, err
			}
		}
	}
	return next, derive(c, &next)
}

// derive keeps implementation details consistent with the user's decisions.
func derive(c engine.Catalog, s *engine.State) error {
	if s.Values == nil {
		s.Values = map[string]any{}
	}
	if noLanguages(*s) {
		for _, id := range []string{"completion.native", "completion.engine", "diagnostics.virtual_text", "lsp.provider", "formatting.provider", "formatting.on_save", "tools.provider"} {
			unset(s, id)
		}
	}
	if notCustomTheme(*s) {
		for _, id := range []string{"appearance.custom_theme.repository", "appearance.custom_theme.revision", "appearance.custom_theme.colorscheme"} {
			unset(s, id)
		}
	}
	switch engine.Value(*s, "appearance.statusline", "native") {
	case "lualine":
		if err := setValues(c, s, map[string]any{"options.showmode": false, "options.laststatus": 3.0}, false); err != nil {
			return err
		}
	case "mini.statusline":
		unset(s, "options.laststatus")
		if err := c.Set(s, "options.showmode", false, false); err != nil {
			return err
		}
	default:
		unset(s, "options.showmode")
		unset(s, "options.laststatus")
	}
	if engine.Enabled(*s, "completion.native") {
		if err := setValues(c, s, map[string]any{"options.completeopt": "menu,menuone,noselect,popup", "options.pumheight": 12.0}, false); err != nil {
			return err
		}
	} else {
		unset(s, "options.completeopt")
		unset(s, "options.pumheight")
	}
	manager := "none"
	if len(engine.Plugins(*s)) > 0 {
		manager = "lazy"
	}
	return c.Set(s, "plugins.manager", manager, false)
}

// recommendedState answers every question with its recommendation.
func recommendedState(c engine.Catalog) (engine.State, error) {
	s := engine.NewState()
	for _, sec := range sections {
		for _, q := range sec.questions {
			if q.skipped(s) || q.input != nil {
				continue
			}
			q = q.resolve(s)
			var err error
			if q.multi {
				s, err = chooseMany(c, s, q, checked(engine.NewState(), q))
			} else {
				s, err = choose(c, s, q.options[recommendedIndex(q)])
			}
			if err != nil {
				return s, err
			}
		}
	}
	return s, nil
}

// summary describes the choices of one section on a single line.
func summary(s engine.State, sec section) string {
	if sec.skipped(s) {
		return "skipped"
	}
	if sec.form {
		changed := 0
		for _, q := range sec.questions {
			if currentIndex(s, q) != recommendedIndex(q) {
				changed++
			}
		}
		text := fmt.Sprintf("%d settings", len(sec.questions))
		switch changed {
		case 0:
			return text + " · all recommended"
		case 1:
			return text + " · 1 changed"
		default:
			return fmt.Sprintf("%s · %d changed", text, changed)
		}
	}
	var parts []string
	for _, q := range sec.questions {
		if q.skipped(s) {
			continue
		}
		q = q.resolve(s)
		if q.input != nil {
			if v, _ := s.Values[q.input.setting].(string); v != "" {
				parts = append(parts, v)
			}
			continue
		}
		if q.multi {
			var names []string
			for i, c := range checked(s, q) {
				if c {
					names = append(names, q.options[i].name())
				}
			}
			if len(names) == 0 {
				names = []string{"no languages"}
			}
			parts = append(parts, strings.Join(names, ", "))
			continue
		}
		if i := currentIndex(s, q); i >= 0 {
			parts = append(parts, q.options[i].name())
		} else {
			parts = append(parts, q.title+": custom")
		}
	}
	return strings.Join(parts, " · ")
}
