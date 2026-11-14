package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func sizedModel(t *testing.T) Model {
	t.Helper()
	m := New(t.TempDir(), t.TempDir())
	next, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	return next.(Model)
}

func TestStartupNotice_ShownOnStatusRowUntilFirstKey(t *testing.T) {
	m := sizedModel(t)
	m.SetStartupNotice("parley: warning: could not restrict permissions on 1 file(s): boom")

	lines := strings.Split(m.View(), "\n")
	if last := lines[len(lines)-1]; !strings.Contains(last, "could not restrict permissions") {
		t.Fatalf("status row = %q, want the startup warning", last)
	}

	next, _ := m.Update(rune1('j'))
	lines = strings.Split(next.(Model).View(), "\n")
	if last := lines[len(lines)-1]; strings.Contains(last, "could not restrict permissions") {
		t.Errorf("warning still shown after a keypress: %q", last)
	}
}

func TestStartupNotice_AbsentByDefault(t *testing.T) {
	m := sizedModel(t)
	if strings.Contains(m.View(), "could not restrict") {
		t.Error("unexpected warning on a fresh model")
	}
}
