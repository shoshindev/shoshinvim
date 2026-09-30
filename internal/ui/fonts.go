package ui

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/shoshindev/shoshinvim/internal/engine"
)

// iconSample is the glyph test: Nerd Font icons for a file, a folder, Git, a warning
// and a search. Without a Nerd Font they show as boxes or question marks.
const iconSample = "\U000f0219  \U000f024b      "

// iconOptions recommend Nerd Font icons only where they are likely to work, since
// missing glyphs would spoil every screen.
func iconOptions(engine.State) []option {
	nerd := detectNerdFont()
	return []option{
		{label: "I see icons", short: "Nerd Font icons", plugin: "mini.icons",
			detail: "Icons for files, Git, diagnostics and menus, as in LazyVim and NvChad", values: is("appearance.icons", "nerd"), recommended: nerd},
		{label: "I see boxes or question marks", short: "plain symbols",
			detail: "Letters and symbols every font has; switch once you install a Nerd Font", values: is("appearance.icons", "plain"), recommended: !nerd},
	}
}

// detectNerdFont is replaced in tests, so recommendations do not depend on the machine.
var detectNerdFont = nerdFontLikely

// nerdFontLikely reports whether the terminal can show Nerd Font glyphs: Ghostty,
// WezTerm and kitty include them, and other terminals need one installed.
func nerdFontLikely() bool {
	if bundlesNerdSymbols(os.Getenv) {
		return true
	}
	if os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_TTY") != "" {
		return false // The terminal's fonts are on another machine.
	}
	for _, dir := range fontDirs() {
		if hasNerdFont(dir) {
			return true
		}
	}
	return false
}

func bundlesNerdSymbols(env func(string) string) bool {
	switch strings.ToLower(env("TERM_PROGRAM")) {
	case "ghostty", "wezterm":
		return true
	}
	return env("KITTY_WINDOW_ID") != "" || env("TERM") == "xterm-kitty" || env("TERM") == "xterm-ghostty"
}

func fontDirs() []string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		return []string{filepath.Join(home, "Library", "Fonts"), "/Library/Fonts"}
	case "windows":
		return []string{filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "Windows", "Fonts"), filepath.Join(os.Getenv("WINDIR"), "Fonts")}
	}
	dirs := []string{filepath.Join(home, ".local", "share", "fonts"), filepath.Join(home, ".fonts"), "/usr/local/share/fonts", "/usr/share/fonts"}
	if data := os.Getenv("XDG_DATA_HOME"); data != "" {
		dirs = append(dirs, filepath.Join(data, "fonts"))
	}
	return dirs
}

// nerdFontFile matches Nerd Font file names: FiraCodeNerdFont-Regular.ttf,
// "Hack Nerd Font Mono.ttf" or the short CaskaydiaCoveNF-Regular.ttf.
var nerdFontFile = regexp.MustCompile(`((?i:nerd.?font).*|[a-z]NF[MP]?([-_ ].*)?)\.(?i:ttf|otf|ttc)$`)

// hasNerdFont looks a few folders deep, so a large system font folder stays quick.
func hasNerdFont(root string) bool {
	found := false
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return nil // Unreadable entries are skipped.
		case d.IsDir():
			if strings.Count(strings.TrimPrefix(path, root), string(filepath.Separator)) > 3 {
				return filepath.SkipDir
			}
		case nerdFontFile.MatchString(d.Name()):
			found = true
			return fs.SkipAll
		}
		return nil
	})
	return found
}
