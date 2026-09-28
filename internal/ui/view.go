package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/shoshindev/shoshinvim/internal/engine"
)

// ---- viewers -------------------------------------------------------------------

func (m *Model) openViewer(title, content string) {
	m.viewerOpen, m.viewerTitle = true, title
	m.viewer.SetContent(content)
	m.viewer.GotoTop()
}

func (m *Model) showLua() {
	files, err := engine.Generate(m.draft)
	if err != nil {
		m.flash(err.Error(), true)
		return
	}
	var b strings.Builder
	for _, name := range sortedKeys(files) {
		b.WriteString(m.th.Accent("── "+name) + "\n")
		for _, line := range strings.Split(strings.TrimRight(files[name], "\n"), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "--") {
				line = m.th.Dim(line)
			}
			b.WriteString(line + "\n")
		}
		b.WriteString("\n")
	}
	m.openViewer("Generated Lua", b.String())
}

func (m *Model) showDiff() {
	if m.plan == nil {
		m.flash("Open the review first; it prepares the file changes.", true)
		return
	}
	diff := engine.Diff(*m.plan)
	if diff == "" {
		m.openViewer("File changes", "No file changes.")
		return
	}
	var b strings.Builder
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"):
			line = m.th.Bold(line)
		case strings.HasPrefix(line, "+"):
			line = m.th.Ok(line)
		case strings.HasPrefix(line, "-"):
			line = m.th.Bad(line)
		case strings.HasPrefix(line, "@@"):
			line = m.th.Blue(line)
		}
		b.WriteString(line + "\n")
	}
	m.openViewer("File changes", b.String())
}

func (m *Model) showHealth() {
	var b strings.Builder
	for _, d := range m.dependencies() {
		if d.Status == "present" {
			fmt.Fprintf(&b, "%s %s  %s\n  %s\n\n", m.th.Ok("✓"), m.th.Bold(d.ID), m.th.Dim(d.Purpose), m.th.Dim(d.Path))
		} else {
			fmt.Fprintf(&b, "%s %s  %s\n", m.th.Warn("!"), m.th.Bold(d.ID), m.th.Dim(d.Purpose))
			if d.Instruction != "" {
				fmt.Fprintf(&b, "  %s\n", d.Instruction)
			}
			b.WriteString("\n")
		}
	}
	b.WriteString(m.th.Dim("Missing tools never block saving. Installing runtimes such as Node.js or ripgrep is up to you."))
	m.openViewer("Dependencies", b.String())
}

const helpText = `Moving through the setup
  ↑ ↓  or  j k      move
  enter             choose, or continue to the next step
  esc               go back one step
  1–9               pick an option by number
  space             toggle a checkbox
  ← →  or  h l      change a value in the Editor form
  r                 reset a form row to its recommended value

Anywhere
  /                 command palette
  ?                 this help
  ctrl+c            cancel work; press twice to quit

Nothing is written until you save from the review.
Saving backs up every file it changes, and downloads always
happen after your configuration has been saved safely.
Manual Lua belongs in lua/user_config/user.lua, which ShoshinVim never overwrites.`

// ---- rendering -------------------------------------------------------------------

func (m *Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	v.WindowTitle = "ShoshinVim"
	return v
}

func (m *Model) render() string {
	if m.tooSmall() {
		lines := []string{"", "  " + m.th.Accent("◯ ShoshinVim"), "",
			"  Make the terminal a little larger", fmt.Sprintf("  to continue (%d×%d, needs %d×%d).", m.width, m.height, minWidth, minHeight),
			"", "  " + m.th.Dim("Your draft is kept. Ctrl+C twice quits.")}
		if m.notice != "" {
			lines = append(lines, "", "  "+m.th.Warn(m.notice))
		}
		return clip(lines, m.width, m.height)
	}
	if m.viewerOpen {
		header := []string{"  " + m.th.Accent("◯ ") + m.th.Bold(m.viewerTitle) + m.th.Dim(fmt.Sprintf("  %d%%", int(m.viewer.ScrollPercent()*100))), ""}
		body := strings.Split(m.viewer.View(), "\n")
		for i := range body {
			body[i] = "  " + body[i]
		}
		footer := []string{"", "  " + m.th.Dim("↑↓ scroll · pgup/pgdn page · esc close")}
		return clip(append(append(header, fit(body, m.height-4)...), footer...), m.width, m.height)
	}
	header := m.renderHeader()
	if m.screen == welcomeScreen && !m.paletteOpen && !m.settingOpen {
		header = []string{""}
	}
	footer := m.renderFooter()
	body := m.renderBody(m.height - len(header) - len(footer))
	lines := append(header, fit(body, m.height-len(header)-len(footer))...)
	return clip(append(lines, footer...), m.width, m.height)
}

