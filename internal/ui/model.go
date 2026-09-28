package ui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
	"github.com/shoshindev/shoshinvim/internal/engine"
)

type Entry struct {
	ID         int
	Role, Text string
}
type progressMsg struct {
	text string
	next <-chan tea.Msg
}

type jobMsg struct {
	kind   string
	plan   *engine.Plan
	result *engine.ApplyResult
	text   string
	err    error
}
type Model struct {
	target, bin         string
	catalog             engine.Catalog
	draft, persisted    engine.State
	undo                []engine.State
	entries             []Entry
	input               textinput.Model
	viewport            viewport.Model
	width, height       int
	focus, viewer       string
	status              string
	busy, writing, quit bool
	cancel              context.CancelFunc
	history             []string
	historyIndex        int
	prompt              string
	proposal            engine.State
	plan                *engine.Plan
	guided              []engine.Setting
	question            int
	recoveryID          string
	recoveryGenerated   bool
	noColor             bool
	palette             string
	savedComposer       string
	selection           int
	transcriptSelection int
}

var commands = []string{"/start", "/back", "/skip", "/why", "/profile", "/configure", "/compare", "/set", "/enable", "/disable", "/show", "/search", "/undo", "/reset", "/review", "/lua", "/diff", "/files", "/health", "/save", "/retry", "/recovery", "/clear", "/help", "/quit"}

