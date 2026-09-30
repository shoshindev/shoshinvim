package ui

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/shoshindev/shoshinvim/internal/engine"
)

// previewDoneMsg reports that the Neovim preview has closed.
type previewDoneMsg struct {
	dir string
	err error
}

// previewLua opens a read-only copy of the generated files in Neovim, one tab per
// file, with Neovim's own syntax colors and movement. --clean keeps the user's
// configuration and plugins out of it, and the target is never touched; :qa returns
// to the setup and the copy is removed. Without a usable Neovim, the files are shown
// in the viewer instead.
func (m *Model) previewLua() tea.Cmd {
	files, err := engine.Generate(m.draft)
	if err != nil {
		m.flash(err.Error(), true)
		return nil
	}
	nvim, _, err := engine.Neovim(context.Background(), m.bin)
	if err != nil {
		m.showLua()
		return nil
	}
	dir, names, err := writePreview(files)
	if err != nil {
		m.flash("Could not prepare the preview: "+err.Error(), true)
		return nil
	}
	return tea.ExecProcess(previewCommand(nvim, dir, names), func(err error) tea.Msg { return previewDoneMsg{dir, err} })
}

// writePreview copies the generated files into a new temporary folder and returns
// their relative paths, init.lua first.
func writePreview(files map[string]string) (dir string, names []string, err error) {
	dir, err = os.MkdirTemp("", "shoshinvim-preview-")
	if err != nil {
		return "", nil, err
	}
	names = []string{"init.lua"}
	for _, name := range sortedKeys(files) {
		if name != "init.lua" {
			names = append(names, name)
		}
	}
	for _, name := range names {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err = os.MkdirAll(filepath.Dir(p), 0o700); err == nil {
			err = os.WriteFile(p, []byte(files[name]), 0o600)
		}
		if err != nil {
			os.RemoveAll(dir)
			return "", nil, err
		}
	}
	return dir, names, nil
}

func previewCommand(nvim, dir string, names []string) *exec.Cmd {
	args := append([]string{"--clean", "-R", "-n", "-p"}, names...)
	args = append(args, "-c", "echo 'ShoshinVim preview, read-only · gt/gT switch files · :qa returns to the setup'")
	cmd := exec.Command(nvim, args...)
	cmd.Dir = dir
	return cmd
}

func (m *Model) previewDone(msg previewDoneMsg) {
	os.RemoveAll(msg.dir)
	if msg.err != nil {
		m.flash("Neovim closed with an error: "+msg.err.Error(), true)
	}
}