func (m *Model) statusTag() string {
	switch m.ownership {
	case "READ ONLY":
		return m.th.Warn("read-only")
	case "RECOVERY":
		return m.th.Bad("needs recovery")
	case "SAVED":
		if m.dirty() {
			return m.th.Warn("unsaved changes")
		}
		return m.th.Ok("saved")
	}
	return m.th.Blue("new")
}

func (m *Model) renderHeader() []string {
	left := "  " + m.th.Accent("◯") + " " + m.th.Gradient("ShoshinVim")
	tag := m.statusTag()
	right := m.th.Dim(shortPath(m.target)) + m.th.Dim(" · ") + tag
	room := m.width - lipgloss.Width(left) - 2
	if lipgloss.Width(right) > room-2 {
		right = tag
	}
	gap := max(1, room-lipgloss.Width(right))
	return []string{left + strings.Repeat(" ", gap) + right, ""}
}

func (m *Model) hints() string {
	switch {
	case m.paletteOpen:
		return "type to filter · ↑↓ move · enter run · esc close"
	case m.settingOpen:
		return "type “setting value” · tab complete · enter set · esc close"
	case m.busy && m.writing:
		return "saving · ctrl+c is disabled while files are written"
	case m.busy:
		return "ctrl+c cancel"
	case m.screen == stepScreen && sections[m.sec].form:
		return "↑↓ move · ←→ change · r recommended · enter continue · esc back · / commands"
	case m.screen == stepScreen && sections[m.sec].questions[m.q].multi:
		return "↑↓ move · space toggle · enter continue · esc back · / commands"
	case m.screen == stepScreen:
		return "↑↓ move · enter choose · esc back · / commands · ? help"
	case m.screen == welcomeScreen || m.screen == doneScreen:
		return "↑↓ move · enter choose · / commands · ? help"
	}
	return "↑↓ move · enter choose · esc back · / commands · ? help"
}

func (m *Model) renderFooter() []string {
	notice := ""
	if m.notice != "" {
		if m.noticeBad {
			notice = "  " + m.th.Warn("● "+m.notice)
		} else {
			notice = "  " + m.th.Blue("● "+m.notice)
		}
	}
	return []string{notice, "  " + m.th.Dim(m.hints())}
}

func (m *Model) renderBody(h int) []string {
	if m.paletteOpen || m.settingOpen {
		return m.renderOverlay(min(m.width-4, 72), h)
	}
	switch m.screen {
	case stepScreen, reviewScreen, pickScreen:
		if m.width >= sidebarMinWidth {
			cw := m.width - 2 - sidebarWidth - 5
			content := m.flowContent(cw, h)
			side := m.sidebar(h)
			out := make([]string, h)
			for i := range out {
				left := ""
				if i < len(side) {
					left = side[i]
				}
				right := ""
				if i < len(content) {
					right = content[i]
				}
				out[i] = "  " + padRight(left, sidebarWidth) + " " + m.th.Dim("│") + "  " + right
			}
			return out
		}
		progress := m.progressBar(m.width - 4)
		content := m.flowContent(m.width-4, h-len(progress))
		return indent(append(progress, content...))
	}
	cw := min(m.width-4, 76)
	var lines []string
	switch m.screen {
	case welcomeScreen:
		lines = m.welcomeContent(cw, h)
	case workingScreen:
		lines = m.workingContent(cw, h)
	case doneScreen:
		lines = m.doneContent(cw, h)
	case recoveryScreen, restoreScreen:
		lines = m.recoveryContent(cw, h)
	}
	return indent(lines)
}

