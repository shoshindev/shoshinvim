package ui

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/shoshindev/shoshinvim/internal/engine"
)

func newModel(t *testing.T) *Model {
	t.Helper()
	// Pin both homes: on Windows the default configuration home contains the temp folder.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	m, err := New(filepath.Join(t.TempDir(), "nvim"), "", true)
	if err != nil {
		t.Fatal(err)
	}
	m.Update(tea.WindowSizeMsg{Width: 110, Height: 36})
	return m
}

func needNeovim(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("nvim"); err != nil {
		t.Skip("requires Neovim 0.12+")
	}
}

// run executes a command synchronously and feeds job messages back, the way the
// Bubble Tea runtime would. Timer-driven messages (spinner, cursor blink) are dropped.
func run(m *Model, cmd tea.Cmd) {
	for cmd != nil {
		switch msg := cmd().(type) {
		case tea.BatchMsg:
			for _, c := range msg {
				run(m, c)
			}
			return
		case progressMsg, jobMsg:
			_, cmd = m.Update(msg)
		default:
			return
		}
	}
}

func press(m *Model, keys ...string) {
	for _, k := range keys {
		var msg tea.KeyPressMsg
		switch k {
		case "enter":
			msg = tea.KeyPressMsg{Code: tea.KeyEnter}
		case "esc":
			msg = tea.KeyPressMsg{Code: tea.KeyEscape}
		case "space":
			msg = tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
		case "down":
			msg = tea.KeyPressMsg{Code: tea.KeyDown}
		case "right":
			msg = tea.KeyPressMsg{Code: tea.KeyRight}
		case "ctrl+c":
			msg = tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
		default:
			msg = tea.KeyPressMsg{Code: rune(k[0]), Text: k}
		}
		_, cmd := m.Update(msg)
		run(m, cmd)
	}
}

func screenText(m *Model) string { return ansi.Strip(m.View().Content) }

// acceptAll presses Enter until the review is reached.
func acceptAll(t *testing.T, m *Model) {
	t.Helper()
	for i := 0; m.screen != reviewScreen; i++ {
		if i > 40 {
			t.Fatalf("review not reached; stuck on screen %d section %d", m.screen, m.sec)
		}
		press(m, "enter")
	}
}

func TestWelcomeOffersNoPresets(t *testing.T) {
	m := newModel(t)
	text := screenText(m)
	for _, preset := range []string{"Rich IDE", "Minimal native", "Web development", "Custom —"} {
		if strings.Contains(text, preset) {
			t.Fatalf("welcome still offers preset %q:\n%s", preset, text)
		}
	}
	if !strings.Contains(text, "Start setup") || !strings.Contains(text, "recommended") {
		t.Fatalf("welcome missing start or recommendation:\n%s", text)
	}
}

func TestLogoLinesHaveTheWordmarkWidth(t *testing.T) {
	for _, line := range newTheme(true, true).Logo() {
		if w := ansi.StringWidth(line); w != logoWidth() {
			t.Fatalf("logo line is %d wide, want %d: %q", w, logoWidth(), line)
		}
	}
}

// Generated start screens use engine.LogoColors; they must stay the welcome's colors.
func TestGeneratedLogoColorsMatchTheTheme(t *testing.T) {
	for _, dark := range []bool{true, false} {
		th := newTheme(dark, false)
		want := engine.LogoColors(dark)
		for y, c := range th.logo {
			r, g, b, _ := c.RGBA()
			if got := fmt.Sprintf("#%02X%02X%02X", r>>8, g>>8, b>>8); got != want[y] {
				t.Fatalf("dark=%v row %d: theme %s, engine.LogoColors %s", dark, y, got, want[y])
			}
		}
	}
}

func TestWelcomeShowsLogoWhenItFits(t *testing.T) {
	m := newModel(t)
	for _, size := range []struct {
		w, h int
		big  bool
	}{{110, 36, true}, {80, 24, true}, {80, 20, false}, {60, 30, false}} {
		m.Update(tea.WindowSizeMsg{Width: size.w, Height: size.h})
		text := screenText(m)
		if big := strings.Contains(text, " █████ ██  ██"); big != size.big {
			t.Fatalf("%d×%d: large logo shown = %v, want %v:\n%s", size.w, size.h, big, size.big, text)
		}
		if !size.big && !strings.Contains(text, "S H O S H I N V I M") {
			t.Fatalf("%d×%d: missing one-line mark:\n%s", size.w, size.h, text)
		}
		if !strings.Contains(text, "Start setup") || !strings.Contains(text, "beginner's mind") {
			t.Fatalf("%d×%d: welcome lost its menu or tagline:\n%s", size.w, size.h, text)
		}
	}
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if text := screenText(m); !strings.Contains(text, "2. Quit") {
		t.Fatalf("80×24: logo pushed the menu off screen:\n%s", text)
	}
}

