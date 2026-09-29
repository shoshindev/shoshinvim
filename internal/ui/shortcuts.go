package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/shoshindev/shoshinvim/internal/engine"
)

type keyRow struct {
	kind    string // continue, add, heading, key
	heading string
	key     engine.EffectiveKeymap
}

var shortcutGroupOrder = []string{"Find", "Files", "Code", "Git", "Debug", "Buffers", "Windows", "Editor", "Yours"}

func (m *Model) shortcutRows() []keyRow {
	rows := []keyRow{{kind: "continue"}, {kind: "add"}}
	keys := engine.EffectiveKeymaps(m.draft)
	for _, group := range shortcutGroupOrder {
		first := true
		for _, k := range keys {
			if k.Group != group {
				continue
			}
			if first {
				rows = append(rows, keyRow{kind: "heading", heading: group})
				first = false
			}
			rows = append(rows, keyRow{kind: "key", key: k})
		}
	}
	return rows
}

func (m *Model) shortcutTyping() bool {
	return m.screen == stepScreen && !sections[m.sec].form && sections[m.sec].questions[m.q].keymaps && (m.keyMode == "key" || m.keyMode == "command")
}

// actionOptions lists every shortcut action, grouped, then a custom command.
func actionOptions() []option {
	var out []option
	for _, a := range engine.KeyActions {
		out = append(out, option{label: a.Label, short: a.ID, group: a.Group})
	}
	return append(out, option{label: "Run a command…", short: engine.CommandAction, group: "Your own", detail: "Any Ex command, such as Lazy or Telescope git_commits"})
}

func (m *Model) shortcutKey(key string) tea.Cmd {
	if m.keyMode == "action" {
		options := actionOptions()
		switch key {
		case "up", "k":
			m.cursor = max(0, m.cursor-1)
		case "down", "j":
			m.cursor = min(len(options)-1, m.cursor+1)
		case "esc":
			m.keyMode, m.cursor = "", 1
		case "enter":
			chosen := options[m.cursor]
			m.input.SetValue("")
			if chosen.short == engine.CommandAction {
				m.keyMode = "command"
				m.input.Placeholder = "Lazy"
			} else {
				m.keyAdd, m.keyMode = chosen.short, "key"
				m.input.Placeholder = "<leader>x"
			}
			m.input.Focus()
		}
		return nil
	}
	rows := m.shortcutRows()
	move := func(dir int) {
		for i := m.cursor + dir; i >= 0 && i < len(rows); i += dir {
			if rows[i].kind != "heading" {
				m.cursor = i
				return
			}
		}
	}
	m.cursor = min(m.cursor, len(rows)-1)
	switch key {
	case "up", "k", "shift+tab":
		move(-1)
	case "down", "j", "tab":
		move(1)
	case "esc", "left", "h":
		return m.moveQuestion(-1)
	case "a":
		m.keyMode, m.cursor = "action", 0
	case "R":
		m.change(restoreShortcuts(m.draft))
		m.flash("Restored the style's shortcuts; your changes were removed.", false)
	case "d", "x", "delete", "backspace":
		if rows[m.cursor].kind == "key" {
			k := rows[m.cursor].key
			m.change(turnOffShortcut(m.draft, k))
			m.flash(fmt.Sprintf("Turned off %s (%s). R restores the style.", engine.KeyLabel(m.draft, k.Key), k.Label), false)
			m.cursor = min(m.cursor, len(m.shortcutRows())-1)
		}
	case "enter":
		switch rows[m.cursor].kind {
		case "continue":
			return m.moveQuestion(1)
		case "add":
			m.keyMode, m.cursor = "action", 0
		case "key":
			k := rows[m.cursor].key
			m.keyTarget, m.keyMode = &k, "key"
			m.input.SetValue("")
			m.input.Placeholder = k.Key
			m.input.Focus()
		}
	}
	return nil
}