func (m *Model) flowContent(cw, h int) []string {
	switch m.screen {
	case reviewScreen:
		return m.reviewContent(cw, h)
	case pickScreen:
		head := []string{m.th.Bold("Change a section"), m.th.Dim("You come back to the review afterwards."), ""}
		return append(head, m.menuLines(m.menu, cw, h-len(head))...)
	}
	if sections[m.sec].form {
		return m.formContent(cw, h)
	}
	return m.questionContent(cw, h)
}

func (m *Model) sidebar(h int) []string {
	var lines []string
	gap := h >= 2*(len(sections)+1)
	add := func(mark, label string) {
		lines = append(lines, mark+" "+label)
		if gap {
			lines = append(lines, "")
		}
	}
	for i, sec := range sections {
		switch {
		case m.screen == stepScreen && i == m.sec:
			add(m.th.Accent("●"), m.th.Accent(sec.title))
		case sec.skipped(m.draft):
			add(m.th.Dim("–"), m.th.Dim(sec.title))
		case m.visited[i]:
			add(m.th.Ok("✓"), sec.title)
		default:
			add(m.th.Dim("○"), m.th.Dim(sec.title))
		}
	}
	if m.screen == reviewScreen || m.screen == pickScreen {
		add(m.th.Accent("●"), m.th.Accent("Review"))
	} else {
		add(m.th.Dim("○"), m.th.Dim("Review"))
	}
	return lines
}

func (m *Model) progressBar(width int) []string {
	total := len(sections) + 1
	index := len(sections)
	label := m.th.Blue("Review")
	if m.screen == stepScreen {
		index, label = m.sec, m.sectionLine()
	}
	step := m.th.Dim(fmt.Sprintf("step %d of %d", index+1, total))
	label += strings.Repeat(" ", max(1, width-lipgloss.Width(label)-lipgloss.Width(step))) + step
	filled := width * (index + 1) / total
	bar := m.th.Accent(strings.Repeat("━", filled)) + m.th.Dim(strings.Repeat("─", max(0, width-filled)))
	return []string{label, bar, ""}
}

func (m *Model) questionPosition() string {
	sec := sections[m.sec]
	index, total := 0, 0
	for j, q := range sec.questions {
		if q.skipped(m.draft) {
			continue
		}
		total++
		if j <= m.q {
			index = total
		}
	}
	return fmt.Sprintf(" · %d of %d", index, total)
}

// sectionLine names the current section and position; compact layouts show it in the progress bar.
func (m *Model) sectionLine() string {
	sec := sections[m.sec]
	if sec.form {
		return m.th.Blue(sec.title) + m.th.Dim(" · text & editing")
	}
	return m.th.Blue(sec.title) + m.th.Dim(m.questionPosition())
}

func (m *Model) questionContent(cw, h int) []string {
	q := sections[m.sec].questions[m.q]
	var lines []string
	if m.width >= sidebarMinWidth {
		lines = append(lines, m.sectionLine(), "")
	}
	lines = append(lines, m.th.Bold(q.title))
	lines = append(lines, wrap(m.th.Dim(q.help), cw)...)
	if q.multi {
		var names []string
		for _, o := range q.options {
			if o.recommended {
				names = append(names, o.name())
			}
		}
		lines = append(lines, m.th.Dim("Recommended: "+strings.Join(names, ", ")))
	}
	lines = append(lines, "")
	return append(lines, m.optionLines(q, cw, h-len(lines))...)
}

