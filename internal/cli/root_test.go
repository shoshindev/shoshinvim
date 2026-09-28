package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
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