func TestEverySectionAndQuestionHasARecommendation(t *testing.T) {
	c := engine.LoadCatalog()
	s, err := recommendedState(c)
	if err != nil {
		t.Fatal(err)
	}
	for _, sec := range sections {
		for _, q := range sec.questions {
			if q.input != nil || q.keymaps || q.skipped(s) {
				continue
			}
			q = q.resolve(s)
			count := 0
			for _, o := range q.options {
				if o.recommended {
					count++
				}
				probe := engine.NewState()
				if err := setValues(c, &probe, o.values, true); err != nil {
					t.Fatalf("%s/%s: %v", q.id, o.label, err)
				}
			}
			if count == 0 || (!q.multi && count != 1) {
				t.Fatalf("%s needs exactly one recommended option", q.id)
			}
		}
	}
}

// The recommendations follow what today's leading starter configurations ship.
func TestRecommendedChoicesInstallTheExpectedPlugins(t *testing.T) {
	c := engine.LoadCatalog()
	s, err := recommendedState(c)
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Validate(s); err != nil {
		t.Fatal(err)
	}
	want := []string{"blink.cmp", "bufferline", "conform", "gitsigns", "guess-indent", "indent-blankline", "lspconfig", "lualine", "mason", "mini.ai", "mini.surround", "neo-tree", "noice", "nui", "nvim-autopairs", "nvim-treesitter", "plenary", "snacks", "telescope", "todo-comments", "tokyonight", "which-key"}
	if got := engine.Plugins(s); !reflect.DeepEqual(got, want) {
		t.Fatalf("plugins %v, want %v", got, want)
	}
	if s.Profile != "custom" {
		t.Fatalf("guided state must not claim a preset: %s", s.Profile)
	}
}

// Transparency: every plugin a choice installs is explained by the choice that caused it.
func TestEveryPluginIsExplained(t *testing.T) {
	c := engine.LoadCatalog()
	base, err := recommendedState(c)
	if err != nil {
		t.Fatal(err)
	}
	for _, sec := range sections {
		for _, q := range sec.questions {
			if q.input != nil || q.multi || q.keymaps {
				continue
			}
			for _, o := range q.resolve(base).options {
				if o.values["appearance.colorscheme"] == engine.CustomTheme {
					continue // Explained by its own screen; needs a looked-up repository.
				}
				s, err := choose(c, base, o)
				if err != nil {
					t.Fatalf("%s/%s: %v", q.id, o.label, err)
				}
				m := &Model{draft: s}
				reasons := m.pluginReasons()
				for _, id := range engine.Plugins(s) {
					if reasons[id] == "" {
						t.Fatalf("choosing %s/%s installs %s without saying why", q.id, o.label, id)
					}
				}
				if o.plugin != "" && !slicesContains(engine.Plugins(s), o.plugin) {
					t.Fatalf("%s/%s names %s but does not install it", q.id, o.label, o.plugin)
				}
			}
		}
	}
}

func slicesContains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func TestEnterThroughEverySectionReachesReview(t *testing.T) {
	needNeovim(t)
	m := newModel(t)
	acceptAll(t, m)
	want, _ := recommendedState(m.catalog)
	if !engine.Equal(m.draft, want) {
		t.Fatalf("draft differs from recommendations:\n%s\n%s", engine.JSON(m.draft), engine.JSON(want))
	}
	if m.plan == nil || m.planErr != nil {
		t.Fatalf("review did not validate: %v", m.planErr)
	}
	text := screenText(m)
	for _, want := range []string{"Review", "Tokyo Night", "Save and download", "Downloads"} {
		if !strings.Contains(text, want) {
			t.Fatalf("review missing %q:\n%s", want, text)
		}
	}
}

