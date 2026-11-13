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
	source               string                  // Where a non-plugin choice comes from, such as a keymap style's origin.
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
	group           string // Heading above form rows.
	options         []option
	dynamic         func(engine.State) []option
	input           *inputSpec
	keymaps         bool     // The shortcut editor screen.
	actions         []string // Shortcut actions whose configured keys the step names.
	multi           bool
	skip            func(engine.State) bool
}

type inputSpec struct {
	setting, placeholder string
	lookup               bool // Submitting looks the repository up on GitHub.
}

// languageOptions lists every language, grouped, saying what each one installs.
func languageOptions(s engine.State) []option {
	mason := engine.Value(s, "tools.provider", "system") == "mason"
	var out []option
	for _, l := range engine.Languages() {
		out = append(out, option{label: l.Label, short: l.Name(), group: l.Group, detail: languageDetail(l, mason), values: on("language." + l.ID + ".lsp"), recommended: l.Recommended})
	}
	return out
}

// languageDetail names a language's tools and, with Mason, the runtimes they need.
func languageDetail(l engine.Language, mason bool) string {
	if l.Server.Package == "" {
		return l.Server.Command + ", which " + l.Server.Note
	}
	tools := []string{l.Server.Package}
	programs := []string{l.Server.Command}
	needs := append([]string{}, l.Server.Needs...)
	if f := l.Formatter; f != nil {
		tools = append(tools, f.Label)
		programs = append(programs, f.Command)
		needs = append(needs, f.Needs...)
		if f.Package == "" {
			tools = tools[:1] // Comes with the language's own toolchain.
		}
	}
	if !mason {
		return "Needs " + strings.Join(programs, " and ") + " on your PATH"
	}
	text := strings.Join(tools, " + ") + ", installed by Mason"
	var names []string
	seen := map[string]bool{}
	for _, n := range needs {
		if !seen[n] {
			seen[n] = true
			names = append(names, engine.RuntimeName(n))
		}
	}
	if len(names) > 0 {
		text += "; needs " + strings.Join(names, " and ")
	}
	return text
}

// formatterOptions names the formatters conform.nvim runs for the chosen languages.
func formatterOptions(s engine.State) []option {
	var labels []string
	seen := map[string]bool{}
	next := engine.Clone(s)
	next.Values["formatting.provider"] = "conform"
	for _, f := range engine.Formatters(next) {
		if !seen[f.Label] {
			seen[f.Label] = true
			labels = append(labels, f.Label)
		}
	}
	detail, short := "Your languages have no separate formatter; each server formats instead", "conform"
	if len(labels) > 0 {
		detail = "Runs " + joinWords(labels) + ", following each project's own config files"
		short = "conform (" + strings.Join(labels, ", ") + ")"
	}
	return []option{
		{label: "conform.nvim", short: short, plugin: "conform", detail: detail, values: is("formatting.provider", "conform"), recommended: true},
		{label: "Language server", short: "server formatting", detail: "Each server's own formatting, when it has one", values: is("formatting.provider", "native")},
	}
}

// joinWords lists items as "a, b and c".
func joinWords(items []string) string {
	if len(items) < 2 {
		return strings.Join(items, "")
	}
	return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
}

