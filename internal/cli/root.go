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
	root.PersistentFlags().StringVar(&target, "config-dir", engine.DefaultTarget(), "Neovim configuration target")
	root.PersistentFlags().StringVar(&bin, "nvim", "", "Neovim executable (default: PATH)")
	root.PersistentFlags().StringVar(&output, "output", "text", "text or json")
	root.PersistentFlags().BoolVar(&plain, "no-alt-screen", false, "Use plain terminal history")
	root.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable color")
	root.PersistentFlags().BoolVar(&noProgress, "no-progress", false, "Suppress progress output")
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
	inspect := &cobra.Command{Use: "inspect", Short: "Inspect target and ownership without writing", RunE: func(cmd *cobra.Command, args []string) error {
		r, err := engine.Inspect(target)
		if err != nil {
			return err
		}
		return printResult(cmd, r, string(engine.JSON(r)))
	}}
	doctor := &cobra.Command{Use: "doctor", Short: "Check dependencies and safe generated configuration", RunE: func(cmd *cobra.Command, args []string) error {
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
		}{ins, engine.Dependencies(s), "not run"}
		n, v, e := engine.Neovim(cmd.Context(), bin)
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
				r.Probe = "Safe generated core validated with " + v + "; arbitrary user/plugin Lua was not executed"
			}
		}
		return printResult(cmd, r, string(engine.JSON(r)))
	}}
	var stateFile, outFile, profile string
	var external bool
	plan := &cobra.Command{Use: "plan", Short: "Validate and export an exact file/action plan", RunE: func(cmd *cobra.Command, args []string) error {
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
	plan.Flags().StringVar(&stateFile, "state", "", "Declarative state JSON")
	plan.Flags().StringVar(&outFile, "out", "", "Export to a new plan file")
	plan.Flags().StringVar(&profile, "profile", "", "Propose rich-ide, custom, minimal, web or lua")
	plan.Flags().BoolVar(&external, "include-external", false, "Authorize listed external actions when this exact plan is applied")
	var planFile string
	var yes bool
	apply := &cobra.Command{Use: "apply --plan <file>", Short: "Apply an exact reviewed plan", RunE: func(cmd *cobra.Command, args []string) error {
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
		if err := engine.VerifyPlan(cmd.Context(), p); err != nil {
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
		r, err := engine.Apply(cmd.Context(), p, engine.ApplyOptions{Progress: progress})
		if err != nil {
			return err
		}
		if err = printResult(cmd, r, "Local configuration saved and verified. Restart Neovim.\n"+string(engine.JSON(r))); err != nil {
			return err
		}
		for _, a := range r.External {
			if a.Status != "installed" {
				return fmt.Errorf("local configuration committed; external features remain pending")
			}
		}
		return nil
	}}
	apply.Flags().StringVar(&planFile, "plan", "", "Exact exported plan")
	apply.Flags().BoolVar(&yes, "yes", false, "Confirm this exact plan; safety checks remain mandatory")
	var txn string
	var generated bool
	rollback := &cobra.Command{Use: "rollback", Short: "Review and confirm recovery", RunE: func(cmd *cobra.Command, args []string) error {
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
	rollback.Flags().StringVar(&txn, "transaction", "", "Transaction ID; requires confirmation")
	rollback.Flags().BoolVar(&generated, "generated", false, "Archive drift and restore the last generated version")
	retry := &cobra.Command{Use: "retry", Short: "Retry previously approved failed acquisitions only", RunE: func(cmd *cobra.Command, args []string) error {
		r, err := engine.Retry(cmd.Context(), target, nil)
		if err != nil {
			return err
		}
		return printResult(cmd, r, string(engine.JSON(r)))
	}}
	update := &cobra.Command{Use: "update", Short: "Update using the installation owner", RunE: func(cmd *cobra.Command, args []string) error {
		return distribution.Update(cmd.Context(), version, func(s string) error { return Confirm(cmd.InOrStdin(), cmd.ErrOrStderr(), s) }, cmd.OutOrStdout())
	}}
	uninstall := &cobra.Command{Use: "uninstall", Short: "Uninstall only the receipt-owned application", RunE: func(cmd *cobra.Command, args []string) error {
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
	man := &cobra.Command{Use: "man <directory>", Short: "Generate manual pages", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
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
