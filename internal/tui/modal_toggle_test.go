package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// These four guard a consistency gap a user reported ("modals/dropdowns
// behave unexpectedly"): showVariables/showCodeSnippet/history already let
// their own open hotkey also close them, but envDropdown/envPanel/palette/
// runner were esc-only despite having a dedicated hotkey - no functional
// reason for the difference.

func TestEnvDropdownKey_OwnHotkeyCloses(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.openEnvDropdown()
	m, _ = m.handleEnvDropdownKey(tea.KeyMsg{Type: tea.KeyCtrlE})
	if m.envDropdown.active {
		t.Error("ctrl+e should close the env dropdown when it's already open")
	}
}

func TestEnvPanelKey_OwnHotkeyCloses(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.openEnvPanel()
	m, _ = m.handleEnvPanelKey(tea.KeyMsg{Type: tea.KeyCtrlE})
	if m.envPanel.active {
		t.Error("ctrl+e should close the env panel when it's already open")
	}
}

func TestEnvPanelKey_OwnHotkeyDoesNotCloseMidEdit(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.openEnvPanel()
	m.envPanel.editing = true
	m, _ = m.handleEnvPanelKey(tea.KeyMsg{Type: tea.KeyCtrlE})
	if !m.envPanel.active {
		t.Error("ctrl+e should not discard an in-progress edit by closing the panel")
	}
}

func TestPaletteKey_OwnHotkeyCloses(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.palette.Open()
	m, _ = m.handlePaletteKey(tea.KeyMsg{Type: tea.KeyCtrlK})
	if m.palette.active {
		t.Error("ctrl+k should close the palette when it's already open")
	}
}

func TestRunnerKey_OwnHotkeyCloses(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.runner.active = true
	m, _ = m.handleRunnerKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}})
	if m.runner.active {
		t.Error("R should close the runner modal when it's already open")
	}
}