func TestEditorFormChangesValues(t *testing.T) {
	m := newModel(t)
	press(m, "enter") // Start setup.
	if m.screen != stepScreen || !sections[m.sec].form {
		t.Fatal("setup does not start with the Editor form")
	}
	press(m, "down", "right") // Relative numbers: off -> on.
	if !strings.Contains(screenText(m), "changed · recommended: off") {
		t.Fatalf("changed row not marked:\n%s", screenText(m))
	}
	press(m, "enter")
	if m.draft.Values["options.relativenumber"] != true || m.draft.Values["options.number"] != true {
		t.Fatalf("form not applied: %v", m.draft.Values)
	}
	if sections[m.sec].id != "appearance" {
		t.Fatal("form did not advance to Appearance")
	}
}

func TestBackKeepsAnswers(t *testing.T) {
	m := newModel(t)
	press(m, "enter", "enter") // Start, accept the Editor form.
	press(m, "2")              // Habamax.
	press(m, "esc")
	if m.q != 0 || m.cursor != 1 {
		t.Fatalf("back did not return to the answered color scheme (q=%d cursor=%d)", m.q, m.cursor)
	}
	press(m, "esc")
	if !sections[m.sec].form {
		t.Fatal("back from the first question did not reach the previous section")
	}
}

func TestNoLanguagesSkipsLanguageDecisions(t *testing.T) {
	m := newModel(t)
	press(m, "enter")
	for sections[m.sec].id != "languages" {
		press(m, "enter")
	}
	if id := sections[m.sec].questions[m.q].id; id != "tools.provider" {
		t.Fatalf("Languages should start with the Mason question, at %s", id)
	}
	press(m, "enter")                          // Mason.
	press(m, "1", "2", "3", "4", "6", "enter") // Untick the recommended languages; Lua is sixth.
	// Syntax highlighting still helps without language servers (Markdown, shell scripts).
	if id := sections[m.sec].questions[m.q].id; id != "syntax.engine" {
		t.Fatalf("expected the syntax step, at %s", id)
	}
	press(m, "enter")
	// Coding helpers stay; the debugger, which is for JavaScript and TypeScript, is skipped.
	for _, id := range []string{"coding.pairs", "coding.surround", "coding.textobjects", "coding.todo", "coding.lint"} {
		if got := sections[m.sec].questions[m.q].id; got != id {
			t.Fatalf("expected %s, at %s", id, got)
		}
		press(m, "enter")
	}
	if sections[m.sec].id != "tools" || sections[m.sec].questions[m.q].id != "git.hunks" {
		t.Fatalf("expected to skip to Git changes, at %s", sections[m.sec].questions[m.q].id)
	}
	for _, id := range []string{"formatting.provider", "completion.native"} {
		if _, ok := m.draft.Values[id]; ok {
			t.Fatalf("%s kept although no language is selected", id)
		}
	}
	// Mason stays chosen: the recommended Tree-sitter parsers still need its compiler.
	if !slices.Contains(engine.Plugins(m.draft), "mason") {
		t.Fatal("Mason dropped although Tree-sitter needs its tree-sitter CLI")
	}
	next := engine.Clone(m.draft)
	next.Values["syntax.engine"], next.Values["coding.lint"] = "native", "none"
	if slices.Contains(engine.Plugins(next), "mason") {
		t.Fatal("Mason installed although nothing needs a separate program")
	}
	if !strings.Contains(summary(m.draft, sections[5]), "skipped") {
		t.Fatal("formatting not reported as skipped")
	}
}

func TestTooSmallTerminalIgnoresInput(t *testing.T) {
	m := newModel(t)
	press(m, "enter")
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	before := engine.Clone(m.draft)
	press(m, "enter", "1", "2", "enter")
	if !engine.Equal(before, m.draft) || m.screen != stepScreen || m.busy {
		t.Fatal("hidden screen received input")
	}
	if !strings.Contains(screenText(m), "larger") {
		t.Fatal("no resize message")
	}
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if m.screen != stepScreen || m.busy {
		t.Fatal("state lost after resize")
	}
}

func TestCtrlCTwiceQuits(t *testing.T) {
	m := newModel(t)
	press(m, "ctrl+c")
	if m.quit || !strings.Contains(screenText(m), "Press Ctrl+C again") {
		t.Fatal("first Ctrl+C must only warn")
	}
	press(m, "ctrl+c")
	if !m.quit {
		t.Fatal("second Ctrl+C did not quit")
	}
}

