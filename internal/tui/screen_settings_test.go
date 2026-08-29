package tui

import (
	"strings"
	"testing"

	workspace "github.com/dhitalkamal/parley/internal/workspace/domain"

	tea "github.com/charmbracelet/bubbletea"
)

func TestOpenSettings_SwitchesScreenAndRemembersPrevious(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.screen = ScreenRequest

	got, _ := m.openSettings()
	if got.screen != ScreenSettings {
		t.Errorf("screen = %v, want ScreenSettings", got.screen)
	}
	if got.previousScreen != ScreenRequest {
		t.Errorf("previousScreen = %v, want ScreenRequest", got.previousScreen)
	}
}

func TestHandleSettingsKey_EscReturnsToPreviousScreen(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.screen = ScreenSettings
	m.previousScreen = ScreenRequest

	next, _ := m.handleSettingsKey(tea.KeyMsg{Type: tea.KeyEsc})
	got := next.(Model)
	if got.screen != ScreenRequest {
		t.Errorf("screen = %v, want ScreenRequest after esc", got.screen)
	}
}

func TestHandleSettingsKey_ToggleOrientationChangesLayout(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.screen = ScreenSettings
	m.orientation = workspace.OrientationVertical

	next, _ := m.handleSettingsKey(tea.KeyMsg{Type: tea.KeyF6})
	got := next.(Model)
	if got.orientation != workspace.OrientationHorizontal {
		t.Errorf("orientation = %v, want Horizontal after toggling", got.orientation)
	}
}

func TestSettingsScreenView_ShowsThemeAndOrientation(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.screen = ScreenSettings
	m.width, m.height = 100, 30

	out := stripANSI(m.settingsScreenView())
	if !strings.Contains(out, "Theme") {
		t.Errorf("expected Theme setting, got:\n%s", out)
	}
	if !strings.Contains(out, "Layout") {
		t.Errorf("expected Layout setting, got:\n%s", out)
	}
}
