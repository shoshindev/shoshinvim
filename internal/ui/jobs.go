package ui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/shoshindev/shoshinvim/internal/engine"
)

// ---- background work -----------------------------------------------------------

func (m *Model) work(kind string, writing bool, fn func(context.Context, func(string)) jobMsg) tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	m.jobID++
	id := m.jobID
	m.cancel, m.busy, m.writing = cancel, true, writing
	updates := make(chan tea.Msg, 64)
	run := func() tea.Msg {
		go func() {
			result := fn(ctx, func(text string) {
				select {
				case updates <- progressMsg{id: id, text: text, next: updates}:
				case <-ctx.Done():
				}
			})
			result.id, result.kind = id, kind
			updates <- result
			close(updates)
		}()
		return <-updates
	}
	return tea.Batch(run, m.spin.Tick)
}

func (m *Model) startLog(title, activity string, total int) {
	m.screen, m.jobTitle, m.activity = workingScreen, title, activity
	m.log, m.wrote, m.actionsDone, m.actionsTotal = nil, 0, 0, total
	m.result, m.resultErr = nil, nil
	m.names = map[string]string{}
	actions, _ := engine.Actions(m.draft, m.target)
	for _, a := range actions {
		m.names[a.ID] = actionName(a.Description)
	}
}

func (m *Model) save(download bool) tea.Cmd {
	if m.plan == nil {
		return nil
	}
	p := *m.plan
	p.RunExternal = download && len(p.Actions) > 0
	p.Digest = p.Fingerprint()
	total := 0
	if p.RunExternal {
		total = len(p.Actions)
	}
	m.startLog("Saving your configuration", "Checking the plan with Neovim", total)
	m.doneKind = "save"
	bin := m.bin
	return m.work("save", true, func(ctx context.Context, report func(string)) jobMsg {
		r, err := engine.Apply(ctx, p, engine.ApplyOptions{Progress: report, Neovim: bin})
		return jobMsg{result: &r, err: err}
	})
}

func (m *Model) retry() tea.Cmd {
	m.startLog("Retrying downloads", "Checking pending downloads", 0)
	m.doneKind = "retry"
	target, bin := m.target, m.bin
	return m.work("retry", false, func(ctx context.Context, report func(string)) jobMsg {
		r, err := engine.Retry(ctx, target, bin, report)
		return jobMsg{retry: r, err: err}
	})
}

func (m *Model) restore(generated bool) tea.Cmd {
	m.startLog("Restoring files", "Checking backups", 0)
	m.doneKind = "restore"
	target, id := m.target, m.restoreID
	return m.work("restore", true, func(context.Context, func(string)) jobMsg {
		if generated {
			return jobMsg{err: engine.RecoverGenerated(target, id)}
		}
		return jobMsg{err: engine.Rollback(target, id)}
	})
}

func (m *Model) addLog(key, kind, text string) {
	if key != "" {
		for i := range m.log {
			if m.log[i].key == key {
				m.log[i].kind, m.log[i].text = kind, text
				return
			}
		}
	}
	m.log = append(m.log, logLine{key, kind, text})
}

func (m *Model) onProgress(text string) {
	switch {
	case text == "backed-up":
		m.addLog("", "ok", "Checked the plan and backed up current files")
		m.activity = "Writing files"
	case strings.HasPrefix(text, "wrote:"):
		m.wrote++
		m.addLog("wrote", "ok", fmt.Sprintf("Wrote %d %s", m.wrote, plural(m.wrote, "file", "files")))
		m.activity = "Loading the new configuration in an isolated Neovim"
	case text == "verified":
		m.addLog("", "ok", "Verified the configuration in an isolated Neovim")
	case text == "LOCAL_COMMITTED":
		m.persisted, m.ownership, m.writing = engine.Clone(m.draft), "SAVED", false
		m.addLog("", "ok", "Configuration saved")
		m.activity = "Preparing downloads"
	case strings.HasPrefix(text, "["):
		status, rest, _ := strings.Cut(strings.TrimPrefix(text, "["), "] ")
		id, detail, _ := strings.Cut(rest, " ")
		name := m.names[id]
		if name == "" {
			name = strings.TrimPrefix(id, "mason:")
		}
		m.actionsDone++
		switch status {
		case "INSTALLED":
			m.addLog("", "ok", "Downloaded "+name)
		case "FAILED":
			m.addLog("", "bad", name+" failed · "+firstLine(detail))
		default:
			m.addLog("", "skip", name+" · "+firstLine(detail))
		}
	default:
		counter := ""
		if m.actionsTotal > 0 {
			counter = fmt.Sprintf("  %d/%d", m.actionsDone+1, m.actionsTotal)
		}
		m.activity = "Downloading " + actionName(text) + counter
	}
}

func (m *Model) finishJob(msg jobMsg) tea.Cmd {
	m.busy, m.writing, m.activity = false, false, ""
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	switch msg.kind {
	case "lookup":
		return m.finishLookup(msg.lookup, msg.err)
	case "review":
		m.plan, m.planErr = msg.plan, msg.err
		m.menu, m.cursor = m.reviewMenu(), 0
		return nil
	case "save":
		m.result, m.resultErr = msg.result, msg.err
		if msg.err == nil && msg.result != nil && msg.result.LocalCommitted {
			// A save without downloads never reports LOCAL_COMMITTED as progress.
			m.persisted, m.ownership = engine.Clone(m.draft), "SAVED"
		}
	case "retry":
		m.result, m.resultErr = &engine.ApplyResult{LocalCommitted: true, External: msg.retry}, msg.err
	case "restore":
		m.resultErr = msg.err
		if msg.err == nil {
			m.addLog("", "ok", "Restored transaction "+m.restoreID)
		}
	}
	if msg.err != nil || msg.kind == "restore" {
		// Reload what is really on disk after a failure or a restore.
		if ins, err := engine.Inspect(m.target); err == nil {
			m.ownership, m.problems = ins.Ownership, ins.Problems
			if msg.kind == "restore" && ins.State != nil {
				m.draft, m.persisted = engine.Clone(*ins.State), engine.Clone(*ins.State)
			}
		}
	}
	m.screen, m.cursor = doneScreen, 0
	failed := m.failedDownloads()
	switch {
	case msg.kind == "restore":
		m.menu = []menuItem{{"Continue", "", "welcome", true}, {"Quit", "", "quit", false}}
	case msg.err != nil && msg.kind == "save":
		m.menu = []menuItem{{"Back to review", "", "review", true}, {"Quit", "", "quit", false}}
	case failed > 0:
		m.menu = []menuItem{{"Retry failed downloads", "Try the downloads that did not finish", "retry", true}, {"Quit", "", "quit", false}, {"Keep editing", "Return to the review", "review", false}}
	default:
		m.menu = []menuItem{{"Quit", "Then start Neovim with the command above", "quit", true}, {"Keep editing", "Return to the review", "review", false}}
	}
	return nil
}

func (m *Model) failedDownloads() int {
	n := 0
	if m.result != nil {
		for _, r := range m.result.External {
			if r.Status != "installed" {
				n++
			}
		}
	}
	return n
}
