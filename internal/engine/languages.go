package engine

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
)

//go:embed data/languages.json
var languageData []byte

// Language is one language ShoshinVim sets up: its server, formatter and parsers.
type Language struct {
	ID          string         `json:"id"`
	Label       string         `json:"label"`
	Short       string         `json:"short,omitempty"` // Name in summaries when the label is long.
	Group       string         `json:"group"`
	Recommended bool           `json:"recommended,omitempty"`
	Server      LanguageServer `json:"server"`
	Formatter   *Formatter     `json:"formatter,omitempty"`
	Parsers     []string       `json:"parsers"`
}

// LanguageServer is a server's recipe. Name is its nvim-lspconfig name. Basic holds the
// Lua fields only the plugin-free recipe needs (nvim-lspconfig has its own); Always holds
// ShoshinVim's additions used with either provider.
type LanguageServer struct {
	Name      string   `json:"name"`
	Command   string   `json:"command"`
	Args      []string `json:"args,omitempty"`
	Package   string   `json:"package,omitempty"` // Mason package; empty when Mason has none.
	Needs     []string `json:"needs,omitempty"`   // Runtimes the server needs, such as node.
	Filetypes []string `json:"filetypes"`
	Roots     []string `json:"roots,omitempty"`
	Basic     string   `json:"basic,omitempty"`
	Always    string   `json:"always,omitempty"`
	Note      string   `json:"note,omitempty"` // Where to get it when Mason has no package.
}

// Formatter is the conform.nvim formatter for a language; languages without one are
// formatted by their server.
type Formatter struct {
	Name      string   `json:"name"`
	Label     string   `json:"label"`
	Command   string   `json:"command"`
	Package   string   `json:"package,omitempty"`
	Needs     []string `json:"needs,omitempty"`
	Filetypes []string `json:"filetypes"`
}

// Name is how summaries refer to the language.
func (l Language) Name() string {
	if l.Short != "" {
		return l.Short
	}
	return l.Label
}

// RuntimeName is how the interface names a runtime a language needs.
func RuntimeName(id string) string {
	names := map[string]string{"node": "Node.js", "python3": "Python", "go": "Go", "cargo": "Rust (rustup)", "java": "Java 21+", "dotnet": ".NET SDK", "ruby": "Ruby", "elixir": "Elixir", "zig": "Zig", "terraform": "Terraform"}
	if n, ok := names[id]; ok {
		return n
	}
	return id
}

// runtimes are programs language tools need; ShoshinVim never installs them.
var runtimes = map[string]Dependency{
	"node":      {ID: "node", Purpose: "Run JavaScript-based language tools", Instruction: "Install Node.js with your package manager (for example: brew install node)."},
	"python3":   {ID: "python3", Purpose: "Run Python-based language tools", Instruction: "Install Python 3 with your package manager."},
	"go":        {ID: "go", Purpose: "Go toolchain; gopls and goimports are built with it", Instruction: "Install Go from https://go.dev/dl or your package manager."},
	"cargo":     {ID: "cargo", Purpose: "Rust toolchain for rust-analyzer and rustfmt", Instruction: "Install Rust with rustup (https://rustup.rs); it also provides rustfmt."},
	"java":      {ID: "java", Purpose: "Java 21 or later for Java and Kotlin tools", Instruction: "Install a JDK 21 or later (for example: brew install openjdk)."},
	"dotnet":    {ID: "dotnet", Purpose: ".NET SDK for csharp-ls", Instruction: "Install the .NET SDK from https://dotnet.microsoft.com/download."},
	"ruby":      {ID: "ruby", Purpose: "Ruby for ruby-lsp", Instruction: "Install Ruby 3 or later (for example with rbenv or your package manager)."},
	"elixir":    {ID: "elixir", Purpose: "Elixir for ElixirLS", Instruction: "Install Elixir from https://elixir-lang.org/install.html."},
	"zig":       {ID: "zig", Purpose: "Zig for zls", Instruction: "Install Zig from https://ziglang.org/download or your package manager."},
	"terraform": {ID: "terraform", Purpose: "Terraform CLI for formatting and validation", Instruction: "Install Terraform from https://developer.hashicorp.com/terraform/install."},
}

var (
	languageOnce sync.Once
	languageList []Language
)

// Languages lists every language ShoshinVim offers, in the order shown.
func Languages() []Language {
	languageOnce.Do(func() {
		if err := json.Unmarshal(languageData, &languageList); err != nil {
			panic(err)
		}
	})
	return languageList
}

func chosenLanguages(s State) []Language {
	var out []Language
	for _, l := range Languages() {
		if Enabled(s, "language."+l.ID+".lsp") {
			out = append(out, l)
		}
	}
	return out
}

func anyLanguage(s State) bool { return len(chosenLanguages(s)) > 0 }

// needsTools is true when a choice needs a separate program that Mason could install.
func needsTools(s State) bool { return anyLanguage(s) || treesitter(s) || linting(s) || debugging(s) }

func luaList(items []string) string { return quoted(items) }

// serverCommand is the server's command line, honoring a custom executable path.
func serverCommand(s State, l Language) []string {
	return append([]string{toolCommand(s, l.Server.Command)}, l.Server.Args...)
}