func (m *Model) optionLines(q question, cw, h int) []string {
	type block struct{ label, detail string }
	blocks := make([]block, len(q.options))
	swatchColumn := 0
	for _, o := range q.options {
		w := lipgloss.Width(o.label)
		if o.recommended {
			w += lipgloss.Width("  " + m.th.Badge("recommended", m.th.accent2))
		}
		swatchColumn = max(swatchColumn, w+2)
	}
	for i, o := range q.options {
		selected := i == m.cursor
		pointer, num, label := "  ", m.th.Dim(fmt.Sprintf("%d. ", i+1)), o.label
		if selected {
			pointer, num, label = m.th.Accent("❯ "), m.th.Accent(fmt.Sprintf("%d. ", i+1)), m.th.Accent(o.label)
		}
		box, pad := "", "     "
		if q.multi {
			box, pad = m.th.Dim("[ ] "), "         "
			if m.checks[i] {
				box = m.th.Ok("[✓] ")
			}
		} else if o.recommended {
			label += "  " + m.th.Badge("recommended", m.th.accent2)
		}
		if len(o.swatch) > 0 && !m.th.plain {
			label = padRight(label, swatchColumn) + m.th.Swatch(o.swatch)
		}
		detail := ""
		if o.detail != "" {
			detail = pad + m.th.Dim(ansi.Truncate(o.detail, max(10, cw-len(pad)), "…"))
		}
		blocks[i] = block{pointer + num + box + label, detail}
	}
	for mode := 0; mode < 3; mode++ {
		var lines []string
		focus := 0
		for i, b := range blocks {
			if i == m.cursor {
				focus = len(lines)
			}
			lines = append(lines, b.label)
			if mode < 2 && b.detail != "" {
				lines = append(lines, b.detail)
			}
			if mode == 0 && i < len(blocks)-1 {
				lines = append(lines, "")
			}
		}
		if mode == 2 && blocks[m.cursor].detail != "" {
			lines = append(lines, "", blocks[m.cursor].detail)
		}
		if len(lines) <= h || mode == 2 {
			return window(lines, h, focus)
		}
	}
	return nil
}

func (m *Model) formContent(cw, h int) []string {
	sec := sections[m.sec]
	var lines []string
	if m.width >= sidebarMinWidth {
		lines = append(lines, m.sectionLine(), "")
	}
	lines = append(lines, wrap(m.th.Dim(sec.intro), cw)...)
	lines = append(lines, "")
	labelW, valueW := 0, 0
	for _, q := range sec.questions {
		labelW = max(labelW, lipgloss.Width(q.title))
		for _, o := range q.options {
			valueW = max(valueW, lipgloss.Width(o.label))
		}
	}
	labelW += 2
	valueW += 5
	var rows []string
	for j, q := range sec.questions {
		idx := m.form[j]
		value := "custom"
		if idx >= 0 {
			value = q.options[idx].label
		}
		selected := j == m.cursor
		pointer, label := "  ", padRight(q.title, labelW)
		shown := "  " + m.th.Blue(value)
		if selected {
			pointer, label = m.th.Accent("❯ "), m.th.Bold(label)
			shown = m.th.Dim("‹ ") + m.th.Accent(value) + m.th.Dim(" ›")
		}
		rec := recommendedIndex(q)
		note := ""
		if idx != rec {
			note = m.th.Warn("changed") + m.th.Dim(" · recommended: "+q.options[rec].label)
		} else if selected {
			note = m.th.Badge("recommended", m.th.accent2)
		}
		rows = append(rows, pointer+label+padRight(shown, valueW)+note)
	}
	q := sec.questions[m.cursor]
	var choices []string
	for i, o := range q.options {
		if i == m.form[m.cursor] {
			choices = append(choices, m.th.Accent(o.label))
		} else {
			choices = append(choices, m.th.Dim(o.label))
		}
	}
	help := append([]string{""}, wrap(m.th.Dim(q.help), cw)...)
	help = append(help, m.th.Dim("← ")+strings.Join(choices, m.th.Dim(" · "))+m.th.Dim(" →"))
	avail := h - len(lines) - len(help)
	lines = append(lines, window(rows, avail, m.cursor)...)
	return append(lines, help...)
}