func New(target, bin string, noColor bool) (*Model, error) {
	ins, err := engine.Inspect(target)
	if err != nil {
		return nil, err
	}
	s := engine.NewState()
	if ins.State != nil {
		s = engine.Clone(*ins.State)
	}
	i := textinput.New()
	i.Prompt = "> "
	i.Placeholder = "/ for commands"
	i.CharLimit = 8192
	i.Focus()
	m := &Model{target: ins.Target, bin: bin, catalog: engine.LoadCatalog(), draft: s, persisted: engine.Clone(s), input: i, viewport: viewport.New(viewport.WithWidth(80), viewport.WithHeight(18)), width: 80, height: 24, focus: "composer", status: ins.Ownership, noColor: noColor}
	m.append("configurator", "ShoshinVim\nMake Neovim feel like your workspace.\nTarget: "+m.target+"\nChoose a starting point, make it yours, then review and save.")
	if ins.Ownership == "NEW" {
		m.profiles()
	} else {
		m.append("result", ins.Ownership+"\n"+strings.Join(ins.Problems, "\n")+"\n/show  /health  /configure  /review")
	}
	return m, nil
}
func (m *Model) Init() tea.Cmd { return textinput.Blink }
func (m *Model) append(role, text string) {
	text = strings.ReplaceAll(text, "\x1b", "^[")
	m.entries = append(m.entries, Entry{ID: len(m.entries) + 1, Role: role, Text: text})
	m.transcriptSelection = len(m.entries) - 1
	m.refresh()
	m.viewport.GotoBottom()
}
func (m *Model) refresh() {
	var b strings.Builder
	for i, e := range m.entries {
		prefix := ""
		if m.focus == "transcript" && i == m.transcriptSelection {
			prefix = "> "
		}
		fmt.Fprintf(&b, "%s%s\n\n", prefix, e.Text)
	}
	m.viewport.SetContent(lipgloss.NewStyle().Width(max(20, m.width-4)).Render(b.String()))
}
func (m *Model) dirty() bool { return !engine.Equal(m.draft, m.persisted) }
func (m *Model) change(next engine.State) {
	m.undo = append(m.undo, engine.Clone(m.draft))
	m.draft = next
	m.plan = nil
	if m.status != "READ ONLY" && m.status != "RECOVERY" {
		if m.dirty() {
			m.status = "DRAFT"
		} else {
			m.status = "SAVED"
		}
	}
}
func (m *Model) profiles() {
	m.prompt = "profile"
	m.append("choice", "What kind of workspace do you want?\n\n  1. Rich IDE — recommended for Web / Lua\n     Tokyo Night, project sidebar, buffer tabs, Git changes,\n     shortcut hints, language intelligence and formatting.\n  2. Minimal native — fast, simple, no plugins\n  3. Web development — compact language-focused setup\n  4. Lua / Neovim — compact language-focused setup\n  5. Custom — choose every feature\n\nReply with a number. You will review every download and file change.")
}
func (m *Model) work(kind string, fn func(context.Context, func(string)) jobMsg) tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.busy = true
	m.writing = kind == "apply" || kind == "restore"
	m.append("activity", "[..] "+kind+" — please wait")
	updates := make(chan tea.Msg, 64)
	return func() tea.Msg {
		go func() {
			result := fn(ctx, func(text string) {
				select {
				case updates <- progressMsg{text: text, next: updates}:
				case <-ctx.Done():
				}
			})
			result.kind = kind
			updates <- result
			close(updates)
		}()
		return <-updates
	}
}
func (m *Model) complete(j jobMsg) {
	m.busy = false
	m.writing = false
	if m.cancel != nil {
		m.cancel()
	}
	m.cancel = nil
	if j.err != nil {
		m.append("error", "[ERROR] "+j.err.Error())
		if ins, e := engine.Inspect(m.target); e == nil && (ins.Ownership == "RECOVERY" || ins.Ownership == "READ ONLY") {
			m.status = ins.Ownership
		}
		return
	}
	if j.plan != nil {
		m.plan = j.plan
		m.append("result", engine.Summary(*j.plan))
		if j.kind == "save" {
			m.prompt = "save"
			m.append("choice", "Proceed with the reviewed local write?\n  1. Save local configuration\n  2. Save, then run listed external actions\n  3. Cancel\nAn explicit number is required.")
		}
		if j.kind == "review" {
			m.prompt = "reviewed"
			m.append("choice", "Ready for the next step?\n1. Continue to save choices\n2. Keep customizing\n/lua, /diff and /files explain the proposed output.")
		}
	}
	if j.result != nil && j.result.LocalCommitted {
		m.persisted = engine.Clone(m.draft)
		m.status = "SAVED"
		m.plan = nil
		m.append("result", "[OK] Local configuration saved and verified.\nBackup transaction: "+j.result.Transaction+"\n[RESTART] Restart Neovim to load saved changes.\nExisting running sessions were not inspected.")
		for _, r := range j.result.External {
			m.append("result", "["+strings.ToUpper(r.Status)+"] "+r.Action.ID+" "+r.Detail)
		}
	}
	if j.text != "" {
		m.append("result", j.text)
	}
}
func (m *Model) questionText() {
	if m.question < 0 {
		m.question = 0
	}
	if m.question >= len(m.guided) {
		m.prompt = ""
		m.append("result", "Topic complete. /review to inspect the proposal.")
		return
	}
	s := m.guided[m.question]
	m.prompt = "guided"
	text := s.Label + "\n" + s.Description + "\n"
	if s.Type == "boolean" {
		text += "  1. On\n  2. Off\n"
	} else {
		for i, c := range s.Choices {
			fmt.Fprintf(newBuilder(&text), "  %d. %s\n", i+1, c)
		}
	}
	m.append("choice", text+"Reply with a value; /skip leaves it unchanged, /back returns, /why explains.")
}

type appendWriter struct{ p *string }

func (w appendWriter) Write(b []byte) (int, error) { *w.p += string(b); return len(b), nil }
func newBuilder(p *string) io.Writer               { return appendWriter{p} }