// lspLua enables each chosen server. With nvim-lspconfig its maintained recipe is used
// as is, plus ShoshinVim's few additions; without it ShoshinVim writes a basic recipe.
func lspLua(s State) string {
	if !anyLanguage(s) {
		return ""
	}
	var b strings.Builder
	lspconfig := Value(s, "lsp.provider", "native") == "lspconfig"
	if lspconfig {
		b.WriteString("-- Server recipes come from nvim-lspconfig; only the settings below are ShoshinVim's.\n")
	} else {
		b.WriteString("-- Basic server recipes written by ShoshinVim (no nvim-lspconfig).\n")
	}
	for _, l := range chosenLanguages(s) {
		srv := l.Server
		var fields []string
		if !lspconfig {
			fields = append(fields, "cmd = { "+luaList(serverCommand(s, l))+" }", "filetypes = { "+luaList(srv.Filetypes)+" }")
			if len(srv.Roots) > 0 {
				fields = append(fields, "root_markers = { "+luaList(srv.Roots)+" }")
			}
			if srv.Basic != "" {
				fields = append(fields, strings.TrimSuffix(srv.Basic, ","))
			}
		} else if toolCommand(s, srv.Command) != srv.Command {
			fields = append(fields, "cmd = { "+luaList(serverCommand(s, l))+" }") // Your own executable.
		}
		if srv.Always != "" {
			fields = append(fields, strings.TrimSuffix(srv.Always, ","))
		}
		if len(fields) > 0 {
			fmt.Fprintf(&b, "vim.lsp.config(%s, { %s })\n", LuaString(srv.Name), strings.Join(fields, ", "))
		}
		fmt.Fprintf(&b, "if not vim.g.shoshinvim_probe and vim.fn.executable(%s) == 1 then vim.lsp.enable(%s) end\n", LuaString(toolCommand(s, srv.Command)), LuaString(srv.Name))
	}
	return b.String()
}

// formattersByFiletype maps each chosen language's file types to its conform formatter.
func formattersByFiletype(s State) string {
	byFT := map[string]string{}
	for _, l := range chosenLanguages(s) {
		if l.Formatter == nil {
			continue
		}
		for _, ft := range l.Formatter.Filetypes {
			byFT[ft] = l.Formatter.Name
		}
	}
	var parts []string
	for _, ft := range sorted(byFT) {
		key := ft
		if !identifier.MatchString(ft) {
			key = "[" + LuaString(ft) + "]"
		}
		parts = append(parts, fmt.Sprintf("%s = { %s }", key, LuaString(byFT[ft])))
	}
	return strings.Join(parts, ", ")
}

// Formatters lists the formatters the chosen languages use, for summaries.
func Formatters(s State) []Formatter {
	seen := map[string]bool{}
	var out []Formatter
	for _, l := range chosenLanguages(s) {
		if f := l.Formatter; f != nil && !seen[f.Name] {
			seen[f.Name] = true
			out = append(out, *f)
		}
	}
	return out
}

// languagePackages are the Mason packages the chosen languages need.
func languagePackages(s State) map[string]bool {
	pkgs := map[string]bool{}
	for _, l := range chosenLanguages(s) {
		if l.Server.Package != "" && toolCommand(s, l.Server.Command) == l.Server.Command {
			pkgs[l.Server.Package] = true
		}
		if f := l.Formatter; f != nil && f.Package != "" && Value(s, "formatting.provider", "native") == "conform" && toolCommand(s, f.Command) == f.Command {
			pkgs[f.Package] = true
		}
	}
	return pkgs
}

// MasonProvides names the programs Mason downloads for a state, so reviews list only
// what the user must install themselves.
func MasonProvides(s State) map[string]bool {
	out := map[string]bool{}
	if Value(s, "tools.provider", "system") != "mason" {
		return out
	}
	for _, l := range chosenLanguages(s) {
		if l.Server.Package != "" && toolCommand(s, l.Server.Command) == l.Server.Command {
			out[l.Server.Command] = true
		}
		if f := l.Formatter; f != nil && f.Package != "" && toolCommand(s, f.Command) == f.Command {
			out[f.Command] = true
		}
	}
	if treesitter(s) {
		out["tree-sitter"] = true
	}
	for _, name := range linters(s) {
		out[name] = true
	}
	if debugging(s) {
		out["js-debug-adapter"] = true
	}
	return out
}

// languageDependencies lists each chosen language's server, formatter and runtimes.
func languageDependencies(s State) []Dependency {
	var list []Dependency
	needs := map[string]bool{}
	mason := Value(s, "tools.provider", "system") == "mason"
	for _, l := range chosenLanguages(s) {
		instruction := "Install " + l.Server.Command + " yourself, or select tools.provider=mason"
		if l.Server.Package == "" {
			instruction = l.Server.Command + " " + l.Server.Note
		}
		list = append(list, Dependency{ID: l.Server.Command, Purpose: l.Label + " language intelligence", Instruction: instruction})
		for _, r := range l.Server.Needs {
			needs[r] = true
		}
		if f := l.Formatter; f != nil && Value(s, "formatting.provider", "native") == "conform" {
			list = append(list, Dependency{ID: f.Command, Purpose: l.Label + " formatting"})
			for _, r := range f.Needs {
				needs[r] = true
			}
		}
	}
	if mason && needs["node"] {
		needs["npm"] = true
	}
	for _, id := range sortedKeys(needs) {
		d, ok := runtimes[id]
		if id == "npm" {
			d, ok = Dependency{ID: "npm", Purpose: "Mason installs JavaScript-based tools with npm"}, true
		}
		if ok {
			list = append(list, d)
		}
	}
	return dedupeDependencies(list)
}

func dedupeDependencies(list []Dependency) []Dependency {
	seen := map[string]bool{}
	var out []Dependency
	for _, d := range list {
		if !seen[d.ID] {
			seen[d.ID] = true
			out = append(out, d)
		}
	}
	return out
}

func sortedKeys(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// toolExecutables are the programs a custom tools.path.<name> setting may point to.
func toolExecutables() map[string]bool {
	out := map[string]bool{}
	for _, l := range Languages() {
		out[l.Server.Command] = true
		if l.Formatter != nil {
			out[l.Formatter.Command] = true
		}
	}
	return out
}
