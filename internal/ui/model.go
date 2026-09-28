package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/shoshindev/shoshinvim/internal/engine"
)

type screen int

const (
	welcomeScreen screen = iota
	stepScreen
	reviewScreen
	pickScreen
	workingScreen
	doneScreen
	recoveryScreen
	restoreScreen
)

const (
	minWidth, minHeight = 44, 14
	sidebarWidth        = 16
	sidebarMinWidth     = 92
)

type menuItem struct {
	label, detail, action string
	recommended           bool
}

type logLine struct{ key, kind, text string }

type progressMsg struct {
	id   int
	text string
	next <-chan tea.Msg
}

type jobMsg struct {
	id     int
	kind   string
	plan   *engine.Plan
	result *engine.ApplyResult
	retry  []engine.ActionResult
	err    error
}

type command struct{ label, action string }

var paletteCommands = []command{
	{"Preview generated Lua", "lua"},
	{"See file changes", "diff"},
	{"Check dependencies", "health"},
	{"Show all settings as JSON", "json"},
	{"Set any setting…", "set"},
	{"Go to review", "review"},
	{"Retry failed downloads", "retry"},
	{"Recovery & history", "recovery"},
	{"Keys & help", "help"},
	{"Quit", "quit"},
}

type Model struct {
	target, bin      string
	catalog          engine.Catalog
	th               theme
	noColor          bool
	ownership        string
	problems         []string
	draft, persisted engine.State
	width, height    int

	screen         screen
	sec, q, cursor int
	form           []int
	checks         []bool
	visited        map[int]bool
	returnToReview bool
	menu           []menuItem

	plan    *engine.Plan
	planErr error

	busy, writing bool
	jobID         int
	jobTitle      string
	activity      string
	cancel        context.CancelFunc
	spin          spinner.Model
	log           []logLine
	wrote         int
	actionsDone   int
	actionsTotal  int
	names         map[string]string
	doneKind      string
	result        *engine.ApplyResult
	resultErr     error

	journals  []engine.Journal
	restoreID string

	viewerOpen  bool
	viewerTitle string
	viewer      viewport.Model

	paletteOpen, settingOpen bool
	input                    textinput.Model

	notice    string
	noticeBad bool
	quitArmed bool
	quit      bool
}

func New(target, bin string, noColor bool) (*Model, error) {
	ins, err := engine.Inspect(target)
	if err != nil {
		return nil, err
	}
	s := engine.NewState()
	if ins.State != nil {
		s = engine.Clone(*ins.State)
	}
	input := textinput.New()
	input.Prompt = "› "
	input.CharLimit = 8192
	m := &Model{target: ins.Target, bin: bin, catalog: engine.LoadCatalog(), noColor: noColor,
		ownership: ins.Ownership, problems: ins.Problems, draft: s, persisted: engine.Clone(s),
		width: 100, height: 30, visited: map[int]bool{}, input: input,
		viewer: viewport.New(viewport.WithWidth(80), viewport.WithHeight(20)),
		spin:   spinner.New(spinner.WithSpinner(spinner.MiniDot))}
	m.setTheme(true)
	if ins.Ownership == "SAVED" {
		for i := range sections {
			m.visited[i] = true
		}
	}
	m.openWelcome()
	return m, nil
}

func (m *Model) setTheme(dark bool) {
	m.th = newTheme(dark, m.noColor)
	if !m.noColor {
		m.spin.Style = lipgloss.NewStyle().Foreground(m.th.accent)
	}
}

func (m *Model) Init() tea.Cmd { return tea.RequestBackgroundColor }

func (m *Model) dirty() bool { return !engine.Equal(m.draft, m.persisted) }

func (m *Model) tooSmall() bool { return m.width < minWidth || m.height < minHeight }

func (m *Model) flash(text string, bad bool) { m.notice, m.noticeBad = text, bad }

func (m *Model) change(next engine.State) {
	m.draft = next
	m.plan = nil
}

// ---- navigation -------------------------------------------------------------

func (m *Model) openWelcome() {
	m.screen = welcomeScreen
	m.cursor = 0
	switch m.ownership {
	case "SAVED":
		m.menu = []menuItem{
			{"Review & change", "See your setup, adjust any section, then save", "review", true},
			{"Walk through every section", "Revisit each decision from the start", "walk", false},
			{"Quit", "", "quit", false},
		}
	case "READ ONLY":
		m.menu = []menuItem{
			{"Explore anyway", "Walk through the setup and preview the Lua; saving stays disabled", "walk", false},
			{"Quit", "", "quit", true},
		}
	case "RECOVERY":
		m.menu = []menuItem{
			{"Open recovery", "Restore a previous version of your files", "recovery", true},
			{"Quit", "", "quit", false},
		}
	default:
		m.menu = []menuItem{
			{"Start setup", "Seven short sections, one decision at a time", "walk", true},
			{"Quit", "", "quit", false},
		}
	}
}

