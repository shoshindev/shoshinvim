package ui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"runtime"
	"strconv"
	"strings"

	"github.com/shoshindev/shoshinvim/internal/engine"
)

// RunPlain asks the same questions as the full-screen interface, one line at a time,
// with normal terminal scrollback and no escape sequences.
func RunPlain(target, bin string, in io.Reader, out io.Writer) error {
	ins, err := engine.Inspect(target)
	if err != nil {
		return err
	}
	c := engine.LoadCatalog()
	s := engine.NewState()
	if ins.State != nil {
		s = engine.Clone(*ins.State)
	}
	persisted := engine.Clone(s)
	fmt.Fprintf(out, "ShoshinVim — build a Neovim you understand.\nTarget: %s (%s)\n", ins.Target, strings.ToLower(ins.Ownership))
	switch ins.Ownership {
	case "READ ONLY":
		fmt.Fprintln(out, "This folder holds a configuration ShoshinVim did not create; it will not be changed.\nTo build a separate one, run: "+withAppName(runtime.GOOS, "shoshin", "shoshinvim --no-alt-screen"))
		return nil
	case "RECOVERY":
		fmt.Fprintln(out, strings.Join(ins.Problems, "\n")+"\nRun `shoshinvim rollback` to list and restore earlier saves.")
		return nil
	}
	fmt.Fprintln(out, "Each question has a recommended choice. Press Enter to accept it, type a number to choose, b to go back, q to quit.")
	scan := bufio.NewScanner(in)
	scan.Buffer(make([]byte, 4096), 1<<20)
	read := func(prompt string) (string, bool) {
		fmt.Fprint(out, prompt)
		if !scan.Scan() {
			fmt.Fprintln(out)
			return "", false
		}
		return strings.TrimSpace(scan.Text()), true
	}
	ended := func() error {
		if !engine.Equal(s, persisted) {
			return fmt.Errorf("input ended before saving; no configuration was written")
		}
		return nil
	}

	type ref struct{ sec, q int }
	var steps []ref
	for i, sec := range sections {
		for j := range sec.questions {
			steps = append(steps, ref{i, j})
		}
	}
	shown := -1
	for i, dir := 0, 1; i < len(steps); {
		if i < 0 {
			i, dir = 0, 1
		}
		st := steps[i]
		sec, q := sections[st.sec], sections[st.sec].questions[st.q]
		if q.skipped(s) {
			i += dir
			continue
		}
		q = q.resolve(s)
		if st.sec != shown {
			fmt.Fprintf(out, "\n── %d/%d %s ── %s\n", st.sec+1, len(sections), sec.title, sec.intro)
			shown = st.sec
		}
		fmt.Fprintf(out, "\n%s — %s\n", q.title, q.help)
		if q.input != nil {
			def, _ := s.Values[q.input.setting].(string)
			line, ok := read(q.title + " (" + q.input.placeholder + ") [" + def + "]: ")
			if !ok {
				return ended()
			}
			switch {
			case strings.EqualFold(line, "q"):
				fmt.Fprintln(out, "Quit without saving. No configuration was written.")
				return nil
			case strings.EqualFold(line, "b"):
				i, dir = i-1, -1
				continue
			case line == "":
				line = def
			}
			next, err := plainInput(c, s, q, line, out)
			if err != nil {
				fmt.Fprintln(out, "  "+err.Error())
				continue
			}
			s = next
			i, dir = i+1, 1
			continue
		}
		var def string
		if q.multi {
			selected := checked(s, q)
			var nums []string
			for k, o := range q.options {
				mark := " "
				if selected[k] {
					mark = "x"
					nums = append(nums, strconv.Itoa(k+1))
				}
				fmt.Fprintf(out, "  %d. [%s] %s%s\n", k+1, mark, o.label, recommendedNote(o))
			}
			def = strings.Join(nums, " ")
			if def == "" {
				def = "none"
			}
		} else {
			current := currentIndex(s, q)
			if current < 0 {
				current = recommendedIndex(q)
			}
			for k, o := range q.options {
				if o.group != "" && (k == 0 || o.group != q.options[k-1].group) {
					fmt.Fprintf(out, "  %s\n", strings.ToUpper(o.group))
				}
				detail := ""
				if o.detail != "" {
					detail = " — " + o.detail
				}
				fmt.Fprintf(out, "  %d. %s%s%s\n", k+1, o.label, recommendedNote(o), detail)
				if line := pluginLine(o.plugin); line != "" {
					fmt.Fprintf(out, "       %s\n", line)
				}
			}
			def = strconv.Itoa(current + 1)
		}
		prompt := "Choice [" + def + "]: "
		if q.multi {
			prompt = "Numbers separated by spaces, or none [" + def + "]: "
		}
		line, ok := read(prompt)
		if !ok {
			return ended()
		}
		switch strings.ToLower(line) {
		case "q":
			fmt.Fprintln(out, "Quit without saving. No configuration was written.")
			return nil
		case "b":
			i, dir = i-1, -1
			continue
		case "":
			line = def
		}
		var next engine.State
		var err error
		if q.multi {
			next, err = plainMulti(c, s, q, line)
		} else {
			n, convErr := strconv.Atoi(line)
			if convErr != nil || n < 1 || n > len(q.options) {
				err = fmt.Errorf("choose a number from 1 to %d", len(q.options))
			} else {
				next, err = choose(c, s, q.options[n-1])
			}
		}
		if err != nil {
			fmt.Fprintln(out, "  "+err.Error())
			continue
		}
		s = next
		i, dir = i+1, 1
	}

	fmt.Fprintln(out, "\n── Review ──")
	for _, sec := range sections {
		fmt.Fprintf(out, "  %-12s %s\n", sec.title, summary(s, sec))
	}
	fmt.Fprintln(out, "\nChecking the configuration with Neovim…")
	p, err := engine.BuildPlan(context.Background(), ins.Target, s, bin, false)
	if err != nil {
		return err
	}
	fmt.Fprint(out, engine.Summary(p))
	choices := []string{"Save", "Quit without saving"}
	if len(p.Actions) > 0 {
		choices = []string{"Save and download (recommended)", "Save without downloading", "Quit without saving"}
	}
	fmt.Fprintln(out)
	for k, choice := range choices {
		fmt.Fprintf(out, "  %d. %s\n", k+1, choice)
	}
	var download bool
	for {
		line, ok := read("Choice (a number is required): ")
		if !ok {
			return ended()
		}
		n, _ := strconv.Atoi(line)
		if n == len(choices) || strings.EqualFold(line, "q") {
			fmt.Fprintln(out, "Quit without saving. No configuration was written.")
			return nil
		}
		if n >= 1 && n < len(choices) {
			download = len(p.Actions) > 0 && n == 1
			break
		}
	}
	p.RunExternal = download
	p.Digest = p.Fingerprint()
	r, err := engine.Apply(context.Background(), p, engine.ApplyOptions{Neovim: bin, Progress: func(text string) {
		switch {
		case text == "LOCAL_COMMITTED":
			fmt.Fprintln(out, "[OK] Configuration saved.")
		case strings.HasPrefix(text, "["):
			fmt.Fprintln(out, "  "+strings.TrimSpace(text))
		case strings.HasPrefix(text, "wrote:"), text == "backed-up", text == "verified":
		default:
			fmt.Fprintln(out, "  Downloading "+actionName(text)+"…")
		}
	}})
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Backup transaction: %s\nStart Neovim with: %s\n", r.Transaction, launchCommand(ins.Target))
	for _, a := range r.External {
		if a.Status != "installed" {
			return fmt.Errorf("configuration saved; some downloads did not finish (run shoshinvim retry)")
		}
	}
	return nil
}

