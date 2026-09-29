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
type question struct {
	id, title, help string
	options         []option
	multi           bool
	skip            func(engine.State) bool
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
		{id: "appearance.colorscheme", title: "Color scheme", help: "The colors of the whole editor.", options: []option{
			{label: "Tokyo Night", swatch: []string{"#7aa2f7", "#bb9af7", "#7dcfff", "#9ece6a", "#e0af68", "#f7768e"}, detail: "Deep night palette with vivid, readable syntax · downloads tokyonight.nvim", values: is("appearance.colorscheme", "tokyonight"), recommended: true},
			{label: "Habamax", swatch: []string{"#5f87af", "#af87af", "#5f8787", "#87af87", "#d7af87", "#d75f87"}, detail: "Muted dark theme · built in", values: is("appearance.colorscheme", "habamax")},
			{label: "Sorbet", swatch: []string{"#87afff", "#d787ff", "#87d7d7", "#87d787", "#ffd787", "#ff87af"}, detail: "Soft purple dark theme · built in", values: is("appearance.colorscheme", "sorbet")},
			{label: "Retrobox", swatch: []string{"#83a598", "#d3869b", "#8ec07c", "#b8bb26", "#fabd2f", "#fb4934"}, detail: "Warm retro theme · built in", values: is("appearance.colorscheme", "retrobox")},
			{label: "Quiet", swatch: []string{"#eeeeee", "#c6c6c6", "#a8a8a8", "#8a8a8a", "#6c6c6c", "#585858"}, detail: "Minimal grayscale theme · built in", values: is("appearance.colorscheme", "quiet")},
			{label: "Default", swatch: []string{"#a6dbff", "#ffcaff", "#8cf8f7", "#b3f6c0", "#fce094", "#ffc0b9"}, detail: "Neovim's own colors · built in", values: is("appearance.colorscheme", "default")},
		}},
		{id: "appearance.statusline", title: "Status line", help: "The bar at the bottom of the screen.", options: []option{
			{label: "Lualine", short: "Lualine", detail: "Mode, Git branch, diagnostics and file details in one clean bar · downloads lualine.nvim", values: is("appearance.statusline", "lualine"), recommended: true},
			{label: "Built in", short: "built-in status line", detail: "Neovim's standard status line", values: is("appearance.statusline", "native")},
		}},
		{id: "appearance.buffers", title: "Open files", help: "How open files are shown.", options: []option{
			{label: "Tabs across the top", short: "buffer tabs", detail: "One tab per open file; Shift+H and Shift+L switch · downloads bufferline.nvim", values: is("appearance.buffers", "bufferline"), recommended: true},
			{label: "No tabs", short: "no buffer tabs", detail: "Switch files with the finder or :bnext", values: is("appearance.buffers", "native")},
		}},
		{id: "options.winborder", title: "Popup borders", help: "Borders around hover docs, signature help and other floating windows.", options: []option{
			{label: "Rounded", short: "rounded borders", detail: "Soft corners that separate popups from code", values: is("options.winborder", "rounded"), recommended: true},
			{label: "Square", short: "square borders", detail: "Thin straight lines", values: is("options.winborder", "single")},
			{label: "None", short: "no borders", detail: "Popups blend into the background", values: is("options.winborder", "")},
		}},
	}},
	{id: "workspace", title: "Workspace", intro: "Shortcuts and discoverability.", questions: []question{
		{id: "experience.layout", title: "Keyboard shortcuts", help: "Space is the leader key: it starts every workspace shortcut.", options: []option{
			{label: "Space shortcuts", detail: "Grouped shortcuts for files, code, Git and windows, a start screen and a guide on Space ? · uses snacks.nvim", values: is("experience.layout", "rich"), recommended: true},
			{label: "Neovim defaults", short: "Neovim default keys", detail: "Only Neovim's built-in keys", values: is("experience.layout", "native")},
		}},
		{id: "navigation.shortcuts", title: "Shortcut hints", help: "Help while you learn the keys.", options: []option{
			{label: "Show hints", short: "shortcut hints", detail: "Pause after Space (or any prefix) to see what comes next · downloads which-key.nvim", values: is("navigation.shortcuts", "which-key"), recommended: true},
			{label: "No hints", short: "no hints", detail: "Keys work silently", values: is("navigation.shortcuts", "native")},
		}},
	}},
	{id: "navigation", title: "Navigation", intro: "Moving around a project.", questions: []question{
		{id: "navigation.picker", title: "Find files and text", help: "The search window for files, text, buffers and more.", options: []option{
			{label: "Snacks picker", detail: "Fast fuzzy search for files, text, buffers, diagnostics and Git · text search uses ripgrep", values: is("navigation.picker", "snacks"), recommended: true},
			{label: "mini.pick", detail: "Compact fuzzy picker for files and text", values: is("navigation.picker", "mini.pick")},
			{label: "Built in", short: "built-in finder", detail: "Type a path with Tab completion · no download", values: is("navigation.picker", "native")},
		}},
		{id: "navigation.explorer", title: "File explorer", help: "Browse and manage files in the project.", options: []option{
			{label: "Project sidebar", short: "project sidebar", detail: "A file tree beside your code (Space e) · uses snacks.nvim", values: is("navigation.explorer", "snacks"), recommended: true},
			{label: "Oil", detail: "Edit folders like text: rename and move files by editing lines", values: is("navigation.explorer", "oil")},
			{label: "netrw", detail: "Neovim's built-in directory browser", values: is("navigation.explorer", "native")},
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
		{id: "completion.native", title: "Completion", help: "Suggestions from the language server.", skip: noLanguages, options: []option{
			{label: "As you type", short: "completion as you type", detail: "Suggestions appear automatically; Ctrl+Y accepts, Ctrl+N and Ctrl+P choose", values: on("completion.native"), recommended: true},
			{label: "Off", short: "no auto-completion", detail: "No automatic suggestions", values: off("completion.native")},
		}},
		{id: "diagnostics.virtual_text", title: "Errors and warnings", help: "How language server diagnostics are shown.", skip: noLanguages, options: []option{
			{label: "Inline", short: "inline diagnostics", detail: "Messages appear at the end of the affected line", values: on("diagnostics.virtual_text"), recommended: true},
			{label: "Signs only", short: "diagnostic signs", detail: "A mark in the sign column; open details on demand", values: off("diagnostics.virtual_text")},
		}},
		{id: "lsp.provider", title: "Server settings", help: "Where each language server's configuration comes from.", skip: noLanguages, options: []option{
			{label: "Written by ShoshinVim", short: "built-in server settings", detail: "Readable server settings in your config · nothing extra to download", values: is("lsp.provider", "native"), recommended: true},
			{label: "Add nvim-lspconfig", short: "nvim-lspconfig", detail: "Community-maintained defaults on top · downloads nvim-lspconfig", values: is("lsp.provider", "lspconfig")},
		}},
	}},
	{id: "formatting", title: "Formatting", intro: "Keeping code tidy.", questions: []question{
		{id: "formatting.provider", title: "Formatter", help: "The tool that rewrites code into a consistent style.", skip: noLanguages, options: []option{
			{label: "Prettier & StyLua", detail: "Consistent formatting through conform.nvim; respects project config files", values: is("formatting.provider", "conform"), recommended: true},
			{label: "Language server", short: "server formatting", detail: "Each server's own formatting, when it has one", values: is("formatting.provider", "native")},
		}},
		{id: "formatting.on_save", title: "When to format", help: "Formatting on request avoids surprise changes in shared projects.", skip: noLanguages, options: []option{
			{label: "When I ask", short: "format on request", detail: "Run :ShoshinFormat (Space c f with Space shortcuts)", values: off("formatting.on_save"), recommended: true},
			{label: "Every save", short: "format on save", detail: "Format automatically before writing the file", values: on("formatting.on_save")},
		}},
	}},
	{id: "tools", title: "Git & tools", intro: "Version control and installing language tools.", questions: []question{
		{id: "git.hunks", title: "Git changes", help: "See what changed since the last commit.", options: []option{
			{label: "Markers in the gutter", short: "Git markers", detail: "Added, changed and deleted lines; preview hunks · downloads gitsigns.nvim", values: is("git.hunks", "gitsigns"), recommended: true},
			{label: "Off", short: "no Git markers", detail: "Use Git from the terminal", values: is("git.hunks", "native")},
		}},
		{id: "tools.provider", title: "Language tools", help: "Language servers and formatters are separate programs.", skip: noLanguages, options: []option{
			{label: "Install them for me", short: "Mason installs tools", detail: "After you review, Mason downloads the selected servers and formatters (web tools need Node.js)", values: is("tools.provider", "mason"), recommended: true},
			{label: "I'll install them", short: "tools from PATH", detail: "Use what is already on your PATH", values: is("tools.provider", "system")},
		}},
	}},
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
	for _, o := range q.options {
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
		for _, id := range []string{"completion.native", "diagnostics.virtual_text", "lsp.provider", "formatting.provider", "formatting.on_save", "tools.provider"} {
			unset(s, id)
		}
	}
	if engine.Value(*s, "appearance.statusline", "native") == "lualine" {
		if err := setValues(c, s, map[string]any{"options.showmode": false, "options.laststatus": 3.0}, false); err != nil {
			return err
		}
	} else {
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
			if q.skipped(s) {
				continue
			}
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
	for i, part := range parts {
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " · ")
}