func (m *Model) reviewContent(cw, h int) []string {
	title := m.th.Bold("Review")
	if m.dirty() {
		title += m.th.Dim(" · nothing is written until you save")
	}
	lines := []string{title, ""}
	for _, sec := range sections {
		text := summary(m.draft, sec)
		label := padRight(sec.title, 13)
		if sec.skipped(m.draft) {
			lines = append(lines, m.th.Dim(label+text))
			continue
		}
		for i, part := range wrap(text, max(10, cw-13)) {
			if i > 0 {
				label = strings.Repeat(" ", 13)
			}
			lines = append(lines, m.th.Dim(label)+part)
		}
	}
	lines = append(lines, "")
	switch {
	case m.busy:
		lines = append(lines, m.spin.View()+" Checking your configuration with Neovim…")
	case m.planErr != nil:
		lines = append(lines, wrap(m.th.Bad("✗ ")+firstLine(m.planErr.Error()), cw)...)
	case m.plan != nil:
		created, updated, removed := m.fileChanges()
		var parts []string
		if created > 0 {
			parts = append(parts, fmt.Sprintf("%d new", created))
		}
		if updated > 0 {
			parts = append(parts, fmt.Sprintf("%d updated", updated))
		}
		if removed > 0 {
			parts = append(parts, fmt.Sprintf("%d removed", removed))
		}
		files := "no changes"
		if len(parts) > 0 {
			files = strings.Join(parts, " · ") + " in " + shortPath(m.target)
		}
		lines = append(lines, m.th.Dim(padRight("Files", 13))+ansi.Truncate(files, max(10, cw-13), "…"))
		lines = append(lines, m.th.Dim(padRight("Downloads", 13))+m.downloadSummary())
		for i, w := range m.missingTools() {
			label := ""
			if i == 0 {
				label = "Install first"
			}
			lines = append(lines, m.th.Dim(padRight(label, 13))+m.th.Warn(ansi.Truncate(w, max(10, cw-13), "…")))
		}
	}
	lines = append(lines, "")
	menu := m.menuLines(m.menu, cw, h-len(lines))
	all := append(lines, menu...)
	focus := len(lines)
	for i, line := range menu {
		if strings.Contains(line, "❯") {
			focus = len(lines) + i
		}
	}
	return window(all, h, focus)
}

// dependencies reports required programs, finding Neovim where --nvim points.
func (m *Model) dependencies() []engine.Dependency {
	deps := engine.Dependencies(m.draft, m.target)
	for i, d := range deps {
		if d.ID == "nvim" && m.bin != "" {
			if p, err := exec.LookPath(m.bin); err == nil {
				deps[i].Path, deps[i].Status = p, "present"
			}
		}
	}
	return deps
}

// missingTools lists programs the user must install; Mason downloads language tools itself.
func (m *Model) missingTools() []string {
	mason := engine.Value(m.draft, "tools.provider", "system") == "mason"
	runtimes := map[string]bool{"nvim": true, "git": true, "rg": true, "node": true, "npm": true}
	var out []string
	for _, d := range m.dependencies() {
		if d.Status != "missing" {
			continue
		}
		if _, custom := m.draft.Values["tools.path."+d.ID]; mason && !custom && !runtimes[d.ID] {
			continue
		}
		out = append(out, d.ID+" · "+d.Purpose)
	}
	return out
}

// menuLines shows every detail when there is room, otherwise only the selected one.
func (m *Model) menuLines(items []menuItem, cw, h int) []string {
	build := func(all bool) ([]string, int) {
		var lines []string
		focus := 0
		for i, item := range items {
			selected := i == m.cursor
			pointer, num, label := "  ", m.th.Dim(fmt.Sprintf("%d. ", i+1)), item.label
			if selected {
				focus = len(lines)
				pointer, num, label = m.th.Accent("❯ "), m.th.Accent(fmt.Sprintf("%d. ", i+1)), m.th.Accent(item.label)
			}
			if item.recommended {
				label += "  " + m.th.Badge("recommended", m.th.accent2)
			}
			lines = append(lines, pointer+num+label)
			if item.detail != "" && (all || selected) {
				lines = append(lines, "     "+m.th.Dim(ansi.Truncate(item.detail, max(10, cw-5), "…")))
			}
		}
		return lines, focus
	}
	lines, focus := build(true)
	if len(lines) > h {
		lines, focus = build(false)
	}
	return window(lines, h, focus)
}