// plainInput stores typed text; a theme repository is looked up on GitHub first.
func plainInput(c engine.Catalog, s engine.State, q question, value string, out io.Writer) (engine.State, error) {
	if value == "" {
		return s, fmt.Errorf("type %s, b to go back, or q to quit", q.input.placeholder)
	}
	next := engine.Clone(s)
	if !q.input.lookup {
		return next, c.Set(&next, q.input.setting, value, true)
	}
	fmt.Fprintln(out, "  Looking up "+value+" on GitHub…")
	l, err := lookupTheme(context.Background(), value)
	if err != nil {
		return s, err
	}
	themeLookups[l.Repository+"@"+l.Revision] = l
	fmt.Fprintf(out, "  Pinned %s at %s (not reviewed by ShoshinVim).\n", l.Repository, l.Revision[:7])
	if repo, _ := next.Values["appearance.custom_theme.repository"].(string); repo != l.Repository {
		unset(&next, "appearance.custom_theme.colorscheme")
	}
	if err := c.Set(&next, "appearance.custom_theme.repository", l.Repository, true); err != nil {
		return s, err
	}
	return next, c.Set(&next, "appearance.custom_theme.revision", l.Revision, true)
}

func recommendedNote(o option) string {
	if o.recommended {
		return " (recommended)"
	}
	return ""
}

func plainMulti(c engine.Catalog, s engine.State, q question, line string) (engine.State, error) {
	selected := make([]bool, len(q.options))
	if !strings.EqualFold(line, "none") && line != "0" {
		for _, field := range strings.FieldsFunc(line, func(r rune) bool { return r == ' ' || r == ',' }) {
			n, err := strconv.Atoi(field)
			if err != nil || n < 1 || n > len(q.options) {
				return s, fmt.Errorf("use numbers from 1 to %d, or none", len(q.options))
			}
			selected[n-1] = true
		}
	}
	return chooseMany(c, s, q, selected)
}
