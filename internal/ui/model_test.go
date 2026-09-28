package ui

import (
	"bytes"
	tea "charm.land/bubbletea/v2"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnknownAndProposal(t *testing.T) {
	m, err := New(filepath.Join(t.TempDir(), "nvim"), "", true)
	if err != nil {
		t.Fatal(err)
	}
	m.Submit("make it magical")
	if m.dirty() {
		t.Fatal("free text mutated")
	}
	m.Submit("/profile web")
	if m.dirty() {
		t.Fatal("proposal mutated")
	}
	m.Submit("1")
	if !m.dirty() {
		t.Fatal("no accepted changes")
	}
	m.Submit("/undo")
	if m.dirty() {
		t.Fatal("undo")
	}
}
func TestResizePreservesComposer(t *testing.T) {
	m, _ := New(filepath.Join(t.TempDir(), "nvim"), "", true)
	m.input.SetValue("/set editor.wrap off")
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	if !strings.Contains(m.View().Content, "Resize terminal") {
		t.Fatal("no blocker")
	}
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if m.input.Value() != "/set editor.wrap off" {
		t.Fatal("lost input")
	}
}
func TestPlainMode(t *testing.T) {
	var out bytes.Buffer
	err := RunPlain(filepath.Join(t.TempDir(), "nvim"), "", strings.NewReader("/set editor.wrap off\n/undo\n/quit\n"), &out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Changed editor.wrap") {
		t.Fatal(out.String())
	}
	if strings.Contains(out.String(), "\x1b") {
		t.Fatal("escape sequences")
	}
}

func TestRichIDEOnboardingKeepsDraftUntilAcceptance(t *testing.T) {
	m, _ := New(filepath.Join(t.TempDir(), "nvim"), "", true)
	for _, answer := range []string{"1", "2", "2"} {
		m.Submit(answer)
		if m.dirty() {
			t.Fatal("wizard changed state before acceptance")
		}
	}
	m.Submit("1")
	if m.draft.Profile != "rich-ide" || m.draft.Values["formatting.on_save"] != true || m.draft.Values["language.lua.lsp"] != false {
		t.Fatalf("wrong choices: %+v", m.draft)
	}
	if m.prompt != "next" {
		t.Fatal("missing obvious next step")
	}
	m.Submit("/undo")
	if m.dirty() {
		t.Fatal("could not undo wizard")
	}
}