func (m *Model) welcomeContent(cw, h int) []string {
	var body []string
	switch m.ownership {
	case "SAVED":
		body = []string{"Welcome back.", m.th.Dim("This configuration was made with ShoshinVim. Change any part of it; your manual Lua in user.lua is never touched.")}
	case "READ ONLY":
		body = []string{m.th.Warn("This folder already holds a Neovim configuration"), m.th.Warn("that ShoshinVim did not create. It will not be changed."), "",
			m.th.Dim("To build a separate configuration next to it, run:"), m.th.Blue(withAppName(runtime.GOOS, "shoshin", "shoshinvim"))}
	case "RECOVERY":
		body = []string{m.th.Bad("An earlier save was interrupted, or generated files"), m.th.Bad("were edited by hand."), ""}
		for _, p := range m.problems {
			body = append(body, m.th.Dim("· "+p))
		}
	default:
		names := make([]string, len(sections))
		for i, sec := range sections {
			names[i] = sec.title
		}
		body = []string{"Build a Neovim you understand, one decision at a time.", "",
			m.th.Dim(strings.Join(names, " · ")), "",
			"Every step has a recommended choice, marked " + m.th.Badge("recommended", m.th.accent2) + ".",
			"Nothing is written or downloaded until you review and save."}
	}
	panelW := min(cw, 72)
	logo := m.th.Accent("◯") + "  " + m.th.Gradient("S H O S H I N V I M") + "   " + m.th.Dim("初心 · beginner's mind")
	content := logo + "\n\n" + strings.Join(wrapAll(body, panelW-8), "\n")
	lines := strings.Split(m.th.Panel(panelW, content), "\n")
	lines = append(lines, "", m.th.Dim("Target  ")+shortPath(m.target)+m.th.Dim(" · ")+m.statusTag(), "")
	return append(lines, m.menuLines(m.menu, cw, h-len(lines))...)
}

func (m *Model) workingContent(cw, h int) []string {
	lines := []string{m.th.Bold(m.jobTitle), ""}
	var log []string
	for _, l := range m.log {
		log = append(log, m.logText(l, cw))
	}
	if m.activity != "" {
		log = append(log, m.spin.View()+" "+ansi.Truncate(m.activity, max(10, cw-2), "…"))
	}
	if m.busy && m.actionsTotal > 0 && m.actionsDone > 0 {
		width := min(cw-12, 40)
		filled := width * m.actionsDone / m.actionsTotal
		log = append(log, "", m.th.Accent(strings.Repeat("━", filled))+m.th.Dim(strings.Repeat("─", width-filled)+fmt.Sprintf("  %d of %d downloads", m.actionsDone, m.actionsTotal)))
	}
	if avail := h - len(lines); len(log) > avail {
		log = log[len(log)-max(0, avail):]
	}
	return append(lines, log...)
}

func (m *Model) logText(l logLine, cw int) string {
	text := ansi.Truncate(l.text, max(10, cw-2), "…")
	switch l.kind {
	case "ok":
		return m.th.Ok("✓ ") + text
	case "bad":
		return m.th.Bad("✗ ") + text
	}
	return m.th.Dim("– " + text)
}