func (m *Model) enterSection(i int, fromEnd bool) tea.Cmd {
	for i >= 0 && i < len(sections) && sections[i].skipped(m.draft) {
		if fromEnd {
			i--
		} else {
			i++
		}
	}
	if i < 0 {
		m.openWelcome()
		return nil
	}
	if i >= len(sections) {
		return m.openReview()
	}
	m.screen, m.sec = stepScreen, i
	sec := sections[i]
	if sec.form {
		m.form = make([]int, len(sec.questions))
		for j, q := range sec.questions {
			m.form[j] = currentIndex(m.draft, q)
		}
		m.cursor = 0
		return nil
	}
	if fromEnd {
		m.q = len(sec.questions)
		return m.moveQuestion(-1)
	}
	m.q = -1
	return m.moveQuestion(1)
}

func (m *Model) moveQuestion(dir int) tea.Cmd {
	sec := sections[m.sec]
	for j := m.q + dir; j >= 0 && j < len(sec.questions); j += dir {
		if !sec.questions[j].skipped(m.draft) {
			m.q = j
			m.loadQuestion()
			return nil
		}
	}
	if dir > 0 {
		return m.finishSection()
	}
	return m.enterSection(m.sec-1, true)
}

func (m *Model) loadQuestion() {
	q := sections[m.sec].questions[m.q]
	if q.multi {
		m.checks = checked(m.draft, q)
		m.cursor = 0
		return
	}
	if m.cursor = currentIndex(m.draft, q); m.cursor < 0 {
		m.cursor = recommendedIndex(q)
	}
}

func (m *Model) finishSection() tea.Cmd {
	m.visited[m.sec] = true
	if m.returnToReview {
		return m.openReview()
	}
	return m.enterSection(m.sec+1, false)
}

func (m *Model) answer() tea.Cmd {
	q := sections[m.sec].questions[m.q]
	var next engine.State
	var err error
	if q.multi {
		next, err = chooseMany(m.catalog, m.draft, q, m.checks)
	} else {
		next, err = choose(m.catalog, m.draft, q.options[m.cursor])
	}
	if err != nil {
		m.flash(err.Error(), true)
		return nil
	}
	m.change(next)
	return m.moveQuestion(1)
}

// applyForm stores the rows of the current form section in the draft.
func (m *Model) applyForm() bool {
	next := m.draft
	for j, q := range sections[m.sec].questions {
		if m.form[j] < 0 {
			continue
		}
		var err error
		if next, err = choose(m.catalog, next, q.options[m.form[j]]); err != nil {
			m.flash(err.Error(), true)
			return false
		}
	}
	m.change(next)
	return true
}

func (m *Model) openReview() tea.Cmd {
	m.screen, m.returnToReview, m.cursor = reviewScreen, false, 0
	m.plan, m.planErr, m.menu = nil, nil, nil
	if m.ownership == "READ ONLY" {
		m.planErr = fmt.Errorf("saving is disabled: this folder holds a configuration ShoshinVim did not create")
		m.menu = m.reviewMenu()
		return nil
	}
	state, target, bin := engine.Clone(m.draft), m.target, m.bin
	return m.work("review", false, func(ctx context.Context, _ func(string)) jobMsg {
		p, err := engine.BuildPlan(ctx, target, state, bin, false)
		if err != nil {
			return jobMsg{err: err}
		}
		return jobMsg{plan: &p}
	})
}

func (m *Model) fileChanges() (created, updated, removed int) {
	if m.plan == nil {
		return
	}
	for _, f := range m.plan.Files {
		switch {
		case f.Before == f.After:
		case f.Delete:
			removed++
		case f.Before == "":
			created++
		default:
			updated++
		}
	}
	return
}

