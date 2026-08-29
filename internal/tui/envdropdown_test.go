package tui

import (
	"strings"
	"testing"
)

func TestEnvDropdown_OpenSetsActiveAndDefaultsToFirstEntry(t *testing.T) {
	var d envDropdownState
	d.Open([]string{"staging", "prod"})
	if !d.active {
		t.Fatal("expected dropdown active after Open")
	}
	if d.IsManageSelected() {
		t.Error("expected the first environment selected by default, not Manage")
	}
	if d.SelectedName() != "staging" {
		t.Errorf("got %q, want staging", d.SelectedName())
	}
}

// TestEnvDropdown_ViewShowsATitle guards the overlay polish pass: every
// other overlay in this app titles itself (History, Runner, Palette,
// Manage Environments); this dropdown was the one exception, rendering as
// an untitled bordered list.
func TestEnvDropdown_ViewShowsATitle(t *testing.T) {
	var d envDropdownState
	d.Open([]string{"staging"})

	got := stripANSI(d.View(""))
	if !strings.Contains(got, "Environments") {
		t.Errorf("View() = %q, want an \"Environments\" title", got)
	}
}

func TestEnvDropdown_Close(t *testing.T) {
	var d envDropdownState
	d.Open([]string{"staging"})
	d.Close()
	if d.active {
		t.Error("expected dropdown inactive after Close")
	}
}

func TestEnvDropdown_MoveCursorReachesManageEntryPastLastEnv(t *testing.T) {
	var d envDropdownState
	d.Open([]string{"staging", "prod"})

	d.MoveCursor(1) // prod
	if d.SelectedName() != "prod" {
		t.Fatalf("got %q, want prod", d.SelectedName())
	}

	d.MoveCursor(1) // past the last env -> Manage
	if !d.IsManageSelected() {
		t.Error("expected cursor to land on \"Manage Environments...\" after the last env")
	}
}

func TestEnvDropdown_MoveCursorClampsAtBounds(t *testing.T) {
	var d envDropdownState
	d.Open([]string{"staging"})

	d.MoveCursor(-1)
	if d.SelectedName() != "staging" {
		t.Error("expected cursor clamped at the top, not wrapped past it")
	}

	d.MoveCursor(5)
	if !d.IsManageSelected() {
		t.Error("expected cursor clamped at Manage (the bottom), not wrapped past it")
	}
}

func TestEnvDropdown_OpenWithNoEnvironmentsStartsOnManage(t *testing.T) {
	var d envDropdownState
	d.Open(nil)
	if !d.IsManageSelected() {
		t.Error("expected Manage selected by default when there are no environments yet")
	}
}

func TestEnvDropdown_ViewShowsEveryEnvAndManageEntry(t *testing.T) {
	var d envDropdownState
	d.Open([]string{"staging", "prod"})
	view := stripANSI(d.View("prod"))

	for _, want := range []string{"staging", "prod", "Manage Environments"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q:\n%s", want, view)
		}
	}
}