func (m *Model) doneContent(cw, h int) []string {
	var lines []string
	failed := m.failedDownloads()
	switch {
	case m.doneKind == "restore" && m.resultErr == nil:
		lines = append(lines, m.th.Ok("✓ Files restored"), "", m.th.Dim("Your configuration is back to the version you picked."))
	case m.doneKind == "restore":
		lines = append(lines, m.th.Bad("✗ Restore stopped"), "")
		lines = append(lines, wrap(m.resultErr.Error(), cw)...)
	case m.doneKind == "save" && m.resultErr != nil:
		lines = append(lines, m.th.Bad("✗ Your configuration was not saved"), "")
		lines = append(lines, wrap(m.th.Dim(m.resultErr.Error()), cw)...)
	case m.doneKind == "retry" && m.resultErr != nil:
		lines = append(lines, m.th.Warn("Nothing to retry"), "")
		lines = append(lines, wrap(m.th.Dim(m.resultErr.Error()), cw)...)
	default:
		if failed > 0 {
			lines = append(lines, m.th.Warn(fmt.Sprintf("✓ Saved · %d %s did not finish", failed, plural(failed, "download", "downloads"))))
		} else {
			lines = append(lines, m.th.Ok("✓ Your Neovim is ready"))
		}
		lines = append(lines, "", m.th.Dim("Start it with"))
		for _, part := range strings.Split(ansi.Wrap(launchCommand(m.target), max(10, cw-2), "/"), "\n") {
			lines = append(lines, "  "+m.th.Accent(part))
		}
		if engine.Value(m.draft, "experience.layout", "native") == "rich" {
			lines = append(lines, "")
			lines = append(lines, wrap(m.th.Dim("Inside Neovim, press Space and pause to discover shortcuts; Space ? opens the guide."), cw)...)
		}
	}
	if len(m.log) > 0 {
		lines = append(lines, "")
		var log []string
		for _, l := range m.log {
			log = append(log, m.logText(l, cw))
		}
		room := max(0, h-len(lines)-len(m.menu)*2-2)
		if len(log) > room {
			log = append([]string{m.th.Dim(fmt.Sprintf("… %d earlier steps", len(log)-room+1))}, log[len(log)-room+1:]...)
		}
		lines = append(lines, log...)
	}
	lines = append(lines, "")
	return append(lines, m.menuLines(m.menu, cw, h-len(lines))...)
}

func (m *Model) recoveryContent(cw, h int) []string {
	lines := []string{m.th.Bold("Recovery & history"), m.th.Dim("Every save keeps a backup of the files it replaced."), ""}
	if m.screen == restoreScreen {
		lines = []string{m.th.Bold("Restore " + journalTime(m.restoreID)), m.th.Dim("Rollback restores configuration files only; plugins and tools stay installed."), ""}
		for _, j := range m.journals {
			if j.ID != m.restoreID {
				continue
			}
			for _, b := range j.Backups {
				state := "restore previous version"
				if !b.Exists {
					state = "remove (created by this save)"
				}
				lines = append(lines, m.th.Dim("· ")+b.Path+m.th.Dim("  "+state))
			}
		}
		lines = append(lines, "")
	} else if len(m.journals) == 0 {
		lines = append(lines, m.th.Dim("No saves yet."), "")
	}
	return append(lines, m.menuLines(m.menu, cw, h-len(lines))...)
}

func (m *Model) renderOverlay(cw, h int) []string {
	var lines []string
	if m.settingOpen {
		lines = append(lines, m.th.Bold("Set any setting"), m.th.Dim("Neovim options use options.<name>; tool paths use tools.path.<tool>."), "", m.input.View(), "")
		for _, s := range m.settingSuggestions() {
			lines = append(lines, "  "+m.th.Blue(padRight(s.ID, 30))+m.th.Dim(ansi.Truncate(s.Label, max(10, cw-34), "…")))
		}
		return indent(lines)
	}
	lines = append(lines, m.input.View(), "")
	commands := m.filteredCommands()
	for i, c := range commands {
		if i == m.cursor {
			lines = append(lines, m.th.Accent("❯ "+c.label))
		} else {
			lines = append(lines, "  "+c.label)
		}
	}
	if len(commands) == 0 {
		lines = append(lines, m.th.Dim("No matching command"))
	}
	panel := strings.Split(m.th.Panel(min(cw, 56), strings.Join(window(lines, h-4, m.cursor+2), "\n")), "\n")
	return indent(append([]string{m.th.Bold("Commands"), ""}, panel...))
}

// ---- helpers -----------------------------------------------------------------------

func fit(lines []string, h int) []string {
	h = max(0, h)
	if len(lines) > h {
		return lines[:h]
	}
	for len(lines) < h {
		lines = append(lines, "")
	}
	return lines
}

