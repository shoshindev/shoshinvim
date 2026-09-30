package engine

import (
	"regexp"
	"slices"
	"strings"
	"testing"
)

func polishedState() State {
	s, _ := LoadCatalog().Profile(NewState(), "rich-ide")
	s.Values["syntax.engine"] = "nvim-treesitter"
	delete(s.Values, "options.pumheight") // The profile sets it; each test chooses its own.
	return s
}

// A value the state sets always wins over ShoshinVim's refinements.
func TestRefinementsNeverOverrideTheState(t *testing.T) {
	chosen := map[string]any{"smoothscroll": false, "splitkeep": "cursor", "pumheight": 5.0, "pumborder": "single", "ruler": true,
		"shortmess": "filnxtToOF", "foldmethod": "manual", "foldexpr": "0", "foldlevel": 0.0, "foldlevelstart": -1.0, "foldtext": "foldtext()"}
	for name := range chosen {
		if !regexp.MustCompile(`vim\.(o|opt)\.` + name + `\b`).MatchString(refinementsLua(polishedState())) {
			t.Fatalf("%s is not refined, so this test does not cover it", name)
		}
	}
	for name, value := range chosen {
		s := polishedState()
		if err := LoadCatalog().Set(&s, "options."+name, value, true); err != nil {
			t.Fatal(err)
		}
		if regexp.MustCompile(`vim\.(o|opt)\.` + name + `\b`).MatchString(refinementsLua(s)) {
			t.Errorf("the refinements override the chosen %s", name)
		}
	}
	s := polishedState()
	s.Values["options.fillchars"] = "eob: ,fold:-"
	if got := refinementsLua(s); !strings.Contains(got, "fillchars:append({ foldsep = ' ', diff = '╱' })") {
		t.Errorf("fillchars must only add what the chosen value lacks:\n%s", got)
	}
}

func TestNeovimDefaultsStayWithoutEditorOptions(t *testing.T) {
	s, _ := LoadCatalog().Profile(NewState(), "custom")
	files, err := Generate(s)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := files["lua/user_config/options.lua"]; ok {
		t.Fatal("a state without options must keep Neovim's defaults")
	}
}

// Plugins with frames of their own follow the popup border choice; with borders off
// they keep their own frame, and the shortcut guide gets a blank one for its title.
func TestPluginBordersFollowTheChoice(t *testing.T) {
	for winborder, want := range map[any]string{"rounded": "rounded", "single": "single", "": "", nil: ""} {
		s := polishedState()
		s.Values["navigation.shortcuts"] = "which-key"
		delete(s.Values, "options.winborder")
		if winborder != nil {
			s.Values["options.winborder"] = winborder
		}
		files, err := Generate(s)
		if err != nil {
			t.Fatal(err)
		}
		whichKey := strings.Contains(files["lua/user_config/appearance.lua"], "win = { border = ")
		lazy := strings.Contains(files["lua/user_config/plugins.lua"], "ui = { border = ")
		if whichKey != (want != "") || lazy != (want != "") || borderStyle(s) != want {
			t.Errorf("winborder %v: which-key border %v, lazy.nvim border %v, style %q", winborder, whichKey, lazy, borderStyle(s))
		}
		guide := map[any]string{"rounded": "rounded", "single": "single", "": "solid", nil: "rounded"}[winborder]
		if got := guideBorder(s); got != guide {
			t.Errorf("winborder %v: guide border %q, want %q", winborder, got, guide)
		}
	}
}

// noice takes the command line and messages, but not the jobs another choice owns.
func TestNoiceLeavesEachJobWithItsPlugin(t *testing.T) {
	s := polishedState()
	s.Values["appearance.cmdline"] = "noice"
	s.Values["appearance.notifications"] = "fidget"
	s.Values["completion.engine"] = "blink.cmp"
	got := noiceLua(s)
	for _, want := range []string{"progress = { enabled = false }", "signature = { enabled = false }", "['cmp.entry.get_documentation'] = false"} {
		if !strings.Contains(got, want) {
			t.Errorf("noice setup lacks %s:\n%s", want, got)
		}
	}
	regex := false
	for _, p := range Parsers(s) {
		regex = regex || p.Language == "regex"
	}
	if !regex {
		t.Error("noice colors searches with the regex parser")
	}
	s.Values["appearance.cmdline"] = "native"
	if noiceLua(s) != "" || slices.Contains(Plugins(s), "noice") {
		t.Error("the built-in command line needs no noice")
	}
}

// Themes that style only the plugins they are told about are told about every chosen
// plugin, and a variant's own setting survives the theme's setup.
func TestThemesStyleEveryChosenPlugin(t *testing.T) {
	s := polishedState()
	for _, id := range []string{"tokyonight", "solarized-osaka/dark"} {
		s.Values["appearance.colorscheme"] = id
		got := colorschemeLua(s)
		for _, name := range repositoryNames(s) {
			if !strings.Contains(got, "["+LuaString(name)+"] = true") {
				t.Errorf("%s is not told about %s:\n%s", id, name, got)
			}
		}
	}
	for id := range materialPlugins {
		if _, ok := PluginInfo(id); !ok {
			t.Errorf("material.nvim's module %s names no plugin", id)
		}
	}
	s.Values["appearance.colorscheme"] = "everforest/dark-hard"
	if got := colorschemeLua(s); !strings.Contains(got, "everforest_background = 'hard'") || !strings.Contains(got, "everforest_diagnostic_virtual_text = 'colored'") ||
		!strings.Contains(got, "everforest_float_style = 'dim'") {
		t.Errorf("everforest loses its variant, colored diagnostics or dim popups:\n%s", got)
	}
}
