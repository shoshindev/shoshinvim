package engine

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
)

//go:embed data/parsers.json
var parserData []byte

// ParserSource is where nvim-treesitter downloads a parser from, at the revision the
// pinned nvim-treesitter commit names. It is recorded for review; nvim-treesitter's
// own table is what the download uses.
type ParserSource struct {
	Language   string `json:"-"`
	Repository string `json:"repository"`
	Revision   string `json:"revision"`
}

var (
	parserOnce    sync.Once
	parserSources map[string]ParserSource
)

func parserTable() map[string]ParserSource {
	parserOnce.Do(func() {
		if err := json.Unmarshal(parserData, &parserSources); err != nil {
			panic(err)
		}
		for lang, p := range parserSources {
			p.Language = lang
			parserSources[lang] = p
		}
	})
	return parserSources
}

// baseParsers is kickstart.nvim's list: the shell, diffs, Lua, Markdown and Neovim's
// own file types, useful whatever languages are chosen.
var baseParsers = []string{"bash", "c", "diff", "html", "lua", "luadoc", "markdown", "markdown_inline", "query", "vim", "vimdoc"}

func treesitter(s State) bool { return Value(s, "syntax.engine", "native") == "nvim-treesitter" }

// Parsers lists the Tree-sitter parsers a state installs, with their sources.
func Parsers(s State) []ParserSource {
	if !treesitter(s) {
		return nil
	}
	set := map[string]bool{}
	for _, p := range baseParsers {
		set[p] = true
	}
	for _, l := range chosenLanguages(s) {
		for _, p := range l.Parsers {
			set[p] = true
		}
	}
	var out []ParserSource
	for _, lang := range sorted(set) {
		out = append(out, parserTable()[lang])
	}
	return out
}

func parserNames(parsers []ParserSource) []string {
	var out []string
	for _, p := range parsers {
		out = append(out, p.Language)
	}
	sort.Strings(out)
	return out
}

// syntaxLua starts Tree-sitter for every buffer whose language has a parser. Parsers are
// installed by ShoshinVim when downloads are approved, never when Neovim starts.
func syntaxLua(s State) string {
	if !treesitter(s) {
		return ""
	}
	return fmt.Sprintf(`-- Tree-sitter parsers (%s) are compiled by ShoshinVim when you approve downloads.
local base = vim.fn.stdpath('data') .. '/shoshinvim'
local ok, ts = false, nil
if not vim.g.shoshinvim_probe then ok, ts = pcall(require, 'nvim-treesitter') end
if ok then ts.setup({ install_dir = base .. '/parsers' }) end
vim.api.nvim_create_autocmd('FileType', {
  group = vim.api.nvim_create_augroup('ShoshinTreesitter', { clear = true }),
  callback = function(event)
    local lang = vim.treesitter.language.get_lang(event.match)
    if not lang then return end
    local found, has_parser = pcall(vim.treesitter.language.add, lang)
    if not (found and has_parser) or not pcall(vim.treesitter.start, event.buf, lang) then return end
    if ok and vim.treesitter.query.get(lang, 'indents') then
      vim.bo[event.buf].indentexpr = "v:lua.require'nvim-treesitter'.indentexpr()"
    end
  end,
})
`, strings.Join(parserNames(Parsers(s)), ", "))
}