// syntaxOptions counts the parsers Tree-sitter would compile for the chosen languages.
func syntaxOptions(s engine.State) []option {
	next := engine.Clone(s)
	next.Values["syntax.engine"] = "nvim-treesitter"
	n := len(engine.Parsers(next))
	return []option{
		{label: "nvim-treesitter", short: "Tree-sitter", plugin: "nvim-treesitter", detail: fmt.Sprintf("Precise colors and indentation from real parsers; compiles %d parsers when you download", n), values: is("syntax.engine", "nvim-treesitter"), recommended: true},
		{label: "Built in", short: "built-in syntax", detail: "Neovim's syntax files; Lua, Markdown and help files already use its bundled parsers", values: is("syntax.engine", "native")},
	}
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

func withGroup(group string, q question) question {
	q.group = group
	return q
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

// noLanguages is true once the user has decided on languages and selected none.
func noLanguages(s engine.State) bool {
	decided := false
	for _, l := range engine.Languages() {
		v, ok := s.Values["language."+l.ID+".lsp"]
		if v == true {
			return false
		}
		decided = decided || ok
	}
	return decided
}

// noTypeScript skips the debugger, whose adapter is for JavaScript and TypeScript.
func noTypeScript(s engine.State) bool { return !engine.Enabled(s, "language.typescript.lsp") }

// surroundOptions names each plugin's keys; LazyVim's style moves mini.surround under gs.
func surroundOptions(s engine.State) []option {
	prefix, source := "s", "kickstart's choice"
	if engine.Value(s, "keymaps.style", "none") == "lazyvim" {
		prefix, source = "gs", "LazyVim's keys"
	}
	return []option{
		{label: "mini.surround", short: "mini.surround", plugin: "mini.surround", detail: fmt.Sprintf("%[1]sa adds, %[1]sd deletes, %[1]sr replaces: %[1]saiw\" quotes a word; %[2]s", prefix, source), values: is("coding.surround", "mini.surround"), recommended: true},
		{label: "nvim-surround", short: "nvim-surround", plugin: "nvim-surround", detail: "ys adds, ds deletes, cs changes: ysiw\" quotes a word; the keys of vim-surround", values: is("coding.surround", "nvim-surround")},
		{label: "Off", short: "no surround", detail: "Edit the brackets and quotes yourself", values: is("coding.surround", "none")},
	}
}

var sections = []section{
	{id: "editor", title: "Editor", form: true,
		intro: "How text looks and behaves. Recommendations follow kickstart.nvim's init.lua where it sets a value, otherwise LazyVim's.",
		questions: []question{
			withGroup("Display", toggle("options.number", "Line numbers", "Show absolute line numbers in the left margin.", true, "on", "off")),
			withGroup("Display", toggle("options.relativenumber", "Relative numbers", "Show each line's distance from the cursor, handy for jumps like 5j.", false, "on", "off")),
			withGroup("Display", toggle("options.cursorline", "Cursor line", "Highlight the line the cursor is on.", true, "highlight", "off")),
			withGroup("Display", question{id: "options.signcolumn", title: "Sign column", help: "Always keeps room for Git and diagnostic signs so text never shifts.", options: []option{
				{label: "always", values: is("options.signcolumn", "yes"), recommended: true},
				{label: "when needed", values: is("options.signcolumn", "auto")},
			}}),
			withGroup("Display", question{id: "invisibles", title: "Invisible characters", help: "Reveal tabs and trailing spaces that are otherwise hard to see.", options: []option{
				{label: "tabs & spaces", values: map[string]any{"options.list": true, "options.listchars": "tab:» ,trail:·,nbsp:␣"}, recommended: true},
				{label: "trailing spaces", values: map[string]any{"options.list": true, "options.listchars": "tab:  ,trail:·,nbsp:␣"}},
				{label: "hidden", values: off("options.list")},
			}}),
			withGroup("Display", question{id: "options.colorcolumn", title: "Column guide", help: "A vertical line at a column, to keep lines short.", options: []option{
				{label: "off", values: is("options.colorcolumn", ""), recommended: true},
				{label: "80", values: is("options.colorcolumn", "80")},
				{label: "100", values: is("options.colorcolumn", "100")},
				{label: "120", values: is("options.colorcolumn", "120")},
			}}),
			withGroup("Display", question{id: "options.fillchars", title: "Empty lines", help: "Neovim marks lines past the end of a file with ~.", options: []option{
				{label: "blank", values: is("options.fillchars", "eob: "), recommended: true},
				{label: "~", values: is("options.fillchars", "eob:~")},
			}}),
			withGroup("Wrapping", toggle("options.wrap", "Wrap long lines", "Off keeps code shaped as written; long lines scroll sideways.", false, "on", "off")),
			withGroup("Wrapping", question{id: "wrapped-lines", title: "Wrapped lines", help: "When lines wrap, break at word boundaries and keep the indentation.", options: []option{
				{label: "tidy", values: map[string]any{"options.linebreak": true, "options.breakindent": true}, recommended: true},
				{label: "plain", values: map[string]any{"options.linebreak": false, "options.breakindent": false}},
			}}),
			withGroup("Indentation", question{id: "indent", title: "Indent with", help: "What the Tab key and automatic indentation insert.", options: []option{
				{label: "spaces", values: on("options.expandtab"), recommended: true},
				{label: "tabs", values: off("options.expandtab")},
			}}),
			withGroup("Indentation", question{id: "options.tabstop", title: "Tab width", help: "How many columns a tab character occupies.", options: []option{
				{label: "2", values: is("options.tabstop", 2.0), recommended: true},
				{label: "4", values: is("options.tabstop", 4.0)},
				{label: "8", values: is("options.tabstop", 8.0)},
			}}),
			withGroup("Indentation", question{id: "options.shiftwidth", title: "Indent width", help: "How many columns one indentation step uses.", options: []option{
				{label: "2", values: is("options.shiftwidth", 2.0), recommended: true},
				{label: "4", values: is("options.shiftwidth", 4.0)},
				{label: "8", values: is("options.shiftwidth", 8.0)},
			}}),
			withGroup("Indentation", question{id: "editor.detect_indent", title: "Detect indentation", help: "Adapt to each file's existing indentation (guess-indent.nvim); project .editorconfig files always apply.", options: []option{
				{label: "guess-indent", plugin: "guess-indent", values: is("editor.detect_indent", "guess-indent"), recommended: true},
				{label: "off", values: is("editor.detect_indent", "none")},
			}}),
			withGroup("Search", question{id: "search-case", title: "Search case", help: "Smart case ignores case until you type a capital letter.", options: []option{
				{label: "smart", values: map[string]any{"options.ignorecase": true, "options.smartcase": true}, recommended: true},
				{label: "ignore", values: map[string]any{"options.ignorecase": true, "options.smartcase": false}},
				{label: "exact", values: map[string]any{"options.ignorecase": false, "options.smartcase": false}},
			}}),
			withGroup("Search", question{id: "options.inccommand", title: "Replace preview", help: "Preview :substitute changes while you type; split also lists matches outside the window.", options: []option{
				{label: "split", values: is("options.inccommand", "split"), recommended: true},
				{label: "inline", values: is("options.inccommand", "nosplit")},
				{label: "off", values: is("options.inccommand", "")},
			}}),
			withGroup("Editing", toggle("options.undofile", "Persistent undo", "Undo history survives closing and reopening a file.", true, "on", "off")),
			withGroup("Editing", question{id: "options.clipboard", title: "Clipboard", help: "Shared means yank and paste use the system clipboard.", options: []option{
				{label: "shared", values: is("options.clipboard", "unnamedplus"), recommended: true},
				{label: "Neovim only", values: is("options.clipboard", "")},
			}}),
			withGroup("Editing", question{id: "options.mouse", title: "Mouse", help: "Click, scroll and select with the mouse.", options: []option{
				{label: "all modes", values: is("options.mouse", "a"), recommended: true},
				{label: "normal & visual", values: is("options.mouse", "nv")},
				{label: "off", values: is("options.mouse", "")},
			}}),
			withGroup("Editing", toggle("editor.highlight_yank", "Highlight on copy", "Briefly highlight text after it is yanked (copied).", true, "on", "off")),
			withGroup("Editing", toggle("editor.restore_cursor", "Reopen at last position", "Return to where you left off when reopening a file.", true, "on", "off")),
			withGroup("Editing", toggle("editor.trim_whitespace", "Trim trailing spaces", "Remove spaces at the end of lines when saving (not in Markdown, where they mean a line break).", false, "on", "off")),
			withGroup("Editing", toggle("editor.spell_prose", "Spell check prose", "Check spelling in Markdown, plain text and commit messages.", true, "on", "off")),
			withGroup("Windows", question{id: "splits", title: "New splits open", help: "Where :split and :vsplit place the new window.", options: []option{
				{label: "right & below", values: map[string]any{"options.splitright": true, "options.splitbelow": true}, recommended: true},
				{label: "left & above", values: map[string]any{"options.splitright": false, "options.splitbelow": false}},
			}}),
			withGroup("Windows", question{id: "options.scrolloff", title: "Context lines", help: "Lines kept visible above and below the cursor while scrolling.", options: []option{
				{label: "10", values: is("options.scrolloff", 10.0), recommended: true},
				{label: "8", values: is("options.scrolloff", 8.0)},
				{label: "4", values: is("options.scrolloff", 4.0)},
				{label: "0", values: is("options.scrolloff", 0.0)},
			}}),
			withGroup("Windows", question{id: "options.sidescrolloff", title: "Context columns", help: "Columns kept visible beside the cursor when lines do not wrap.", options: []option{
				{label: "8", values: is("options.sidescrolloff", 8.0), recommended: true},
				{label: "0", values: is("options.sidescrolloff", 0.0)},
			}}),
			withGroup("Windows", question{id: "options.confirm", title: "Unsaved changes", help: "When quitting with unsaved changes, ask what to do instead of failing.", options: []option{
				{label: "ask", values: on("options.confirm"), recommended: true},
				{label: "refuse", values: off("options.confirm")},
			}}),
			withGroup("Timing", question{id: "timing", title: "Key timing", help: "Fast shows shortcut hints sooner and refreshes diagnostics and Git signs quickly.", options: []option{
				{label: "fast", values: map[string]any{"options.timeoutlen": 300.0, "options.updatetime": 250.0}, recommended: true},
				{label: "Neovim defaults", values: map[string]any{"options.timeoutlen": 1000.0, "options.updatetime": 4000.0}},
			}}),
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
		{id: "appearance.icons", title: "Icons", help: "Do these look like icons?   " + iconSample + "   Nerd Font icons need a Nerd Font in your terminal.", dynamic: iconOptions},
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
		{id: "appearance.indent", title: "Indent guides", help: "Vertical lines that show indentation levels.", options: []option{
			{label: "indent-blankline.nvim", short: "indent-blankline", plugin: "indent-blankline", detail: "Thin guides at every indentation level; NvChad's choice", values: is("appearance.indent", "indent-blankline"), recommended: true},
			{label: "Snacks indent", short: "Snacks indent", plugin: "snacks", detail: "Guides with the current scope highlighted; LazyVim's choice", values: is("appearance.indent", "snacks")},
			{label: "mini.indentscope", short: "mini.indentscope", plugin: "mini.indentscope", detail: "Shows only the scope around the cursor, from mini.nvim", values: is("appearance.indent", "mini.indentscope")},
			{label: "None", short: "no indent guides", detail: "No guides", values: is("appearance.indent", "none")},
		}},
		{id: "appearance.notifications", title: "Notifications", help: "How messages and language server progress appear.", options: []option{
			{label: "Snacks notifier", short: "Snacks notifications", plugin: "snacks", detail: "Messages as tidy popups, and nicer input prompts; LazyVim's choice", values: is("appearance.notifications", "snacks"), recommended: true},
			{label: "fidget.nvim", short: "fidget", plugin: "fidget", detail: "Language server progress in a corner; kickstart's choice", values: is("appearance.notifications", "fidget")},
			{label: "Built in", short: "built-in messages", detail: "Messages in the command line", values: is("appearance.notifications", "none")},
		}},
		{id: "appearance.cmdline", title: "Command line", help: "Where : commands, searches and messages appear.", options: []option{
			{label: "noice.nvim", short: "floating command line", plugin: "noice", detail: "Commands in a popup near the top, searches at the bottom, messages in a corner; LazyVim's choice", values: is("appearance.cmdline", "noice"), recommended: true},
			{label: "Built in", short: "built-in command line", detail: "Neovim's command line at the bottom of the screen", values: is("appearance.cmdline", "native")},
		}},
		{id: "options.winborder", title: "Popup borders", help: "Borders around hover docs, signature help and other floating windows.", options: []option{
			{label: "Rounded", short: "rounded borders", detail: "Soft corners that separate popups from code", values: is("options.winborder", "rounded"), recommended: true},
			{label: "Square", short: "square borders", detail: "Thin straight lines", values: is("options.winborder", "single")},
			{label: "None", short: "no borders", detail: "Popups blend into the background", values: is("options.winborder", "")},
		}},
	}},
	{id: "workspace", title: "Workspace", intro: "Shortcuts, hints and the start screen.", questions: []question{
		{id: "keymaps.style", title: "Shortcut style", help: "Community keymap sets, copied from their source. You can change any key in the next step.", options: []option{
			{label: "kickstart.nvim", short: "kickstart.nvim shortcuts", source: "kickstart", detail: "Small and easy to learn: Space s… to search, Space f to format, \\ for files",
				values: is("keymaps.style", "kickstart"), recommended: true},
			{label: "LazyVim", short: "LazyVim shortcuts", source: "lazyvim", detail: "Larger: Space Space for files, Space c… for code, Shift+H/L between buffers",
				values: is("keymaps.style", "lazyvim")},
			{label: "Neovim defaults only", short: "Neovim default keys", detail: "No extra shortcuts; built-in keys like grn (rename) still work",
				values: is("keymaps.style", "none")},
		}},
		{id: "keymaps.leader", title: "Leader key", help: "Leader shortcuts start with this key. kickstart.nvim and LazyVim both use Space.", options: []option{
			{label: "Space", short: "Space leader", detail: "Easy to reach with either thumb", values: is("keymaps.leader", "space"), recommended: true},
			{label: "Comma", short: "comma leader", detail: "A common choice in older Vim configurations", values: is("keymaps.leader", "comma")},
			{label: "Backslash", short: "backslash leader", detail: "Neovim's default", values: is("keymaps.leader", "backslash")},
		}},
		{id: "keymaps-editor", title: "Your shortcuts", help: "Every shortcut you will have, and where it comes from. Change a key, turn one off, or add your own.", keymaps: true},
		{id: "navigation.shortcuts", title: "Shortcut hints", help: "A popup that shows what comes next after you press a prefix such as the leader key.", options: []option{
			{label: "which-key.nvim", short: "which-key", plugin: "which-key", detail: "Pause after a prefix to see the next keys, grouped", values: is("navigation.shortcuts", "which-key"), recommended: true},
			{label: "mini.clue", short: "mini.clue", plugin: "mini.clue", detail: "A lighter hint window from mini.nvim", values: is("navigation.shortcuts", "mini.clue")},
			{label: "No hints", short: "no hints", detail: "Keys work silently", values: is("navigation.shortcuts", "native")},
		}},
		{id: "workspace.start_screen", title: "Start screen", help: "What Neovim shows when you start it without a file.", options: []option{
			{label: "Snacks dashboard", short: "Snacks dashboard", plugin: "snacks", detail: "Shortcuts to find, explore and recent files; LazyVim's choice", values: is("workspace.start_screen", "snacks"), recommended: true},
			{label: "mini.starter", short: "mini.starter", plugin: "mini.starter", detail: "A minimal start screen from mini.nvim", values: is("workspace.start_screen", "mini.starter")},
			{label: "None", short: "no start screen", detail: "Neovim's own intro message; kickstart's default", values: is("workspace.start_screen", "none")},
		}},
	}},
	{id: "navigation", title: "Navigation", intro: "Moving around a project.", questions: []question{
		{id: "navigation.picker", title: "Find files and text", help: "The search window for files, text, buffers, help and more. Text search uses ripgrep.", actions: []string{"find.files", "find.grep", "find.buffers"}, options: []option{
			{label: "Telescope", short: "Telescope", plugin: "telescope", detail: "The most widely used finder, with many extensions", values: is("navigation.picker", "telescope"), recommended: true},
			{label: "Snacks picker", short: "Snacks picker", plugin: "snacks", detail: "Fast finder from snacks.nvim; LazyVim's default", values: is("navigation.picker", "snacks")},
			{label: "fzf-lua", short: "fzf-lua", plugin: "fzf-lua", detail: "Uses the fzf program for very fast matching on large projects", values: is("navigation.picker", "fzf-lua")},
			{label: "mini.pick", short: "mini.pick", plugin: "mini.pick", detail: "A compact finder from mini.nvim", values: is("navigation.picker", "mini.pick")},
			{label: "Built in", short: "built-in finder", detail: "Type a path with Tab completion; :grep for text", values: is("navigation.picker", "native")},
		}},
		{id: "navigation.explorer", title: "File explorer", help: "Browse and manage files in the project.", actions: []string{"explorer.toggle"}, options: []option{
			{label: "neo-tree.nvim", short: "neo-tree", plugin: "neo-tree", detail: "A file tree sidebar with Git status", values: is("navigation.explorer", "neo-tree"), recommended: true},
			{label: "nvim-tree.lua", short: "nvim-tree", plugin: "nvim-tree", detail: "A file tree sidebar; NvChad's choice", values: is("navigation.explorer", "nvim-tree")},
			{label: "Snacks explorer", short: "Snacks explorer", plugin: "snacks", detail: "A sidebar from snacks.nvim; LazyVim's default", values: is("navigation.explorer", "snacks")},
			{label: "oil.nvim", short: "oil", plugin: "oil", detail: "Edit folders like text: rename and move files by editing lines", values: is("navigation.explorer", "oil")},
			{label: "mini.files", short: "mini.files", plugin: "mini.files", detail: "Column view of folders from mini.nvim", values: is("navigation.explorer", "mini.files")},
			{label: "netrw", short: "netrw", detail: "Neovim's built-in directory browser", values: is("navigation.explorer", "native")},
		}},
	}},
	{id: "languages", title: "Languages", intro: "Code intelligence for the languages you use.", questions: []question{
		{id: "tools.provider", title: "Language tools", help: "Language servers, formatters, linters and the Tree-sitter compiler are separate programs. Mason can download them for you.", options: []option{
			{label: "mason.nvim", short: "Mason installs tools", plugin: "mason", detail: "After you review, downloads the servers, formatters and other tools your choices need", values: is("tools.provider", "mason"), recommended: true},
			{label: "I'll install them", short: "tools from PATH", detail: "Use what is already on your PATH", values: is("tools.provider", "system")},
		}},
		{id: "languages", title: "Which languages do you use?", multi: true, help: "Each gets its language server for completion, go to definition, hover docs and diagnostics, a formatter where one is common, and syntax parsers.", dynamic: languageOptions},
		{id: "syntax.engine", title: "Syntax highlighting", help: "How Neovim reads code to color and indent it.", dynamic: syntaxOptions},
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
		{id: "lsp.provider", title: "Server settings", help: "Each language server needs a recipe: the program to run, its file types and how to find your project.", skip: noLanguages, options: []option{
			{label: "nvim-lspconfig", short: "nvim-lspconfig recipes", plugin: "lspconfig", detail: "Recipes for hundreds of servers, maintained alongside Neovim", values: is("lsp.provider", "lspconfig"), recommended: true},
			{label: "No plugin", short: "basic recipes", detail: "ShoshinVim writes a basic recipe for each server into your config", values: is("lsp.provider", "native")},
		}},
	}},
	{id: "formatting", title: "Formatting", intro: "Keeping code tidy.", questions: []question{
		{id: "formatting.provider", title: "Formatter", help: "The tool that rewrites code into a consistent style.", skip: noLanguages, dynamic: formatterOptions},
		{id: "formatting.on_save", title: "When to format", help: "In a project that uses a different style, formatting on save can rewrite whole files.", skip: noLanguages, actions: []string{"code.format"}, options: []option{
			{label: "Every save", short: "format on save", detail: "Format before writing the file; LazyVim's and AstroNvim's choice", values: on("formatting.on_save"), recommended: true},
			{label: "When I ask", short: "format on request", detail: "Run :ShoshinFormat or its shortcut; kickstart's and NvChad's choice", values: off("formatting.on_save")},
		}},
	}},
	{id: "coding", title: "Coding", intro: "Helpers for writing, checking and debugging code.", questions: []question{
		{id: "coding.pairs", title: "Brackets and quotes", help: "Typing an opening bracket or quote inserts the closing one.", options: []option{
			{label: "nvim-autopairs", short: "nvim-autopairs", plugin: "nvim-autopairs", detail: "Closes pairs as you type, with rules for each language; NvChad's and AstroNvim's choice", values: is("coding.pairs", "nvim-autopairs"), recommended: true},
			{label: "mini.pairs", short: "mini.pairs", plugin: "mini.pairs", detail: "A minimal pair closer from mini.nvim; LazyVim's choice", values: is("coding.pairs", "mini.pairs")},
			{label: "Off", short: "no auto-pairs", detail: "Type both yourself; kickstart's default", values: is("coding.pairs", "none")},
		}},
		{id: "coding.surround", title: "Surround", help: "Add, change or delete the brackets, quotes and tags around text.", dynamic: surroundOptions},
		{id: "coding.textobjects", title: "Text objects", help: "Select around or inside things with a and i, such as va) or ci'.", options: []option{
			{label: "mini.ai", short: "mini.ai", plugin: "mini.ai", detail: "Adds arguments (a), function calls (f), quotes (q) and more, as in cia or vaf", values: is("coding.textobjects", "mini.ai"), recommended: true},
			{label: "Built in", short: "built-in text objects", detail: "Neovim's own: words, sentences, paragraphs, brackets, quotes and tags", values: is("coding.textobjects", "none")},
		}},
		{id: "coding.todo", title: "TODO comments", help: "Highlight TODO, FIXME, HACK and NOTE in comments so they stand out.", options: []option{
			{label: "todo-comments.nvim", short: "todo-comments", plugin: "todo-comments", detail: "Colors each keyword and can list them all; kickstart's, LazyVim's and AstroNvim's choice", values: is("coding.todo", "todo-comments"), recommended: true},
			{label: "mini.hipatterns", short: "mini.hipatterns", plugin: "mini.hipatterns", detail: "Highlights the same keywords with a small pattern plugin from mini.nvim", values: is("coding.todo", "mini.hipatterns")},
			{label: "Off", short: "plain comments", detail: "Comments keep the color scheme's comment color", values: is("coding.todo", "none")},
		}},
		{id: "coding.lint", title: "Linting", help: "Checks beyond what language servers report, such as style rules.", options: []option{
			{label: "Off", short: "no linters", detail: "Language servers already report errors; kickstart's and LazyVim's default for these languages", values: is("coding.lint", "none"), recommended: true},
			{label: "nvim-lint", short: "nvim-lint", plugin: "nvim-lint", detail: "markdownlint for Markdown, and ESLint for JavaScript and TypeScript in projects set up for it", values: is("coding.lint", "nvim-lint")},
		}},
		{id: "coding.debug", title: "Debugger", help: "Breakpoints and stepping through JavaScript and TypeScript run with Node.js.", skip: noTypeScript, options: []option{
			{label: "Off", short: "no debugger", detail: "kickstart and LazyVim leave debugging off until you add it", values: is("coding.debug", "none"), recommended: true},
			{label: "nvim-dap + nvim-dap-ui", short: "nvim-dap-ui", plugin: "nvim-dap-ui", detail: "Panels for variables, breakpoints, call stack and console; kickstart's example", values: is("coding.debug", "nvim-dap-ui")},
			{label: "nvim-dap + nvim-dap-view", short: "nvim-dap-view", plugin: "nvim-dap-view", detail: "One lighter window with a tab for each view", values: is("coding.debug", "nvim-dap-view")},
		}},
	}},
	{id: "tools", title: "Git", intro: "Version control.", questions: []question{
		{id: "git.hunks", title: "Git changes", help: "See what changed since the last commit.", options: []option{
			{label: "gitsigns.nvim", short: "gitsigns", plugin: "gitsigns", detail: "Markers for added, changed and deleted lines; preview, stage and blame", values: is("git.hunks", "gitsigns"), recommended: true},
			{label: "mini.diff", short: "mini.diff", plugin: "mini.diff", detail: "Lightweight change markers with an inline diff overlay, from mini.nvim", values: is("git.hunks", "mini.diff")},
			{label: "Off", short: "no Git markers", detail: "Use Git from the terminal", values: is("git.hunks", "native")},
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
		for _, id := range []string{"completion.native", "completion.engine", "diagnostics.virtual_text", "lsp.provider", "formatting.provider", "formatting.on_save"} {
			unset(s, id)
		}
	}
	if noTypeScript(*s) {
		unset(s, "coding.debug")
	}
	if _, chosen := s.Values["keymaps.style"]; chosen {
		unset(s, "experience.layout") // Replaced by the separate workspace choices.
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
			if q.skipped(s) || q.input != nil || q.keymaps {
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
		if q.keymaps {
			if n := changedShortcuts(s); n > 0 {
				parts = append(parts, fmt.Sprintf("%d shortcut change%s", n, map[bool]string{true: "", false: "s"}[n == 1]))
			}
			continue
		}
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

// ---- shortcuts ------------------------------------------------------------------

func withoutKeymap(keys []engine.Keymap, mode, key string) []engine.Keymap {
	var out []engine.Keymap
	for _, k := range keys {
		if k.Mode != mode || k.Key != key {
			out = append(out, k)
		}
	}
	return out
}

// rebindShortcut moves a shortcut to a new key. A style's key is turned off with an
// explicit entry, so the generated file shows exactly what changed.
func rebindShortcut(s engine.State, k engine.EffectiveKeymap, key string) engine.State {
	next := engine.Clone(s)
	next.Keymaps = withoutKeymap(withoutKeymap(next.Keymaps, k.Mode, k.Key), k.Mode, key)
	if k.Source != "yours" {
		next.Keymaps = append(next.Keymaps, engine.Keymap{Mode: k.Mode, Key: k.Key, Action: engine.NoAction})
	}
	next.Keymaps = append(next.Keymaps, engine.Keymap{Mode: k.Mode, Key: key, Action: k.Action})
	return next
}

// turnOffShortcut removes one of the user's shortcuts or turns off a style's key.
func turnOffShortcut(s engine.State, k engine.EffectiveKeymap) engine.State {
	next := engine.Clone(s)
	next.Keymaps = withoutKeymap(next.Keymaps, k.Mode, k.Key)
	if k.Source != "yours" {
		next.Keymaps = append(next.Keymaps, engine.Keymap{Mode: k.Mode, Key: k.Key, Action: engine.NoAction})
	}
	return next
}

// addShortcut adds the user's own shortcut, replacing any shortcut on the same key.
func addShortcut(s engine.State, mode, key, action string) engine.State {
	next := engine.Clone(s)
	next.Keymaps = append(withoutKeymap(next.Keymaps, mode, key), engine.Keymap{Mode: mode, Key: key, Action: action})
	return next
}

// restoreShortcuts undoes every change to the chosen style.
func restoreShortcuts(s engine.State) engine.State {
	next := engine.Clone(s)
	next.Keymaps = nil
	return next
}

// changedShortcuts counts the user's own entries (additions, moves and removals).
func changedShortcuts(s engine.State) int { return len(s.Keymaps) }