func TestLayoutsFitTheTerminal(t *testing.T) {
	m := newModel(t)
	for _, size := range [][2]int{{120, 40}, {80, 24}, {60, 18}, {44, 14}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		for _, s := range []func(){func() { m.openWelcome() }, func() { m.enterSection(0, false) }, func() { m.enterSection(1, false) }} {
			s()
			lines := strings.Split(m.View().Content, "\n")
			if len(lines) != size[1] {
				t.Fatalf("%dx%d: %d lines", size[0], size[1], len(lines))
			}
			for _, line := range lines {
				if w := ansi.StringWidth(line); w > size[0] {
					t.Fatalf("%dx%d: line of width %d: %q", size[0], size[1], w, ansi.Strip(line))
				}
			}
		}
	}
}

func TestShortcutRowsFitTheTerminal(t *testing.T) {
	m := newModel(t)
	press(m, "enter", "enter") // Start, accept the Editor form.
	for m.question().id != "keymaps-editor" {
		press(m, "enter")
	}
	for _, width := range []int{92, 100, 120} {
		m.Update(tea.WindowSizeMsg{Width: width, Height: 30})
		for i, r := range m.shortcutRows() {
			if r.kind != "key" {
				continue
			}
			m.cursor = i
			marker := "❯ " + engine.KeyLabel(m.draft, r.key.Key)
			var row string
			for _, line := range strings.Split(m.View().Content, "\n") {
				if strings.Contains(ansi.Strip(line), marker) {
					row = line
				}
			}
			text := strings.TrimRight(ansi.Strip(row), " ")
			if row == "" {
				t.Fatalf("width %d: row for %s not shown", width, r.key.Label)
			}
			if w := ansi.StringWidth(row); w > width {
				t.Fatalf("width %d: row of width %d: %q", width, w, text)
			}
			if !strings.HasSuffix(text, " "+r.key.Source) {
				t.Fatalf("width %d: row does not end with %q: %q", width, r.key.Source, text)
			}
		}
	}
}

