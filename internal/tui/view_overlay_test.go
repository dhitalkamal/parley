package tui

import (
	"strings"
	"testing"
)

func TestView_ConfirmDialogFloatsOverStillVisibleBackground(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 80, 24
	m.screen = ScreenRequest
	m.confirm.Open(confirmQuit, "Quit parley?", "")

	got := stripANSI(m.View())

	if !strings.Contains(got, "Quit parley?") {
		t.Errorf("expected the confirm dialog text in the view, got:\n%s", got)
	}
	if !strings.Contains(got, "Request") {
		t.Errorf("expected the request zone (background) still visible behind the dialog, got:\n%s", got)
	}
}

func TestView_PromptDialogFloatsOverStillVisibleBackground(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 80, 24
	m.screen = ScreenRequest
	m.prompt.Open(promptNewRequest, "", "New request name", "")

	got := stripANSI(m.View())

	if !strings.Contains(got, "New request name") {
		t.Errorf("expected the prompt dialog text in the view, got:\n%s", got)
	}
	if !strings.Contains(got, "Request") {
		t.Errorf("expected the request zone (background) still visible behind the dialog, got:\n%s", got)
	}
}