func (m *Model) shortcutInputKey(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.keyMode, m.keyTarget, m.keyAdd = "", nil, ""
		m.input.Blur()
		return nil
	case "enter":
		value := strings.TrimSpace(m.input.Value())
		if m.keyMode == "command" {
			if value == "" || strings.ContainsAny(value, "\n\r") {
				m.flash("Type a command, for example Lazy.", true)
				return nil
			}
			m.keyAdd, m.keyMode = engine.CommandAction+strings.TrimPrefix(value, ":"), "key"
			m.input.SetValue("")
			m.input.Placeholder = "<leader>x"
			return nil
		}
		if !engine.ValidShortcutKey(value) {
			m.flash("Type a key in Vim notation, for example <leader>ff, <C-s> or ]q.", true)
			return nil
		}
		if m.keyTarget != nil {
			m.change(rebindShortcut(m.draft, *m.keyTarget, value))
			m.flash(fmt.Sprintf("%s is now on %s.", m.keyTarget.Label, engine.KeyLabel(m.draft, value)), false)
		} else {
			mode := "n"
			if m.keyAdd == "terminal.normal" {
				mode = "t"
			}
			m.change(addShortcut(m.draft, mode, value, m.keyAdd))
			m.flash(fmt.Sprintf("Added %s.", engine.KeyLabel(m.draft, value)), false)
		}
		m.keyMode, m.keyTarget, m.keyAdd = "", nil, ""
		m.input.Blur()
		return nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return cmd
}

func (m *Model) shortcutContent(cw, h int) []string {
	var lines []string
	switch m.keyMode {
	case "key":
		label := "the new shortcut"
		if m.keyTarget != nil {
			label = fmt.Sprintf("%s (now %s)", m.keyTarget.Label, engine.KeyLabel(m.draft, m.keyTarget.Key))
		} else if a, ok := engine.KeyActionFor(m.keyAdd); ok {
			label = a.Label
		}
		return append(lines, "", m.th.Bold("Key for "+label), m.th.Dim("Vim notation, for example <leader>ff, <C-s>, <S-h> or ]q."), "", m.input.View())
	case "command":
		return append(lines, "", m.th.Bold("Command to run"), m.th.Dim("Without the colon, for example Lazy or Telescope git_commits."), "", m.input.View())
	case "action":
		q := question{options: actionOptions()}
		return append(append(lines, "", m.th.Bold("What should the shortcut do?")), m.optionLines(q, cw, h-2)...)
	}
	rows := m.shortcutRows()
	focus := 0
	var body []string
	for i, r := range rows {
		selected := i == m.cursor
		pointer := "  "
		if selected {
			pointer, focus = m.th.Accent("❯ "), len(body)
		}
		switch r.kind {
		case "continue":
			label := "Keep these shortcuts and continue"
			if selected {
				label = m.th.Accent(label)
			}
			body = append(body, pointer+label)
		case "add":
			label := "+ Add a shortcut"
			if selected {
				label = m.th.Accent(label)
			}
			body = append(body, pointer+label)
		case "heading":
			body = append(body, "", m.th.Dim(strings.ToUpper(r.heading)))
		case "key":
			key, label := padRight(engine.KeyLabel(m.draft, r.key.Key), 18), r.key.Label
			if selected {
				key, label = m.th.Accent(key), m.th.Bold(label)
			}
			if mode := engine.ModeLabel(r.key.Mode); mode != "" {
				label += m.th.Dim(" · " + mode)
			}
			body = append(body, pointer+key+padRight(label, 46)+m.th.Dim(r.key.Source))
		}
	}
	var notes []string
	if n := changedShortcuts(m.draft); n > 0 {
		notes = append(notes, m.th.Warn(fmt.Sprintf("%d change%s to the style · R restores it", n, map[bool]string{true: "", false: "s"}[n == 1])))
	}
	for _, c := range engine.KeymapConflicts(m.draft) {
		notes = append(notes, m.th.Warn("! "+c))
	}
	notes = append(notes, wrap(m.th.Dim("Always available from Neovim: grn rename · gra code action · grr references · gri implementation · gO symbols · K documentation · [d ]d diagnostics"), cw)...)
	avail := h - len(notes) - 1
	return append(append(window(body, avail, focus), ""), notes...)
}

// shortcutHint names the keys configured for a step's actions, for example
// "Your shortcuts: Space sf find files · Space sg search text in the project".
func shortcutHint(s engine.State, actions []string) string {
	keys := engine.EffectiveKeymaps(s)
	var parts []string
	for _, id := range actions {
		for _, k := range keys {
			if k.Action == id {
				parts = append(parts, engine.KeyLabel(s, k.Key)+" "+strings.ToLower(k.Label[:1])+k.Label[1:])
				break
			}
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "Your shortcuts: " + strings.Join(parts, " · ")
}

// shortcutLabel names a shortcut and its modes other than normal, for listings.
func shortcutLabel(k engine.EffectiveKeymap) string {
	if mode := engine.ModeLabel(k.Mode); mode != "" {
		return k.Label + " · " + mode
	}
	return k.Label
}
