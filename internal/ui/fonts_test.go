package ui

import (
	"os"
	"reflect"
	"testing"

	"github.com/shoshindev/shoshinvim/internal/engine"
)

func TestMain(m *testing.M) {
	detectNerdFont = func() bool { return false } // Recommendations must not depend on this machine's fonts.
	os.Exit(m.Run())
}

func TestIconRecommendationFollowsTheFonts(t *testing.T) {
	defer func() { detectNerdFont = func() bool { return false } }()
	for _, nerd := range []bool{false, true} {
		detectNerdFont = func() bool { return nerd }
		var recommended []string
		for _, o := range iconOptions(engine.NewState()) {
			if o.recommended {
				recommended = append(recommended, o.values["appearance.icons"].(string))
			}
		}
		want := map[bool]string{false: "plain", true: "nerd"}[nerd]
		if !reflect.DeepEqual(recommended, []string{want}) {
			t.Fatalf("with a Nerd Font %v, recommended %v", nerd, recommended)
		}
	}
}

func TestRecommendedNerdFontIconsAddMiniIcons(t *testing.T) {
	defer func() { detectNerdFont = func() bool { return false } }()
	detectNerdFont = func() bool { return true }
	c := engine.LoadCatalog()
	s, err := recommendedState(c)
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Validate(s); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, id := range engine.Plugins(s) {
		found = found || id == "mini.icons"
	}
	if !found || engine.Value(s, "appearance.icons", "") != "nerd" {
		t.Fatalf("Nerd Font icons must install mini.icons: %v", engine.Plugins(s))
	}
}

func TestNerdFontFileNames(t *testing.T) {
	for name, want := range map[string]bool{
		"FiraCodeNerdFont-Regular.ttf": true, "Hack Nerd Font Mono.ttf": true, "SymbolsNerdFontMono-Regular.TTF": true,
		"CaskaydiaCoveNF.ttf": true, "CaskaydiaCoveNF-Regular.otf": true, "JetBrainsMonoNFM-Bold.ttf": true,
		"Menlo.ttc": false, "DejaVuSansMono.ttf": false, "Inconsolata-NF.txt": false, "confnet-Regular.ttf": false, "NerdFont.md": false,
	} {
		if got := nerdFontFile.MatchString(name); got != want {
			t.Errorf("%s: %v, want %v", name, got, want)
		}
	}
}

func TestTerminalsWithNerdSymbols(t *testing.T) {
	for env, want := range map[[2]string]bool{
		{"TERM_PROGRAM", "ghostty"}: true, {"TERM_PROGRAM", "WezTerm"}: true, {"KITTY_WINDOW_ID", "1"}: true, {"TERM", "xterm-kitty"}: true,
		{"TERM_PROGRAM", "Apple_Terminal"}: false, {"TERM_PROGRAM", "iTerm.app"}: false, {"TERM", "xterm-256color"}: false,
	} {
		got := bundlesNerdSymbols(func(k string) string {
			if k == env[0] {
				return env[1]
			}
			return ""
		})
		if got != want {
			t.Errorf("%s=%s: %v, want %v", env[0], env[1], got, want)
		}
	}
}
