package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/shoshindev/shoshinvim/internal/engine"
)

func TestApplyRequiresExactPlan(t *testing.T) {
	c := New("test")
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&out)
	c.SetArgs([]string{"apply", "--yes"})
	if err := c.Execute(); err == nil {
		t.Fatal("accepted yes without plan")
	}
}
func TestPlanDoesNotWriteTargetAndRejectsChangedDestination(t *testing.T) {
	if _, err := exec.LookPath("nvim"); err != nil {
		t.Skip("Neovim required")
	}
	temp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(temp, "preferences"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(temp, "data"))
	target := filepath.Join(temp, "target")
	plan := filepath.Join(temp, "plan.json")
	run := func(args ...string) error {
		c := New("test")
		var out bytes.Buffer
		c.SetOut(&out)
		c.SetErr(&out)
		c.SetArgs(args)
		return c.Execute()
	}
	if err := run("--config-dir", target, "plan", "--profile", "minimal", "--out", plan); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("planning wrote target")
	}
	if err := run("--config-dir", filepath.Join(temp, "other"), "apply", "--plan", plan, "--yes"); err == nil {
		t.Fatal("accepted different explicit target")
	}
	if err := run("apply", "--plan", plan, "--yes"); err != nil {
		t.Fatal(err)
	}
	if err := run("apply", "--plan", plan, "--yes"); err == nil {
		t.Fatal("accepted stale plan")
	}
}

func TestTextOutputIsReadable(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(temp, "preferences"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(temp, "data"))
	target := filepath.Join(temp, "target")
	run := func(args ...string) string {
		c := New("test")
		var out bytes.Buffer
		c.SetOut(&out)
		c.SetErr(&out)
		c.SetArgs(args)
		if err := c.Execute(); err != nil {
			t.Fatal(err)
		}
		return out.String()
	}
	for _, command := range []string{"inspect", "doctor"} {
		text := run("--config-dir", target, command)
		if strings.HasPrefix(strings.TrimSpace(text), "{") || !strings.Contains(text, "Target  ") || !strings.Contains(text, "Status  new") {
			t.Fatalf("%s prints no readable text:\n%s", command, text)
		}
		var v map[string]any
		if err := json.Unmarshal([]byte(run("--config-dir", target, "--output", "json", command)), &v); err != nil {
			t.Fatalf("%s --output json is not JSON: %v", command, err)
		}
	}
	if help := run("--help"); strings.Contains(help, "Generate manual pages") {
		t.Fatalf("the release-only man command is listed:\n%s", help)
	}
}

func TestDoctorReportsTheChosenNeovim(t *testing.T) {
	nvim, err := exec.LookPath("nvim")
	if err != nil {
		t.Skip("Neovim required")
	}
	if nvim, err = filepath.Abs(nvim); err != nil {
		t.Fatal(err)
	}
	temp := t.TempDir()
	t.Setenv("PATH", filepath.Join(temp, "empty")) // Only --nvim names Neovim.
	if d := doctorNeovim(t, temp, "--nvim", nvim); d.Status != "present" || d.Path != nvim {
		t.Fatalf("doctor reports %+v for the Neovim given with --nvim (%s)", d, nvim)
	}
}

func TestDoctorReportsAnUnusableChosenNeovim(t *testing.T) {
	temp := t.TempDir()
	// PATH has a Neovim, but --nvim names one that does not exist.
	bin := filepath.Join(temp, "bin")
	name := "nvim"
	if runtime.GOOS == "windows" {
		name = "nvim.bat"
	}
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, name), []byte("exit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	if d := doctorNeovim(t, temp, "--nvim", filepath.Join(temp, "missing", "nvim")); d.Status != "missing" || d.Path != "" {
		t.Fatalf("doctor reports %+v for a --nvim that does not exist", d)
	}
}

// doctorNeovim runs doctor --output json for a new target in temp and returns its nvim row.
func doctorNeovim(t *testing.T, temp string, args ...string) engine.Dependency {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(temp, "preferences"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(temp, "data"))
	c := New("test")
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&out)
	c.SetArgs(append(append([]string{"--config-dir", filepath.Join(temp, "target"), "--output", "json"}, args...), "doctor"))
	if err := c.Execute(); err != nil {
		t.Fatal(err)
	}
	var r struct{ Dependencies []engine.Dependency }
	if err := json.Unmarshal(out.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	for _, d := range r.Dependencies {
		if d.ID == "nvim" {
			return d
		}
	}
	t.Fatal("doctor reports no Neovim")
	return engine.Dependency{}
}

func TestApplyPrintsAWorkingUndoCommand(t *testing.T) {
	if _, err := exec.LookPath("nvim"); err != nil {
		t.Skip("Neovim required")
	}
	temp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(temp, "preferences"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(temp, "data"))
	t.Setenv("SHOSHINVIM_TARGET_DIR", "")
	// Not the default target, which lies under XDG_CONFIG_HOME; the space needs quoting.
	target := filepath.Join(temp, "my config")
	plan := filepath.Join(temp, "plan.json")
	run := func(in string, args ...string) string {
		c := New("test")
		var out bytes.Buffer
		c.SetIn(strings.NewReader(in))
		c.SetOut(&out)
		c.SetErr(&out)
		c.SetArgs(args)
		if err := c.Execute(); err != nil {
			t.Fatalf("%q: %v\n%s", args, err, out.String())
		}
		return out.String()
	}
	run("", "--config-dir", target, "plan", "--profile", "minimal", "--out", plan)
	text := run("", "apply", "--plan", plan, "--yes")
	_, undo, _ := strings.Cut(text, "  shoshinvim rollback --config-dir ")
	dir, id, ok := strings.Cut(strings.TrimSpace(undo), " --transaction ")
	if canonical, err := engine.Canonical(target); err != nil || !ok || dir != shellArg(runtime.GOOS, canonical) {
		t.Fatalf("apply names no undo command for %s:\n%s", target, text)
	}
	run("yes\n", "rollback", "--config-dir", target, "--transaction", id)
	if ins, err := engine.Inspect(target); err != nil || ins.Ownership != "NEW" {
		t.Fatalf("the undo command left %+v, %v", ins, err)
	}
}

func TestShellArg(t *testing.T) {
	for _, c := range []struct{ goos, in, want string }{
		{"darwin", "/Users/me/.config/nvim", "/Users/me/.config/nvim"},
		{"linux", "/tmp/my config/it's", `'/tmp/my config/it'\''s'`},
		{"windows", `C:\Users\me\AppData\Local\nvim`, `C:\Users\me\AppData\Local\nvim`},
		{"windows", `C:\Users\RUNNER~1\it's`, `'C:\Users\RUNNER~1\it''s'`},
	} {
		if got := shellArg(c.goos, c.in); got != c.want {
			t.Errorf("shellArg(%s, %q) = %s, want %s", c.goos, c.in, got, c.want)
		}
	}
	if runtime.GOOS == "windows" {
		return
	}
	for _, s := range []string{"/tmp/my config/it's", "/tmp/$HOME/*/`x`/a\\b\"c"} {
		if out, err := exec.Command("sh", "-c", "printf %s "+shellArg(runtime.GOOS, s)).Output(); err != nil || string(out) != s {
			t.Errorf("sh reads %s as %q (%v), want %q", shellArg(runtime.GOOS, s), out, err, s)
		}
	}
}
