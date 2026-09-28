package ui

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/shoshindev/shoshinvim/internal/engine"
)

func newModel(t *testing.T) *Model {
	t.Helper()
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

func TestEverySectionAndQuestionHasARecommendation(t *testing.T) {
	c := engine.LoadCatalog()
	for _, sec := range sections {
		for _, q := range sec.questions {
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

func TestRecommendedChoicesMatchTheFormerRichIDE(t *testing.T) {
	c := engine.LoadCatalog()
	s, err := recommendedState(c)
	if err != nil {
		t.Fatal(err)
	}
	if err = c.Validate(s); err != nil {
		t.Fatal(err)
	}
	rich, _ := c.Profile(engine.NewState(), "rich-ide")
	if !reflect.DeepEqual(engine.Plugins(s), engine.Plugins(rich)) {
		t.Fatalf("plugins %v, want %v", engine.Plugins(s), engine.Plugins(rich))
	}
	target := filepath.Join(t.TempDir(), "nvim")
	a, _ := engine.Actions(s, target)
	b, _ := engine.Actions(rich, target)
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("downloads differ:\n%v\n%v", a, b)
	}
	if s.Profile != "custom" {
		t.Fatalf("guided state must not claim a preset: %s", s.Profile)
	}
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
	press(m, "1", "2", "3", "4", "5", "enter") // Untick every language.
	if sections[m.sec].id != "tools" || sections[m.sec].questions[m.q].id != "git.hunks" {
		t.Fatalf("expected to skip to Git changes, at %s", sections[m.sec].questions[m.q].id)
	}
	for _, id := range []string{"formatting.provider", "tools.provider", "completion.native"} {
		if _, ok := m.draft.Values[id]; ok {
			t.Fatalf("%s kept although no language is selected", id)
		}
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
	answers := strings.Repeat("\n", 40) // Accept every recommendation, then quit at the review.
	err := RunPlain(filepath.Join(t.TempDir(), "nvim"), "", strings.NewReader(answers+"q\n"), &out)
	if err != nil {
		t.Fatal(err, out.String())
	}
	text := out.String()
	for _, want := range []string{"── 1/7 Editor", "Color scheme", "(recommended)", "── Review ──", "Quit without saving"} {
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
		filepath.Join(root, "config", "shoshin"): "NVIM_APPNAME=shoshin nvim",
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
		if a.Kind == "mason" {
			marker = filepath.Join(a.Destination, "packages", a.Package)
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