func (m *Model) reviewMenu() []menuItem {
	var items []menuItem
	created, updated, removed := m.fileChanges()
	changes := created + updated + removed
	if m.plan != nil {
		files := plural(changes, "file", "files")
		plugins, tools := actionCounts(m.plan.Actions)
		downloads := downloadText(plugins, tools)
		switch {
		case changes > 0 && downloads != "":
			items = append(items,
				menuItem{"Save and download", fmt.Sprintf("Write %d %s, then download %s", changes, files, downloads), "save-download", true},
				menuItem{"Save without downloading", "Missing plugins and tools stay pending; save again to download", "save", false})
		case changes > 0:
			items = append(items, menuItem{"Save", fmt.Sprintf("Write %d %s", changes, files), "save", true})
		case downloads != "":
			items = append(items, menuItem{"Download plugins and tools", "Nothing to save; fetch anything missing (" + downloads + ")", "download", true})
		}
	}
	items = append(items, menuItem{"Preview the generated Lua", "Every file ShoshinVim would write", "lua", false})
	if changes > 0 {
		items = append(items, menuItem{"See file changes", "A diff against what is on disk", "diff", false})
	}
	items = append(items, menuItem{"Change a section", "Jump to any section, then come back here", "sections", false}, menuItem{"Quit", "", "quit", false})
	return items
}

func (m *Model) openPick() {
	m.screen, m.cursor, m.menu = pickScreen, 0, nil
	for i, sec := range sections {
		if !sec.skipped(m.draft) {
			m.menu = append(m.menu, menuItem{label: sec.title, detail: summary(m.draft, sec), action: "section:" + strconv.Itoa(i)})
		}
	}
	m.menu = append(m.menu, menuItem{label: "Back to review", action: "back-review"})
}

func (m *Model) openRecovery() {
	js, err := engine.Journals(m.target)
	if err != nil {
		m.flash(err.Error(), true)
		return
	}
	m.journals, m.screen, m.cursor, m.menu = js, recoveryScreen, 0, nil
	for _, j := range js {
		files := 0
		for _, f := range j.Plan.Files {
			if f.Before != f.After {
				files++
			}
		}
		m.menu = append(m.menu, menuItem{label: journalTime(j.ID) + " · " + j.Status, detail: fmt.Sprintf("%d %s changed · transaction %s", files, plural(files, "file", "files"), j.ID), action: "journal:" + j.ID})
	}
	m.menu = append(m.menu, menuItem{label: "Back", action: "welcome"})
}

func (m *Model) openRestore(id string) {
	m.restoreID, m.screen = id, restoreScreen
	m.menu = []menuItem{{"Restore the files from before this save", "Puts back the backups recorded for this save", "restore", false}}
	for _, j := range m.journals {
		if j.ID == id && j.Status == "committed" {
			m.menu = append(m.menu, menuItem{"Restore the files this save wrote", "Your manual edits are archived in .shoshinvim/recovery first", "restore-generated", false})
		}
	}
	m.menu = append(m.menu, menuItem{"Cancel", "", "recovery", true})
	m.cursor = len(m.menu) - 1 // Never preselect a destructive choice.
}

func (m *Model) activate(action string) tea.Cmd {
	switch {
	case action == "walk":
		return m.enterSection(0, false)
	case action == "review":
		return m.openReview()
	case action == "back-review":
		m.screen, m.menu, m.cursor = reviewScreen, m.reviewMenu(), 0
	case action == "recovery":
		m.openRecovery()
	case action == "welcome":
		m.openWelcome()
	case action == "quit":
		return m.requestQuit()
	case action == "save", action == "save-download", action == "download":
		return m.save(action != "save")
	case action == "lua":
		m.showLua()
	case action == "diff":
		m.showDiff()
	case action == "sections":
		m.openPick()
	case action == "retry":
		return m.retry()
	case action == "restore", action == "restore-generated":
		return m.restore(action == "restore-generated")
	case strings.HasPrefix(action, "section:"):
		i, _ := strconv.Atoi(strings.TrimPrefix(action, "section:"))
		m.returnToReview = true
		return m.enterSection(i, false)
	case strings.HasPrefix(action, "journal:"):
		m.openRestore(strings.TrimPrefix(action, "journal:"))
	}
	return nil
}

func (m *Model) requestQuit() tea.Cmd {
	if m.quitArmed || !m.dirty() || m.ownership == "READ ONLY" {
		m.quit = true
		return tea.Quit
	}
	m.quitArmed = true
	m.flash("Your choices are not saved. Choose Quit again or press Ctrl+C to leave.", true)
	return nil
}