func (m *Model) Submit(line string) tea.Cmd {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}
	m.append("user", "> "+line)
	m.history = append(m.history, line)
	m.historyIndex = len(m.history)
	if !strings.HasPrefix(line, "/") {
		return m.answer(line)
	}
	parts := strings.SplitN(line, " ", 3)
	cmd := parts[0]
	arg := ""
	if len(parts) > 1 {
		arg = parts[1]
	}
	rest := ""
	if len(parts) > 2 {
		rest = parts[2]
	}
	errText := func(err error) { m.append("error", "[ERROR] "+err.Error()) }
	switch cmd {
	case "/start":
		m.profiles()
	case "/profile":
		if arg == "" {
			m.profiles()
			break
		}
		next, err := m.catalog.Profile(m.draft, arg)
		if err != nil {
			errText(err)
			break
		}
		m.proposal = next
		m.prompt = "proposal"
		m.append("proposal", "Profile "+arg+" proposes:\n"+stateChanges(m.draft, next)+"\n1. Accept into draft\n2. Cancel\nNo files or tools will change.")
	case "/set", "/enable", "/disable":
		if cmd != "/set" {
			rest = "on"
			if cmd == "/disable" {
				rest = "off"
			}
		}
		v, err := m.catalog.Parse(arg, rest)
		if err != nil {
			errText(err)
			break
		}
		next := engine.Clone(m.draft)
		if err = m.catalog.Set(&next, arg, v, true); err != nil {
			errText(err)
			break
		}
		m.change(next)
		m.append("proposal", "Changed "+arg+" to "+fmt.Sprint(v)+" in the draft. /review checks dependencies.")
	case "/configure":
		if arg == "" {
			arg = "core"
		}
		if arg == "profiles" {
			m.profiles()
			break
		}
		if arg == "dependencies" {
			return m.Submit("/health")
		}
		if arg == "recovery" {
			return m.Submit("/recovery")
		}
		m.guided = nil
		for _, s := range m.catalog.Settings {
			if arg == "core" && strings.HasPrefix(s.ID, "editor.") || strings.HasPrefix(s.ID, arg+".") {
				m.guided = append(m.guided, s)
			}
		}
		if len(m.guided) == 0 {
			m.append("warning", "This topic is planned or is edited through /set options.<name> and declarative state. /search lists supported settings.")
			break
		}
		m.question = 0
		m.questionText()
	case "/back":
		if len(m.guided) > 0 {
			m.question--
			m.questionText()
		} else {
			m.profiles()
		}
	case "/skip":
		if m.prompt == "guided" {
			m.question++
			m.questionText()
		}
	case "/why":
		if m.prompt == "guided" {
			s := m.guided[m.question]
			m.append("configurator", s.Description+"\nSetting: "+s.ID+"\nGenerated option: "+s.Option+"\nChanges apply after save and restarting Neovim.")
		} else {
			m.append("configurator", "Choices edit the draft. /lua shows executable output; /files shows ownership; /review checks the complete proposal.")
		}
	case "/show":
		m.viewer = string(engine.JSON(m.draft))
		m.viewport.SetContent(m.viewer)
		m.focus = "viewer"
	case "/search":
		found := m.catalog.Search(strings.TrimSpace(arg + " " + rest))
		var b strings.Builder
		for _, s := range found {
			fmt.Fprintf(&b, "%-32s %s\n", s.ID, s.Label)
		}
		m.viewer = b.String()
		m.viewport.SetContent(m.viewer)
		m.focus = "viewer"
	case "/compare":
		text, err := m.catalog.Comparison(arg)
		if err != nil {
			errText(err)
			break
		}
		m.viewer = text
		m.focus = "viewer"
		m.viewport.SetContent(m.viewer)
	case "/undo":
		if len(m.undo) == 0 {
			m.append("warning", "Nothing to undo.")
			break
		}
		next := m.undo[len(m.undo)-1]
		m.undo = m.undo[:len(m.undo)-1]
		m.draft = next
		m.plan = nil
		if m.status != "READ ONLY" && m.status != "RECOVERY" {
			if m.dirty() {
				m.status = "DRAFT"
			} else {
				m.status = "SAVED"
			}
		}
		m.append("proposal", "Undid the latest draft change.")
	case "/reset":
		m.proposal = engine.Clone(m.draft)
		if arg == "" || arg == "all" {
			m.proposal = engine.NewState()
		} else {
			for id := range m.proposal.Values {
				if strings.HasPrefix(id, arg+".") {
					delete(m.proposal.Values, id)
					delete(m.proposal.Explicit, id)
				}
			}
		}
		m.prompt = "proposal"
		m.append("choice", "Reset proposal:\n"+stateChanges(m.draft, m.proposal)+"\n1. Accept into draft\n2. Cancel")
	case "/review", "/save":
		state := engine.Clone(m.draft)
		target, bin := m.target, m.bin
		kind := strings.TrimPrefix(cmd, "/")
		export := ""
		if arg == "export" {
			export = rest
		}
		return m.work(kind, func(ctx context.Context, report func(string)) jobMsg {
			p, err := engine.BuildPlan(ctx, target, state, bin, false)
			if err == nil && export != "" {
				err = exportPlan(export, p)
			}
			if err != nil {
				return jobMsg{err: err}
			}
			return jobMsg{plan: &p}
		})
	case "/lua":
		files, err := engine.Generate(m.draft)
		if err != nil {
			errText(err)
			break
		}
		keys := make([]string, 0, len(files))
		for k := range files {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var b strings.Builder
		for _, name := range keys {
			if arg == "" || strings.Contains(name, arg) {
				fmt.Fprintf(&b, "-- %s\n%s\n", name, files[name])
			}
		}
		m.viewer = b.String()
		m.viewport.SetContent(m.viewer)
		m.focus = "viewer"
	case "/diff", "/files":
		if m.plan == nil {
			m.append("warning", "Run /review first to build a fresh file plan.")
			break
		}
		m.viewer = engine.Summary(*m.plan)
		if cmd == "/diff" {
			m.viewer = engine.Diff(*m.plan)
		}
		m.viewport.SetContent(m.viewer)
		m.focus = "viewer"
	case "/health":
		state := engine.Clone(m.draft)
		m.append("result", string(engine.JSON(engine.Dependencies(state))))
	case "/retry":
		target := m.target
		return m.work("retry", func(ctx context.Context, report func(string)) jobMsg {
			r, err := engine.Retry(ctx, target, report)
			return jobMsg{text: string(engine.JSON(r)), err: err}
		})
	case "/recovery":
		js, err := engine.Journals(m.target)
		if err != nil {
			errText(err)
			break
		}
		if arg == "restore" || arg == "generated" {
			found := false
			for _, j := range js {
				if j.ID == rest {
					found = true
					m.recoveryID = j.ID
					m.recoveryGenerated = arg == "generated"
					m.prompt = "recovery"
					m.append("choice", "Recovery transaction: "+j.ID+"\n"+engine.Summary(j.Plan)+"\nRestore backups, or archive manual changes before restoring generated files (generated mode).\n1. Confirm this recovery\n2. Cancel")
					break
				}
			}
			if !found {
				m.append("error", "Unknown transaction ID.")
			}
			break
		}
		var b strings.Builder
		for _, j := range js {
			fmt.Fprintf(&b, "%s  %s\n", j.ID, j.Status)
		}
		m.append("result", b.String()+"/recovery restore <id> restores verified backups.\n/recovery generated <id> archives drift then restores the generated snapshot.\nNo destructive choice is preselected.")
	case "/clear":
		if m.dirty() {
			m.append("warning", "Unsaved draft: /clear is unavailable.")
		} else {
			m.entries = nil
			m.refresh()
		}
	case "/help":
		if arg != "" {
			if s, err := m.catalog.Find(arg); err == nil {
				m.append("configurator", s.Label+"\n"+s.Description)
				break
			}
		}
		m.viewer = helpText
		m.viewport.SetContent(m.viewer)
		m.focus = "viewer"
	case "/quit":
		if m.dirty() {
			m.prompt = "quit"
			m.append("choice", "Unsaved changes:\n1. Return to the session\n2. Discard draft and quit")
		} else {
			m.quit = true
		}
	default:
		m.append("error", "Unknown command. /help lists supported commands.")
	}
	return nil
}

