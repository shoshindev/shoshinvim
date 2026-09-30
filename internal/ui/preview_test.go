package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/shoshindev/shoshinvim/internal/engine"
)

func TestPreviewOpensEveryFileReadOnlyInNeovim(t *testing.T) {
	needNeovim(t)
	m := newModel(t)
	files, err := engine.Generate(m.draft)
	if err != nil {
		t.Fatal(err)
	}
	dir, names, err := writePreview(files)
	if err != nil {
		t.Fatal(err)
	}
	if names[0] != "init.lua" || len(names) != len(files) {
		t.Fatalf("preview files %v, want init.lua first and all %d files", names, len(files))
	}
	for _, name := range names {
		if b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name))); err != nil || string(b) != files[name] {
			t.Fatalf("%s was not copied: %v", name, err)
		}
	}

	cmd := previewCommand("nvim", dir, names)
	if cmd.Dir != dir || !slices.Contains(cmd.Args, "--clean") || !slices.Contains(cmd.Args, "-R") || !slices.Contains(cmd.Args, "-p") {
		t.Fatalf("preview is not a clean, read-only Neovim in the copy: %v in %s", cmd.Args, cmd.Dir)
	}
	// Run the same command headless: one tab per file, each read-only Lua.
	nvim, _ := exec.LookPath("nvim")
	args := append([]string{"--headless"}, cmd.Args[1:]...)
	args = append(args, "-c", `lua local t = {} for _, tp in ipairs(vim.api.nvim_list_tabpages()) do local b = vim.api.nvim_win_get_buf(vim.api.nvim_tabpage_get_win(tp)); t[#t + 1] = vim.bo[b].filetype .. (vim.bo[b].readonly and ':ro' or ':rw') end io.stdout:write('TABS ' .. table.concat(t, ' ') .. '\n')`, "-c", "qa!")
	run := exec.Command(nvim, args...)
	run.Dir = dir
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	_, tabs, _ := strings.Cut(string(out), "TABS ")
	if got := strings.Fields(tabs); len(got) != len(names) || slices.ContainsFunc(got, func(s string) bool { return s != "lua:ro" }) {
		t.Fatalf("tabs %v, want %d read-only Lua files", got, len(names))
	}

	m.Update(previewDoneMsg{dir: dir})
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("the preview copy was not removed after Neovim closed")
	}
}

func TestPreviewCommandsOpenNeovim(t *testing.T) {
	needNeovim(t)
	tmp := t.TempDir() // The copy lands here; the command itself is not run.
	t.Setenv("TMPDIR", tmp)
	t.Setenv("TMP", tmp)
	t.Setenv("TEMP", tmp)
	m := newModel(t)
	if cmd := m.runCommand("lua"); cmd == nil || m.viewerOpen {
		t.Fatal("the command menu's preview did not hand over to Neovim")
	}
	acceptAll(t, m)
	if cmd := m.activate("lua"); cmd == nil || m.viewerOpen {
		t.Fatal("the review's preview did not hand over to Neovim")
	}
}
