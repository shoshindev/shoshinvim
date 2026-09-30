package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(temp, "preferences"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(temp, "data"))
	t.Setenv("PATH", filepath.Join(temp, "empty")) // Only --nvim names Neovim.
	c := New("test")
	var out bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&out)
	c.SetArgs([]string{"--config-dir", filepath.Join(temp, "target"), "--nvim", nvim, "--output", "json", "doctor"})
	if err := c.Execute(); err != nil {
		t.Fatal(err)
	}
	var r struct {
		Dependencies []struct{ ID, Status, Path string }
	}
	if err := json.Unmarshal(out.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	for _, d := range r.Dependencies {
		if d.ID == "nvim" && (d.Status != "present" || d.Path != nvim) {
			t.Fatalf("doctor reports %+v for the Neovim given with --nvim (%s)", d, nvim)
		}
	}
}