func (m *Model) answer(line string) tea.Cmd {
	switch m.prompt {
	case "profile":
		if line == "1" || line == "Rich IDE" {
			m.proposal, _ = m.catalog.Profile(m.draft, "rich-ide")
			m.prompt = "rich-languages"
			m.append("choice", "What will you work on?\n1. Web + Lua / Neovim — recommended\n2. Web only — JavaScript, TypeScript, HTML, CSS and JSON\n3. Lua / Neovim only\n\nLanguage servers provide completion, navigation and diagnostics.")
			return nil
		}
		names := map[string]string{"2": "minimal", "3": "web", "4": "lua", "5": "custom", "Minimal native": "minimal", "Web development": "web", "Lua / Neovim": "lua", "Custom": "custom"}
		if name, ok := names[line]; ok {
			return m.Submit("/profile " + name)
		}
	case "rich-languages":
		if line == "1" || line == "2" || line == "3" {
			for _, lang := range []string{"typescript", "html", "css", "json", "lua"} {
				enabled := line == "1" || (line == "2" && lang != "lua") || (line == "3" && lang == "lua")
				_ = m.catalog.Set(&m.proposal, "language."+lang+".lsp", enabled, true)
			}
			m.prompt = "rich-format"
			m.append("choice", "When should code be formatted?\n1. When I ask — Space cf formats the file\n2. Whenever I save — Prettier for Web, StyLua for Lua\n\nBoth use the same formatting tools. Existing project formatter settings are respected.")
			return nil
		}
	case "rich-format":
		if line == "1" || line == "2" {
			_ = m.catalog.Set(&m.proposal, "formatting.on_save", line == "2", true)
			m.prompt = "proposal"
			m.append("proposal", "Your Rich IDE workspace\n\n  Look: Tokyo Night, statusline and buffer tabs\n  Navigate: project sidebar, file/text search, recent files\n  Code: completion, diagnostics, rename and formatting\n  Learn: press Space to discover commands; Space ? for a guide\n\nNo special font is required. Git and selected language tools are\nchecked during review; missing runtimes get manual instructions.\n\n1. Accept this setup into the draft\n2. Cancel\n\nNo files have changed. /show and /lua expose every setting.")
			return nil
		}
	case "next":
		if line == "1" {
			return m.Submit("/review")
		}
		if line == "2" {
			return m.Submit("/configure core")
		}
		if line == "3" {
			return m.Submit("/health")
		}
	case "reviewed":
		if line == "1" {
			return m.Submit("/save")
		}
		if line == "2" {
			m.prompt = "next"
			m.append("choice", "1. Review\n2. Customize editor settings\n3. Check dependencies\n/configure appearance changes the look.")
			return nil
		}
	case "proposal":
		if line == "1" {
			m.change(m.proposal)
			m.prompt = "next"
			m.append("proposal", "Your draft is ready.\n1. Review the configuration and downloads\n2. Customize editor settings\n3. Check dependencies\n\n/configure appearance changes the look; /undo goes back.")
			return nil
		}
		if line == "2" {
			m.prompt = ""
			m.append("result", "Cancelled.")
			return nil
		}
	case "guided":
		s := m.guided[m.question]
		value := line
		if s.Type == "boolean" {
			if line == "1" {
				value = "on"
			}
			if line == "2" {
				value = "off"
			}
		} else if len(s.Choices) > 0 {
			if n, e := strconv.Atoi(line); e == nil && n > 0 && n <= len(s.Choices) {
				value = s.Choices[n-1]
			}
		}
		v, err := m.catalog.Parse(s.ID, value)
		if err == nil {
			next := engine.Clone(m.draft)
			err = m.catalog.Set(&next, s.ID, v, true)
			if err == nil {
				m.change(next)
				m.append("proposal", "Changed "+s.Label+" to "+fmt.Sprint(v))
				m.question++
				m.questionText()
				return nil
			}
		}
		m.append("error", fmt.Sprint(err))
		return nil
	case "quit":
		if line == "2" {
			m.quit = true
			return nil
		}
		if line == "1" {
			m.prompt = ""
			return nil
		}
	case "save":
		if line == "3" {
			m.prompt = ""
			m.append("result", "Cancelled. No files changed.")
			return nil
		}
		if (line == "1" || line == "2") && m.plan != nil {
			p := *m.plan
			p.RunExternal = line == "2"
			p.Digest = p.Fingerprint()
			m.prompt = ""
			return m.work("apply", func(ctx context.Context, report func(string)) jobMsg {
				r, err := engine.Apply(ctx, p, engine.ApplyOptions{Progress: report})
				return jobMsg{result: &r, err: err}
			})
		}
	case "recovery":
		if line == "2" {
			m.prompt = ""
			return nil
		}
		if line == "1" {
			target, id, generated := m.target, m.recoveryID, m.recoveryGenerated
			m.prompt = ""
			return m.work("restore", func(ctx context.Context, report func(string)) jobMsg {
				var err error
				if generated {
					err = engine.RecoverGenerated(target, id)
				} else {
					err = engine.Rollback(target, id)
				}
				return jobMsg{text: "Recovery finished. Reopen ShoshinVim to load the restored state.", err: err}
			})
		}
	}
	m.append("error", "Input was not recognized. Answer the active choice exactly or use /help; no configuration changed.")
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case progressMsg:
		if msg.text == "LOCAL_COMMITTED" {
			m.persisted = engine.Clone(m.draft)
			m.writing = false
			m.status = "INSTALLING"
			m.append("result", "[OK] Local configuration committed. External work has a separate outcome.")
		} else {
			m.append("activity", "[..] "+msg.text)
		}
		return m, func() tea.Msg { return <-msg.next }
	case jobMsg:
		m.complete(msg)
		return m, nil
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.SetWidth(max(10, m.width-6))
		m.viewport.SetWidth(max(10, m.width-4))
		m.viewport.SetHeight(max(3, m.height-7))
		if m.focus == "viewer" {
			m.viewport.SetContent(m.viewer)
		} else {
			m.refresh()
		}
		return m, nil
	case tea.KeyPressMsg:
		key := msg.String()
		if m.palette != "" {
			return m.paletteUpdate(msg)
		}
		if m.busy {
			if key == "ctrl+c" || key == "esc" {
				if m.writing {
					m.append("warning", "Local write/verification cannot be interrupted. Recovery is journaled.")
				} else if m.cancel != nil {
					m.cancel()
				}
			}
			return m, nil
		}
		if m.width < 52 || m.height < 16 {
			if key == "ctrl+d" {
				cmd := m.Submit("/quit")
				if m.quit {
					return m, tea.Quit
				}
				return m, cmd
			}
			if key == "1" || key == "2" {
				m.answer(key)
				if m.quit {
					return m, tea.Quit
				}
			}
			return m, nil
		}
		if m.focus == "viewer" {
			if key == "esc" || key == "q" || key == "tab" {
				m.focus = "composer"
				m.viewer = ""
				m.refresh()
				return m, m.input.Focus()
			}
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
		if key == "ctrl+p" {
			m.openPalette("commands")
			return m, nil
		}
		if key == "?" && m.input.Value() == "" {
			m.viewer = helpText
			m.focus = "viewer"
			m.viewport.SetContent(m.viewer)
			return m, nil
		}
		if m.focus == "transcript" {
			if key == "/" {
				m.openPalette("transcript")
				return m, nil
			}
			if (key == "j" || key == "down" || key == "k" || key == "up") && len(m.entries) > 0 {
				delta := 1
				if key == "k" || key == "up" {
					delta = -1
				}
				m.transcriptSelection = min(len(m.entries)-1, max(0, m.transcriptSelection+delta))
				m.refresh()
				line := 0
				for _, e := range m.entries[:m.transcriptSelection] {
					line += strings.Count(e.Text, "\n") + 2
				}
				m.viewport.SetYOffset(line)
				return m, nil
			}
			if key == "tab" || key == "i" {
				m.focus = "composer"
				return m, m.input.Focus()
			}
			if key == "y" {
				if len(m.entries) > 0 {
					return m, tea.SetClipboard(m.entries[m.transcriptSelection].Text)
				}
				return m, nil
			}
			if key == "g" {
				m.viewport.GotoTop()
				return m, nil
			}
			if key == "G" {
				m.viewport.GotoBottom()
				return m, nil
			}
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
		switch key {
		case "enter":
			line := m.input.Value()
			m.input.SetValue("")
			cmd := m.Submit(line)
			if m.quit {
				return m, tea.Quit
			}
			return m, cmd
		case "tab":
			matches := m.completions()
			if len(matches) > 0 {
				if len(matches) == 1 {
					m.input.SetValue(matches[0] + " ")
					m.input.CursorEnd()
				} else {
					m.openPalette("completion")
				}
			} else {
				m.focus = "transcript"
				m.input.Blur()
			}
			return m, nil
		case "shift+tab":
			m.focus = "transcript"
			m.input.Blur()
			return m, nil
		case "up", "down":
			if key == "up" {
				m.historyIndex = max(0, m.historyIndex-1)
			} else {
				m.historyIndex = min(len(m.history), m.historyIndex+1)
			}
			if m.historyIndex < len(m.history) {
				m.input.SetValue(m.history[m.historyIndex])
			} else {
				m.input.SetValue("")
			}
			m.input.CursorEnd()
			return m, nil
		case "ctrl+r":
			m.openPalette("history")
			return m, nil
		case "ctrl+c", "esc":
			m.input.SetValue("")
			return m, nil
		case "ctrl+d":
			if m.input.Value() == "" {
				cmd := m.Submit("/quit")
				if m.quit {
					return m, tea.Quit
				}
				return m, cmd
			}
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}
func (m *Model) completions() []string {
	value := m.input.Value()
	if !strings.HasPrefix(value, "/") {
		return nil
	}
	var out []string
	if strings.HasPrefix(value, "/set ") {
		prefix := strings.TrimPrefix(value, "/set ")
		for _, s := range m.catalog.Search("") {
			if strings.HasPrefix(s.ID, prefix) {
				out = append(out, "/set "+s.ID)
			}
		}
	} else {
		for _, c := range commands {
			if strings.HasPrefix(c, value) {
				out = append(out, c)
			}
		}
	}
	return out
}
func (m *Model) View() tea.View {
	if m.width < 52 || m.height < 16 {
		v := tea.NewView(fmt.Sprintf("ShoshinVim\nResize terminal: %dx%d; minimum 52x16.\nDraft is retained. Ctrl+D: /quit\n%s", m.width, m.height, func() string {
			if m.prompt == "quit" {
				return "Unsaved draft: 1 return, 2 discard and quit"
			}
			return ""
		}()))
		v.AltScreen = true
		return v
	}
	if m.palette != "" {
		options := m.paletteOptions()
		var b strings.Builder
		fmt.Fprintf(&b, "ShoshinVim — %s\n\n", m.palette)
		start := max(0, m.selection-max(3, m.height-8)+1)
		for i := start; i < min(len(options), start+max(3, m.height-8)); i++ {
			prefix := "  "
			if i == m.selection {
				prefix = "> "
			}
			fmt.Fprintf(&b, "%s%s\n", prefix, options[i])
		}
		b.WriteString("\n" + m.input.View() + "\nUp/Down select   Enter choose   Esc return")
		v := tea.NewView(lipgloss.NewStyle().Width(m.width).Render(b.String()))
		v.AltScreen = true
		return v
	}
	title := "ShoshinVim"
	if m.width >= 80 {
		title += "  " + m.target
	}
	if !m.noColor {
		title = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render(title)
	}
	status := m.status
	if m.dirty() && status != "READ ONLY" && status != "RECOVERY" {
		status = "DRAFT"
	}
	if m.busy {
		if status != "INSTALLING" {
			status = "WORKING"
		}
		if m.writing {
			status = "SAVING"
		}
	}
	footer := "Enter send   Tab focus/complete   Ctrl+P commands   ? help"
	if m.focus == "viewer" {
		footer = "j/k scroll   Esc return"
	}
	if m.width < 80 {
		footer = "Enter send   ? help"
	}
	completion := ""
	matches := m.completions()
	if len(matches) > 0 && m.focus == "composer" {
		completion = strings.Join(matches[:min(3, len(matches))], "  ")
	}
	title = ansi.Truncate(title, m.width, "…")
	completion = ansi.Truncate(completion, m.width, "…")
	v := tea.NewView(title + "\n\n" + m.viewport.View() + "\n" + m.input.View() + "\n" + status + " | " + m.focus + "\n" + completion + "\n" + footer)
	v.AltScreen = true
	return v
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
func RunPlain(target, bin string, in io.Reader, out io.Writer) error {
	m, err := New(target, bin, true)
	if err != nil {
		return err
	}
	cursor := 0
	printNew := func() {
		if cursor > len(m.entries) {
			cursor = 0
		}
		for _, e := range m.entries[cursor:] {
			fmt.Fprintln(out, e.Text)
		}
		cursor = len(m.entries)
		if m.focus == "viewer" {
			fmt.Fprintln(out, m.viewer)
			m.focus = "composer"
		}
	}
	printNew()
	scan := bufio.NewScanner(in)
	scan.Buffer(make([]byte, 4096), 1<<20)
	for !m.quit {
		fmt.Fprint(out, "> ")
		if !scan.Scan() {
			if m.dirty() {
				return fmt.Errorf("input ended with an unsaved draft; no configuration was written")
			}
			return scan.Err()
		}
		cmd := m.Submit(scan.Text())
		printNew()
		for cmd != nil {
			_, cmd = m.Update(cmd())
			printNew()
		}
		printNew()
	}
	return nil
}
func exportPlan(path string, p engine.Plan) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, err = f.Write(engine.JSON(p))
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}
func stateChanges(a, b engine.State) string {
	var rows []string
	for k, v := range b.Values {
		if fmt.Sprint(a.Values[k]) != fmt.Sprint(v) {
			rows = append(rows, fmt.Sprintf("%s: %v -> %v", k, a.Values[k], v))
		}
	}
	for k := range a.Values {
		if _, ok := b.Values[k]; !ok {
			rows = append(rows, k+": reset to Neovim default")
		}
	}
	sort.Strings(rows)
	if len(rows) == 0 {
		return "No setting values change."
	}
	return strings.Join(rows, "\n")
}

const helpText = `ShoshinVim commands
/start  /profile minimal|web|lua|custom
/configure core|navigation|language|formatting|plugins
/set <id> <value>  /enable <id>  /disable <id>
/search <text>  /compare <capability>  /show
/back  /skip  /why  /undo  /reset [topic|all]
/review  /review export <new-file.json>
/lua [file]  /diff  /files  /health
/save  /retry  /recovery
/recovery restore <transaction-id>
/recovery generated <transaction-id>
/clear  /help [id]  /quit

Tab completes commands/settings or moves focus. Shift+Tab selects transcript.
Up/Down: prompt history. Ctrl+R: history viewer. Ctrl+P: commands.
Transcript: j/k, PgUp/PgDown, g/G, y copy, Tab/i input.
Viewers: j/k/PgUp/PgDown scroll, Esc returns with input preserved.
Ctrl+C cancels non-write work or clears input. Ctrl+D quits an empty prompt.
No color, icons, mouse, credentials, AI model, or network is required to configure.
Draft, saved configuration, installed dependencies, and running Neovim are separate.
`

func (m *Model) openPalette(kind string) {
	m.palette = kind
	m.savedComposer = m.input.Value()
	m.selection = 0
	m.input.SetValue("")
}
func (m *Model) paletteOptions() []string {
	var source []string
	switch m.palette {
	case "commands":
		source = commands
	case "history":
		source = m.history
	case "completion":
		value := m.input.Value()
		m.input.SetValue(m.savedComposer)
		source = m.completions()
		m.input.SetValue(value)
	case "transcript":
		for _, e := range m.entries {
			source = append(source, fmt.Sprintf("%d. %s", e.ID, strings.ReplaceAll(e.Text, "\n", " ")))
		}
	}
	var out []string
	for _, item := range source {
		if strings.Contains(strings.ToLower(item), strings.ToLower(m.input.Value())) {
			out = append(out, item)
		}
	}
	return out
}
func (m *Model) paletteUpdate(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	options := m.paletteOptions()
	switch msg.String() {
	case "esc", "ctrl+c":
		m.palette = ""
		m.input.SetValue(m.savedComposer)
		m.input.CursorEnd()
		return m, nil
	case "down":
		m.selection = min(max(0, len(options)-1), m.selection+1)
		return m, nil
	case "up":
		m.selection = max(0, m.selection-1)
		return m, nil
	case "enter":
		if len(options) == 0 {
			return m, nil
		}
		chosen := options[min(m.selection, len(options)-1)]
		if m.palette == "transcript" {
			var id int
			fmt.Sscanf(chosen, "%d.", &id)
			m.transcriptSelection = max(0, id-1)
			m.input.SetValue(m.savedComposer)
			m.focus = "transcript"
			m.refresh()
		} else {
			m.input.SetValue(chosen + " ")
			m.input.CursorEnd()
			m.focus = "composer"
		}
		m.palette = ""
		return m, nil
	}
	m.selection = 0
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}
