package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/shoshindev/shoshinvim/internal/distribution"
	"github.com/shoshindev/shoshinvim/internal/engine"
	"github.com/shoshindev/shoshinvim/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

type Preferences struct {
	Neovim      string `toml:"neovim"`
	NoColor     bool   `toml:"no_color"`
	NoAltScreen bool   `toml:"no_alt_screen"`
}

func preferences() (Preferences, error) {
	var p Preferences
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		var err error
		dir, err = os.UserConfigDir()
		if err != nil {
			return p, err
		}
		if home, e := os.UserHomeDir(); e == nil && strings.HasSuffix(dir, "Library/Application Support") {
			dir = filepath.Join(home, ".config")
		}
	}
	b, err := os.ReadFile(filepath.Join(dir, "shoshinvim", "config.toml"))
	if os.IsNotExist(err) {
		return p, nil
	}
	if err != nil {
		return p, err
	}
	err = toml.Unmarshal(b, &p)
	return p, err
}
func Confirm(in io.Reader, out io.Writer, text string) error {
	fmt.Fprintln(out, text+"\nType yes to confirm; anything else cancels.")
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && len(line) == 0 {
		return fmt.Errorf("confirmation required")
	}
	if strings.TrimSpace(line) != "yes" {
		return fmt.Errorf("cancelled; no action taken")
	}
	return nil
}
func writeNew(path string, b []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	e := f.Close()
	if err != nil {
		return err
	}
	return e
}