// ---- input -----------------------------------------------------------------------

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		m.setTheme(msg.IsDark())
		return m, nil
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.viewer.SetWidth(max(10, m.width-4))
		m.viewer.SetHeight(max(3, m.height-5))
		m.input.SetWidth(max(10, m.width-12))
		return m, nil
	case spinner.TickMsg:
		if !m.busy {
			return m, nil
		}
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	case progressMsg:
		if msg.id == m.jobID {
			m.onProgress(msg.text)
		}
		return m, func() tea.Msg { return <-msg.next }
	case jobMsg:
		if msg.id == m.jobID {
			m.finishJob(msg)
		}
		return m, nil
	case tea.KeyPressMsg:
		return m, m.key(msg)
	}
	if m.paletteOpen || m.settingOpen {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *Model) key(msg tea.KeyPressMsg) tea.Cmd {
	key := msg.String()
	if key == "ctrl+c" {
		return m.ctrlC()
	}
	m.quitArmed, m.notice = false, ""
	if m.tooSmall() {
		return nil // Hidden prompts never receive input.
	}
	switch {
	case m.viewerOpen:
		if key == "esc" || key == "q" {
			m.viewerOpen = false
			return nil
		}
		var cmd tea.Cmd
		m.viewer, cmd = m.viewer.Update(msg)
		return cmd
	case m.settingOpen:
		return m.settingKey(msg)
	case m.paletteOpen:
		return m.paletteKey(msg)
	case m.busy:
		if key == "esc" && !m.writing && m.screen == reviewScreen {
			m.cancelReview()
			return m.enterSection(len(sections)-1, true)
		}
		return nil
	}
	switch key {
	case "/":
		m.paletteOpen, m.cursor = true, 0
		m.input.SetValue("")
		m.input.Placeholder = "Search commands"
		return m.input.Focus()
	case "?":
		m.openViewer("Keys & help", helpText)
		return nil
	}
	if m.screen == stepScreen {
		if sections[m.sec].form {
			return m.formKey(key)
		}
		return m.questionKey(key)
	}
	return m.menuKey(key)
}

func (m *Model) ctrlC() tea.Cmd {
	switch {
	case m.busy && m.writing:
		m.flash("Saving can't be interrupted. It is journaled and recoverable.", false)
		return nil
	case m.busy:
		if m.screen == reviewScreen {
			m.cancelReview()
			return nil
		}
		if m.cancel != nil {
			m.cancel()
		}
		m.flash("Cancelling the remaining downloads…", false)
		return nil
	case m.viewerOpen || m.paletteOpen || m.settingOpen:
		m.viewerOpen, m.paletteOpen, m.settingOpen = false, false, false
		m.input.Blur()
		return nil
	case m.quitArmed:
		m.quit = true
		return tea.Quit
	}
	m.quitArmed = true
	text := "Press Ctrl+C again to quit"
	if m.dirty() && m.ownership != "READ ONLY" {
		text += " · your choices are not saved"
	}
	m.flash(text, true)
	return nil
}

func (m *Model) cancelReview() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	m.jobID++ // Ignore the cancelled result.
	m.busy = false
	m.planErr = fmt.Errorf("check cancelled")
	m.menu = m.reviewMenu()
}

func digit(key string) int {
	if len(key) == 1 && key[0] >= '1' && key[0] <= '9' {
		return int(key[0] - '0')
	}
	return 0
}

func (m *Model) questionKey(key string) tea.Cmd {
	q := sections[m.sec].questions[m.q]
	n := len(q.options)
	switch key {
	case "up", "k", "shift+tab":
		m.cursor = max(0, m.cursor-1)
	case "down", "j", "tab":
		m.cursor = min(n-1, m.cursor+1)
	case "space", " ":
		if q.multi {
			m.checks[m.cursor] = !m.checks[m.cursor]
		}
	case "enter":
		return m.answer()
	case "esc", "left", "h":
		return m.moveQuestion(-1)
	default:
		if d := digit(key); d > 0 && d <= n {
			m.cursor = d - 1
			if q.multi {
				m.checks[m.cursor] = !m.checks[m.cursor]
				return nil
			}
			return m.answer()
		}
	}
	return nil
}

func (m *Model) formKey(key string) tea.Cmd {
	qs := sections[m.sec].questions
	q := qs[m.cursor]
	cycle := func(dir int) {
		n := len(q.options)
		if m.form[m.cursor] < 0 {
			m.form[m.cursor] = 0
			return
		}
		m.form[m.cursor] = (m.form[m.cursor] + dir + n) % n
	}
	switch key {
	case "up", "k", "shift+tab":
		m.cursor = max(0, m.cursor-1)
	case "down", "j", "tab":
		m.cursor = min(len(qs)-1, m.cursor+1)
	case "left", "h":
		cycle(-1)
	case "right", "l", "space", " ":
		cycle(1)
	case "r":
		m.form[m.cursor] = recommendedIndex(q)
	case "enter":
		if m.applyForm() {
			return m.finishSection()
		}
	case "esc":
		if m.applyForm() {
			return m.enterSection(m.sec-1, true)
		}
	}
	return nil
}