// window returns at most h lines, scrolled so that the focus line stays visible.
func window(lines []string, h, focus int) []string {
	h = max(0, h)
	if len(lines) <= h {
		return lines
	}
	start := min(max(0, focus-h/2), len(lines)-h)
	return lines[start : start+h]
}

func clip(lines []string, w, h int) string {
	lines = fit(lines, h)
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, w, "")
	}
	return strings.Join(lines, "\n")
}

func indent(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = "  " + l
	}
	return out
}

func padRight(s string, w int) string {
	return s + strings.Repeat(" ", max(0, w-lipgloss.Width(s)))
}

func wrap(s string, w int) []string {
	return strings.Split(ansi.Wordwrap(s, max(10, w), " "), "\n")
}

func wrapAll(lines []string, w int) []string {
	var out []string
	for _, l := range lines {
		out = append(out, wrap(l, w)...)
	}
	return out
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// pendingActions are the downloads not yet on disk.
func pendingActions(actions []engine.Action) []engine.Action {
	var out []engine.Action
	for _, a := range actions {
		if !engine.Installed(a) {
			out = append(out, a)
		}
	}
	return out
}

// downloadSummary says what still needs downloading and what is already installed.
func (m *Model) downloadSummary() string {
	if len(m.plan.Actions) == 0 {
		return "none"
	}
	pending := pendingActions(m.plan.Actions)
	installed := len(m.plan.Actions) - len(pending)
	if len(pending) == 0 {
		return fmt.Sprintf("all %d installed", installed)
	}
	text := downloadText(actionCounts(pending))
	if installed > 0 {
		text += fmt.Sprintf(" · %d already installed", installed)
	}
	return text
}

func actionCounts(actions []engine.Action) (plugins, tools int) {
	for _, a := range actions {
		if a.Kind == "git" {
			plugins++
		} else {
			tools++
		}
	}
	return
}

func downloadText(plugins, tools int) string {
	var parts []string
	if plugins > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", plugins, plural(plugins, "plugin", "plugins")))
	}
	if tools > 0 {
		parts = append(parts, fmt.Sprintf("%d language %s", tools, plural(tools, "tool", "tools")))
	}
	return strings.Join(parts, " and ")
}

// actionName turns an engine action description into a short display name.
func actionName(description string) string {
	if rest, ok := strings.CutPrefix(description, "Git clone "); ok {
		repo, _, _ := strings.Cut(rest, " ")
		return path.Base(repo)
	}
	if rest, ok := strings.CutPrefix(description, "Mason registry/download/build for "); ok {
		name, _, _ := strings.Cut(rest, " ")
		return name
	}
	return description
}

func shortPath(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	resolved, _ := filepath.EvalSymlinks(home)
	for _, h := range []string{home, resolved} {
		if h != "" && strings.HasPrefix(p, h+string(filepath.Separator)) {
			return "~" + strings.TrimPrefix(p, h)
		}
	}
	return p
}

func journalTime(id string) string {
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return id
	}
	return time.Unix(0, n).Format("Jan 2 15:04:05")
}

// launchCommand tells the user how to start Neovim with this target. It follows the
// same rule as engine.DataDirFor, so Neovim finds the plugins that were downloaded.
func launchCommand(target string) string {
	switch app := engine.AppName(target); app {
	case "":
		if runtime.GOOS == "windows" {
			return "nvim -u " + filepath.Join(target, "init.lua") // PowerShell does not expand ~ for programs.
		}
		return "nvim -u " + filepath.Join(shortPath(target), "init.lua")
	case "nvim":
		return "nvim"
	default:
		return withAppName(runtime.GOOS, app, "nvim")
	}
}

// withAppName runs command with NVIM_APPNAME set, in the syntax of the platform's
// usual shell: POSIX shells on macOS and Linux, PowerShell on Windows.
func withAppName(goos, app, command string) string {
	if goos == "windows" {
		return "$env:NVIM_APPNAME='" + strings.ReplaceAll(app, "'", "''") + "'; " + command
	}
	return "NVIM_APPNAME=" + app + " " + command
}