func TestDependenciesReportTheChosenNeovim(t *testing.T) {
	m := newModel(t)
	// PATH has a Neovim, but --nvim names one that does not exist.
	bin := t.TempDir()
	name := "nvim"
	if runtime.GOOS == "windows" {
		name = "nvim.bat"
	}
	if err := os.WriteFile(filepath.Join(bin, name), []byte("exit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	m.bin = filepath.Join(t.TempDir(), "missing", "nvim")
	nvim := func() engine.Dependency {
		for _, d := range m.dependencies() {
			if d.ID == "nvim" {
				return d
			}
		}
		t.Fatal("no Neovim row")
		return engine.Dependency{}
	}
	if d := nvim(); d.Status != "missing" || d.Path != "" {
		t.Fatalf("a --nvim that does not exist is reported as %+v", d)
	}
	// A plan names the Neovim it was checked with.
	m.plan = &engine.Plan{Neovim: "/opt/nvim/bin/nvim"}
	if d := nvim(); d.Status != "present" || d.Path != m.plan.Neovim {
		t.Fatalf("the Neovim a plan was checked with is reported as %+v", d)
	}
}

func TestSaveWithoutDownloadingWritesTarget(t *testing.T) {
	needNeovim(t)
	m := newModel(t)
	acceptAll(t, m)
	press(m, "2") // Save without downloading.
	if m.screen != doneScreen || m.resultErr != nil {
		t.Fatalf("save failed: %v\n%s", m.resultErr, screenText(m))
	}
	if !strings.Contains(screenText(m), "Your Neovim is ready") {
		t.Fatalf("missing success screen:\n%s", screenText(m))
	}
	if _, err := os.Stat(filepath.Join(m.target, "init.lua")); err != nil {
		t.Fatal(err)
	}
	reopened, err := New(m.target, "", true)
	if err != nil || reopened.ownership != "SAVED" || !strings.Contains(ansi.Strip(reopened.View().Content), "Review & change") {
		t.Fatalf("reopened target: %v %s", err, reopened.ownership)
	}
}

func TestPlainModeAsksTheSameQuestions(t *testing.T) {
	needNeovim(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	var out bytes.Buffer
	answers := strings.Repeat("\n", 90) // Accept every recommendation, then quit at the review.
	err := RunPlain(filepath.Join(t.TempDir(), "nvim"), "", strings.NewReader(answers+"q\n"), &out)
	if err != nil {
		t.Fatal(err, out.String())
	}
	text := out.String()
	for _, want := range []string{"── 1/8 Editor", "Color scheme", "(recommended)", "── Review ──", "Quit without saving"} {
		if !strings.Contains(text, want) {
			t.Fatalf("plain output missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "\x1b") {
		t.Fatal("escape sequences in plain mode")
	}
}

func TestPlainModeRefusesToLoseAnswersSilently(t *testing.T) {
	var out bytes.Buffer
	err := RunPlain(filepath.Join(t.TempDir(), "nvim"), "", strings.NewReader("\n\n"), &out)
	if err == nil || !strings.Contains(err.Error(), "no configuration was written") {
		t.Fatalf("expected unsaved-input error, got %v", err)
	}
}

func TestLaunchCommandMatchesWherePluginsAre(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("NVIM_APPNAME", "")
	if err := os.MkdirAll(filepath.Join(root, "config"), 0700); err != nil {
		t.Fatal(err)
	}
	for target, want := range map[string]string{
		filepath.Join(root, "config", "nvim"):    "nvim",
		filepath.Join(root, "config", "shoshin"): withAppName(runtime.GOOS, "shoshin", "nvim"),
		filepath.Join(root, "elsewhere", "nvim"): "nvim -u ",
	} {
		if got := launchCommand(target); !strings.HasPrefix(got, want) {
			t.Fatalf("%s: %q, want %q", target, got, want)
		}
	}
}

func TestReviewKnowsWhatIsAlreadyInstalled(t *testing.T) {
	needNeovim(t)
	m := newModel(t)
	acceptAll(t, m)
	for _, a := range m.plan.Actions {
		marker := filepath.Join(a.Destination, ".git")
		switch a.Kind {
		case "mason":
			marker = filepath.Join(a.Destination, "packages", a.Package)
		case "parsers":
			for _, lang := range a.Parsers {
				if err := os.MkdirAll(filepath.Join(a.Destination, "parser"), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(a.Destination, "parser", lang+".so"), nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			continue
		}
		if err := os.MkdirAll(marker, 0700); err != nil {
			t.Fatal(err)
		}
	}
	run(m, m.openReview())
	if m.menu[0].label != "Save" || !m.menu[0].recommended {
		t.Fatalf("expected a plain Save once everything is installed, got %q", m.menu[0].label)
	}
	if text := screenText(m); !strings.Contains(text, fmt.Sprintf("all %d installed", len(m.plan.Actions))) {
		t.Fatalf("review does not report installed downloads:\n%s", text)
	}
}

func TestDoneScreenWrapsAtEightyColumns(t *testing.T) {
	needNeovim(t)
	m := newModel(t)
	acceptAll(t, m)
	press(m, "2") // Save without downloading.
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	text := screenText(m)
	if !strings.Contains(text, "the guide.") || !strings.Contains(text, "init.lua") {
		t.Fatalf("hint or launch command cut off:\n%s", text)
	}
	if m.dirty() || m.ownership != "SAVED" || !strings.Contains(text, "saved") {
		t.Fatal("a save without downloads must count as saved")
	}
	press(m, "enter") // Quit must not ask about unsaved choices.
	if !m.quit {
		t.Fatal("quitting after a save asked about unsaved choices")
	}
}

func TestWrappedLinesKeepTheirColor(t *testing.T) {
	th := newTheme(true, false)
	text := "Each language server needs a recipe: the program to run, its file types and how to find your project."
	lines := wrap(th.Dim(text), 30)
	if len(lines) < 3 {
		t.Fatalf("expected the text to wrap, got %d lines", len(lines))
	}
	var words []string
	for i, line := range lines {
		if !strings.HasPrefix(line, "\x1b[") || !strings.HasSuffix(line, "\x1b[m") {
			t.Fatalf("line %d does not set and reset its own color: %q", i+1, line)
		}
		words = append(words, strings.Fields(ansi.Strip(line))...)
	}
	if strings.Join(words, " ") != text {
		t.Fatalf("wrapping changed the text: %q", strings.Join(words, " "))
	}
}

func TestAppNameCommandUsesThePlatformShell(t *testing.T) {
	if got := withAppName("darwin", "work", "nvim"); got != "NVIM_APPNAME=work nvim" {
		t.Fatal(got)
	}
	if got := withAppName("windows", "work", "nvim"); got != "$env:NVIM_APPNAME='work'; nvim" {
		t.Fatal(got)
	}
	if got := withAppName("windows", "it's", "nvim"); got != "$env:NVIM_APPNAME='it''s'; nvim" {
		t.Fatal(got)
	}
}

func TestCustomThemeFromGitHub(t *testing.T) {
	rev := strings.Repeat("a", 40)
	lookupTheme = func(_ context.Context, repo string) (engine.ThemeLookup, error) {
		return engine.ThemeLookup{Repository: repo, Revision: rev, Colorschemes: []string{"dusk", "dusk-light"}, Stars: 42}, nil
	}
	defer func() { lookupTheme = engine.LookupTheme }()
	m := newModel(t)
	press(m, "enter", "enter") // Start, accept the Editor form.
	m.cursor = len(m.question().options) - 1
	if m.question().options[m.cursor].short != "custom theme" {
		t.Fatal("last color scheme option is not the custom theme")
	}
	press(m, "enter")
	if !m.inputQuestion() {
		t.Fatalf("expected the repository input, at %s", m.question().id)
	}
	for _, r := range "someone/dusk.nvim" {
		m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	press(m, "enter")
	if q := m.question(); q.id != "custom-theme-colorscheme" || len(q.options) != 2 {
		t.Fatalf("expected the colorscheme choice, at %s", q.id)
	}
	press(m, "2")
	for id, want := range map[string]any{"appearance.colorscheme": engine.CustomTheme, "appearance.custom_theme.repository": "someone/dusk.nvim",
		"appearance.custom_theme.revision": rev, "appearance.custom_theme.colorscheme": "dusk-light"} {
		if m.draft.Values[id] != want {
			t.Fatalf("%s = %v, want %v", id, m.draft.Values[id], want)
		}
	}
	if err := m.catalog.Validate(m.draft); err != nil {
		t.Fatal(err)
	}
	uses, _ := engine.PluginsInUse(m.draft)
	found := false
	for _, u := range uses {
		if u.Repository == "someone/dusk.nvim" {
			found = !u.Listed
		}
	}
	if !found {
		t.Fatal("custom theme is not listed as an unreviewed plugin")
	}
	// Choosing a listed theme afterwards removes the custom details.
	next, err := choose(m.catalog, m.draft, themeOptions(m.draft)[0])
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := next.Values["appearance.custom_theme.repository"]; ok || m.catalog.Validate(next) != nil {
		t.Fatal("custom theme details kept after choosing a listed theme")
	}
}

func keyFor(keys []engine.EffectiveKeymap, action string) (engine.EffectiveKeymap, bool) {
	for _, k := range keys {
		if k.Action == action {
			return k, true
		}
	}
	return engine.EffectiveKeymap{}, false
}

func TestShortcutEditor(t *testing.T) {
	m := newModel(t)
	press(m, "enter", "enter") // Start, accept the Editor form.
	for m.question().id != "keymaps-editor" {
		press(m, "enter")
	}
	if !strings.Contains(screenText(m), "kickstart.nvim") || !strings.Contains(screenText(m), "Keep these shortcuts") {
		t.Fatalf("shortcut list missing its source:\n%s", screenText(m))
	}
	files, ok := keyFor(engine.EffectiveKeymaps(m.draft), "find.files")
	if !ok || files.Key != "<leader>sf" || files.Source != "kickstart.nvim" {
		t.Fatalf("kickstart find files: %+v", files)
	}
	// Move Find files to <leader>ff.
	rows := m.shortcutRows()
	for i, r := range rows {
		if r.kind == "key" && r.key.Action == "find.files" {
			m.cursor = i
		}
	}
	press(m, "enter")
	for _, r := range "<leader>ff" {
		m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	press(m, "enter")
	if k, _ := keyFor(engine.EffectiveKeymaps(m.draft), "find.files"); k.Key != "<leader>ff" || k.Source != "yours" {
		t.Fatalf("rebind failed: %+v", k)
	}
	// Add a command shortcut.
	press(m, "a")
	m.cursor = len(actionOptions()) - 1
	press(m, "enter")
	for _, r := range "Lazy" {
		m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	press(m, "enter")
	for _, r := range "<leader>L" {
		m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	press(m, "enter")
	if k, ok := keyFor(engine.EffectiveKeymaps(m.draft), "command:Lazy"); !ok || k.Key != "<leader>L" {
		t.Fatalf("command shortcut not added: %+v", m.draft.Keymaps)
	}
	if err := m.catalog.Validate(m.draft); err != nil {
		t.Fatal(err)
	}
	files2, _ := engine.Generate(m.draft)
	lua := files2["lua/user_config/keymaps.lua"]
	for _, want := range []string{`map("n", "<leader>sf"`, `"<leader>ff", "<cmd>ShoshinFiles<CR>"`, `"<leader>L", "<cmd>Lazy<CR>"`, "-- Your shortcuts", "-- From kickstart.nvim"} {
		if want == `map("n", "<leader>sf"` {
			if strings.Contains(lua, want) {
				t.Fatal("moved shortcut still generated on its old key")
			}
			continue
		}
		if !strings.Contains(lua, want) {
			t.Fatalf("keymaps.lua missing %q:\n%s", want, lua)
		}
	}
	press(m, "R")
	if len(m.draft.Keymaps) != 0 {
		t.Fatal("restore did not remove changes")
	}
}

func TestShortcutStylesAreCredited(t *testing.T) {
	for _, style := range []string{"kickstart", "lazyvim"} {
		s := engine.NewState()
		s.Values["keymaps.style"], s.Values["git.hunks"] = style, "gitsigns"
		keys := engine.EffectiveKeymaps(s)
		if len(keys) < 30 {
			t.Fatalf("%s: only %d shortcuts", style, len(keys))
		}
		for _, k := range keys {
			if k.Source == "" || k.Label == "" {
				t.Fatalf("%s: uncredited shortcut %+v", style, k)
			}
		}
	}
	// Git shortcuts disappear when Git changes are off.
	s := engine.NewState()
	s.Values["keymaps.style"], s.Values["git.hunks"] = "kickstart", "native"
	if _, ok := keyFor(engine.EffectiveKeymaps(s), "git.next_hunk"); ok {
		t.Fatal("git shortcut kept without a Git plugin")
	}
	// LazyVim's gr is nowait, so it is not reported as a conflict; a user "gr" would be.
	s.Values["keymaps.style"] = "lazyvim"
	if c := engine.KeymapConflicts(s); len(c) != 0 {
		t.Fatalf("unexpected conflicts: %v", c)
	}
	s.Values["keymaps.style"] = "kickstart"
	s.Keymaps = []engine.Keymap{{Mode: "n", Key: "gr", Action: "code.references"}}
	if c := engine.KeymapConflicts(s); len(c) == 0 {
		t.Fatal("gr before grd was not reported")
	}
}

// Steps name the keys the chosen style really has, spelled with the chosen leader.
func TestStepsNameConfiguredKeys(t *testing.T) {
	s := engine.NewState()
	for style, want := range map[string]string{"kickstart": `Your shortcuts: \ file explorer`, "lazyvim": "Your shortcuts: Space e file explorer", "none": ""} {
		s.Values["keymaps.style"] = style
		if got := shortcutHint(s, []string{"explorer.toggle"}); got != want {
			t.Fatalf("%s: got %q, want %q", style, got, want)
		}
	}
	s.Values["keymaps.style"], s.Values["keymaps.leader"] = "kickstart", "comma"
	if got := shortcutHint(s, []string{"find.files", "code.format"}); got != "Your shortcuts: Comma sf find files · Comma f format file" {
		t.Fatalf("got %q", got)
	}
	if got := engine.KeyLabel(s, "<Esc><Esc>"); got != "Esc Esc" {
		t.Fatalf("got %q", got)
	}
	if got := engine.KeyLabel(s, "<C-S-x>"); got != "Ctrl+Shift+x" {
		t.Fatalf("got %q", got)
	}
}

// Mason downloads servers and formatters, never runtimes or tools it has no package for.
func TestMasonProvidesOnlyItsPackages(t *testing.T) {
	s := engine.NewState()
	s.Values["tools.provider"], s.Values["formatting.provider"] = "mason", "conform"
	for _, id := range []string{"java", "swift", "python", "rust"} {
		s.Values["language."+id+".lsp"] = true
	}
	got := engine.MasonProvides(s)
	for _, want := range []string{"jdtls", "pyright-langserver", "ruff", "rust-analyzer"} {
		if !got[want] {
			t.Errorf("Mason should provide %s", want)
		}
	}
	for _, not := range []string{"java", "sourcekit-lsp", "rustfmt", "cargo"} {
		if got[not] {
			t.Errorf("Mason cannot provide %s", not)
		}
	}
	s.Values["tools.provider"] = "system"
	if len(engine.MasonProvides(s)) != 0 {
		t.Error("nothing is provided without Mason")
	}
}