func (m *Model) menuKey(key string) tea.Cmd {
	n := len(m.menu)
	switch key {
	case "up", "k", "shift+tab":
		m.cursor = max(0, m.cursor-1)
	case "down", "j", "tab":
		m.cursor = min(n-1, m.cursor+1)
	case "enter":
		if n > 0 {
			return m.activate(m.menu[m.cursor].action)
		}
	case "esc", "left", "h":
		switch m.screen {
		case reviewScreen:
			return m.enterSection(len(sections)-1, true)
		case pickScreen:
			return m.activate("back-review")
		case recoveryScreen:
			m.openWelcome()
		case restoreScreen:
			m.openRecovery()
		}
	default:
		if d := digit(key); d > 0 && d <= n {
			m.cursor = d - 1
			return m.activate(m.menu[m.cursor].action)
		}
	}
	return nil
}

func (m *Model) filteredCommands() []command {
	query := strings.ToLower(strings.TrimSpace(m.input.Value()))
	var out []command
	for _, c := range paletteCommands {
		if strings.Contains(strings.ToLower(c.label), query) {
			out = append(out, c)
		}
	}
	return out
}

func (m *Model) paletteKey(msg tea.KeyPressMsg) tea.Cmd {
	options := m.filteredCommands()
	switch msg.String() {
	case "esc":
		m.paletteOpen = false
		m.input.Blur()
		return nil
	case "up", "shift+tab":
		m.cursor = max(0, m.cursor-1)
		return nil
	case "down", "tab":
		m.cursor = min(max(0, len(options)-1), m.cursor+1)
		return nil
	case "enter":
		if len(options) == 0 {
			return nil
		}
		m.paletteOpen = false
		m.input.Blur()
		return m.runCommand(options[min(m.cursor, len(options)-1)].action)
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.cursor = 0
	return cmd
}

func (m *Model) runCommand(action string) tea.Cmd {
	cursor := m.cursor
	switch action {
	case "lua":
		m.showLua()
	case "diff":
		m.showDiff()
	case "health":
		m.showHealth()
	case "json":
		m.openViewer("All settings", string(engine.JSON(m.draft)))
	case "help":
		m.openViewer("Keys & help", helpText)
	case "set":
		m.settingOpen = true
		m.input.SetValue("")
		m.input.Placeholder = "options.colorcolumn 80"
		return m.input.Focus()
	case "recovery":
		m.openRecovery()
	case "retry":
		return m.retry()
	case "review":
		return m.openReview()
	case "quit":
		return m.requestQuit()
	}
	if m.screen == stepScreen || m.screen == welcomeScreen {
		m.cursor = cursor
	}
	return nil
}

func (m *Model) settingSuggestions() []engine.Setting {
	id, _, _ := strings.Cut(strings.TrimSpace(m.input.Value()), " ")
	var out []engine.Setting
	for _, s := range m.catalog.Search("") {
		if id == "" || strings.HasPrefix(s.ID, id) {
			out = append(out, s)
			if len(out) == 6 {
				break
			}
		}
	}
	return out
}

func (m *Model) settingKey(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.settingOpen = false
		m.input.Blur()
		return nil
	case "tab":
		if s := m.settingSuggestions(); len(s) > 0 {
			m.input.SetValue(s[0].ID + " ")
			m.input.CursorEnd()
		}
		return nil
	case "enter":
		id, text, _ := strings.Cut(strings.TrimSpace(m.input.Value()), " ")
		v, err := m.catalog.Parse(id, strings.TrimSpace(text))
		next := engine.Clone(m.draft)
		if err == nil {
			err = m.catalog.Set(&next, id, v, true)
		}
		if err == nil {
			err = derive(m.catalog, &next)
		}
		if err != nil {
			m.flash(err.Error(), true)
			return nil
		}
		m.change(next)
		m.settingOpen = false
		m.input.Blur()
		m.flash(fmt.Sprintf("Set %s to %v in your draft", id, v), false)
		if m.screen == stepScreen {
			m.enterSection(m.sec, false)
		}
		return nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return cmd
}

func Run(target, bin string, noColor bool) error {
	m, err := New(target, bin, noColor)
	if err != nil {
		return err
	}
	var opts []tea.ProgramOption
	if noColor {
		opts = append(opts, tea.WithColorProfile(colorprofile.ASCII))
	}
	_, err = tea.NewProgram(m, opts...).Run()
	return err
}