func New(version string) *cobra.Command {
	var target, bin, output string
	var plain, noColor, noProgress bool
	root := &cobra.Command{Use: "shoshinvim", Short: "Build your Neovim from the terminal", Version: version, SilenceUsage: true, SilenceErrors: true}
	root.PersistentFlags().StringVar(&target, "config-dir", engine.DefaultTarget(), "Neovim configuration folder")
	root.PersistentFlags().StringVar(&bin, "nvim", "", "Neovim to use (default: the first nvim on PATH)")
	root.PersistentFlags().StringVar(&output, "output", "text", "text or json")
	root.PersistentFlags().BoolVar(&plain, "no-alt-screen", false, "Ask questions in ordinary terminal output instead of full screen")
	root.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable color")
	root.PersistentFlags().BoolVar(&noProgress, "no-progress", false, "Hide progress output")
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if output != "text" && output != "json" {
			return fmt.Errorf("output must be text or json")
		}
		p, err := preferences()
		if err != nil {
			return err
		}
		if !cmd.Flags().Changed("nvim") && bin == "" {
			bin = p.Neovim
		}
		noColor = noColor || p.NoColor || os.Getenv("NO_COLOR") != ""
		plain = plain || p.NoAltScreen
		return nil
	}
	root.RunE = func(cmd *cobra.Command, args []string) error {
		if output == "json" {
			return fmt.Errorf("choose inspect, doctor, plan or apply for JSON output")
		}
		if plain {
			return ui.RunPlain(target, bin, cmd.InOrStdin(), cmd.OutOrStdout())
		}
		return ui.Run(target, bin, noColor)
	}
	printResult := func(cmd *cobra.Command, v any, text string) error {
		if output == "json" {
			_, err := cmd.OutOrStdout().Write(engine.JSON(v))
			return err
		}
		_, err := fmt.Fprintln(cmd.OutOrStdout(), text)
		return err
	}
	inspect := &cobra.Command{Use: "inspect", Short: "Show the configuration folder and whether ShoshinVim created it", RunE: func(cmd *cobra.Command, args []string) error {
		r, err := engine.Inspect(target)
		if err != nil {
			return err
		}
		return printResult(cmd, r, inspectionText(r))
	}}
	doctor := &cobra.Command{Use: "doctor", Short: "Check the programs your configuration needs and test its generated Lua", RunE: func(cmd *cobra.Command, args []string) error {
		ins, err := engine.Inspect(target)
		if err != nil {
			return err
		}
		s := engine.NewState()
		if ins.State != nil {
			s = *ins.State
		}
		r := struct {
			Inspection   engine.Inspection   `json:"inspection"`
			Dependencies []engine.Dependency `json:"dependencies"`
			Probe        string              `json:"probe"`
		}{ins, engine.Dependencies(s, ins.Target), "not run"}
		probed := false
		n, v, e := engine.Neovim(cmd.Context(), bin)
		r.Dependencies = engine.WithNeovim(r.Dependencies, n, e)
		if e != nil {
			r.Probe = e.Error()
		} else {
			files, e := engine.Generate(s)
			if e == nil {
				e = engine.Probe(cmd.Context(), n, files)
			}
			if e != nil {
				r.Probe = e.Error()
			} else {
				r.Probe, probed = "Safe generated core validated with "+v+"; arbitrary user/plugin Lua was not executed", true
			}
		}
		return printResult(cmd, r, doctorText(r.Inspection, r.Dependencies, r.Probe, probed))
	}}
	var stateFile, outFile, profile string
	var external bool
	plan := &cobra.Command{Use: "plan", Short: "Check a configuration and list its files and downloads, for scripts", RunE: func(cmd *cobra.Command, args []string) error {
		s := engine.NewState()
		if stateFile != "" {
			if err := engine.DecodeFile(stateFile, &s); err != nil {
				return err
			}
		} else {
			ins, err := engine.Inspect(target)
			if err != nil {
				return err
			}
			if ins.State != nil {
				s = *ins.State
			}
		}
		if profile != "" {
			var err error
			s, err = engine.LoadCatalog().Profile(s, profile)
			if err != nil {
				return err
			}
		}
		p, err := engine.BuildPlan(cmd.Context(), target, s, bin, external)
		if err != nil {
			return err
		}
		if outFile != "" {
			if err = writeNew(outFile, engine.JSON(p)); err != nil {
				return err
			}
		}
		return printResult(cmd, p, engine.Summary(p))
	}}
	plan.Flags().StringVar(&stateFile, "state", "", "Read your choices from this JSON file")
	plan.Flags().StringVar(&outFile, "out", "", "Save the plan to this new file")
	plan.Flags().StringVar(&profile, "profile", "", "Start from a preset: rich-ide, custom, minimal, web or lua")
	plan.Flags().BoolVar(&external, "include-external", false, "Allow the downloads this plan lists when it is applied")
	var planFile string
	var yes bool
	apply := &cobra.Command{Use: "apply --plan <file>", Short: "Save a configuration from a plan made by plan --out", RunE: func(cmd *cobra.Command, args []string) error {
		if planFile == "" {
			return fmt.Errorf("--plan is required")
		}
		var p engine.Plan
		if err := engine.DecodeFile(planFile, &p); err != nil {
			return err
		}
		if cmd.Flags().Changed("config-dir") {
			t, err := engine.Canonical(target)
			if err != nil {
				return err
			}
			if t != p.Target {
				return fmt.Errorf("explicit target does not match plan")
			}
		}
		if _, err := engine.VerifyPlan(cmd.Context(), p, bin); err != nil {
			return err
		}
		if !yes {
			if err := Confirm(cmd.InOrStdin(), cmd.ErrOrStderr(), engine.Summary(p)); err != nil {
				return err
			}
		}
		var progress func(string)
		if !noProgress && output != "json" {
			progress = func(s string) { fmt.Fprintln(cmd.ErrOrStderr(), "[..] "+s) }
		}
		r, err := engine.Apply(cmd.Context(), p, engine.ApplyOptions{Progress: progress, Neovim: bin})
		if err != nil {
			return err
		}
		if err = printResult(cmd, r, applyText(p.Target, r)); err != nil {
			return err
		}
		for _, a := range r.External {
			if a.Status != "installed" {
				return fmt.Errorf("local configuration committed; external features remain pending")
			}
		}
		return nil
	}}
	apply.Flags().StringVar(&planFile, "plan", "", "Plan file made by plan --out")
	apply.Flags().BoolVar(&yes, "yes", false, "Skip the confirmation; the safety checks still run")
	var txn string
	var generated bool
	rollback := &cobra.Command{Use: "rollback", Short: "List earlier saves, or restore one", RunE: func(cmd *cobra.Command, args []string) error {
		t, err := engine.Canonical(target)
		if err != nil {
			return err
		}
		js, err := engine.Journals(t)
		if err != nil {
			return err
		}
		if txn == "" {
			return printResult(cmd, js, "Select --transaction <id> from:\n"+journalList(js))
		}
		for _, j := range js {
			if j.ID == txn {
				if err = Confirm(cmd.InOrStdin(), cmd.ErrOrStderr(), engine.Summary(j.Plan)+"\nRestore transaction "+txn); err != nil {
					return err
				}
				if generated {
					err = engine.RecoverGenerated(t, txn)
				} else {
					err = engine.Rollback(t, txn)
				}
				if err != nil {
					return err
				}
				return printResult(cmd, map[string]string{"status": "restored"}, "Recovery completed.")
			}
		}
		return fmt.Errorf("unknown transaction")
	}}
	rollback.Flags().StringVar(&txn, "transaction", "", "ID of the save to restore; asks before restoring")
	rollback.Flags().BoolVar(&generated, "generated", false, "Restore the files that save wrote, moving hand edits to .shoshinvim/recovery first")
	retry := &cobra.Command{Use: "retry", Short: "Retry downloads that failed", RunE: func(cmd *cobra.Command, args []string) error {
		r, err := engine.Retry(cmd.Context(), target, bin, nil)
		if err != nil {
			return err
		}
		return printResult(cmd, r, actionsText(r))
	}}
	update := &cobra.Command{Use: "update", Short: "Install a newer ShoshinVim release", RunE: func(cmd *cobra.Command, args []string) error {
		return distribution.Update(cmd.Context(), version, func(s string) error { return Confirm(cmd.InOrStdin(), cmd.ErrOrStderr(), s) }, cmd.OutOrStdout())
	}}
	uninstall := &cobra.Command{Use: "uninstall", Short: "Remove the shoshinvim executable the installer put in place", RunE: func(cmd *cobra.Command, args []string) error {
		return distribution.Uninstall(func(s string) error { return Confirm(cmd.InOrStdin(), cmd.ErrOrStderr(), s) }, cmd.OutOrStdout())
	}}
	completion := &cobra.Command{Use: "completion [bash|zsh|fish|powershell]", Short: "Generate shell completions", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return root.GenBashCompletionV2(cmd.OutOrStdout(), true)
		case "zsh":
			return root.GenZshCompletion(cmd.OutOrStdout())
		case "fish":
			return root.GenFishCompletion(cmd.OutOrStdout(), true)
		case "powershell":
			return root.GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
		default:
			return fmt.Errorf("unsupported shell")
		}
	}}
	// Release builds run man to generate the manual pages; it is not listed for users.
	man := &cobra.Command{Use: "man <directory>", Short: "Generate manual pages", Hidden: true, Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if err := os.MkdirAll(args[0], 0755); err != nil {
			return err
		}
		return doc.GenManTree(root, &doc.GenManHeader{Title: "SHOSHINVIM", Section: "1"}, args[0])
	}}
	root.AddCommand(inspect, doctor, plan, apply, rollback, retry, update, uninstall, completion, man)
	root.SetContext(context.Background())
	return root
}
func journalList(js []engine.Journal) string {
	var b strings.Builder
	for _, j := range js {
		fmt.Fprintf(&b, "%s  %s\n", j.ID, j.Status)
	}
	return b.String()
}
