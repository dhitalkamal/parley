package tui

import "testing"

func TestEnvPanel_OpenSetsActiveAndDefaultsToGlobals(t *testing.T) {
	var p envPanelState
	p.Open([]string{"staging", "prod"})
	if !p.active {
		t.Fatal("expected panel to be active after Open")
	}
	if !p.IsGlobalsSelected() {
		t.Error("expected Globals to be selected by default")
	}
	if p.ScopeName() != "" {
		t.Errorf("got scope name %q, want empty string for Globals", p.ScopeName())
	}
}

func TestEnvPanel_Close(t *testing.T) {
	var p envPanelState
	p.Open([]string{"staging"})
	p.Close()
	if p.active {
		t.Error("expected panel inactive after Close")
	}
}

func TestEnvPanel_MoveCursorSelectsNamedEnvironments(t *testing.T) {
	var p envPanelState
	p.Open([]string{"staging", "prod"})

	p.MoveCursor(1)
	if p.IsGlobalsSelected() {
		t.Fatal("expected globals no longer selected")
	}
	if p.ScopeName() != "staging" {
		t.Errorf("got %q, want staging", p.ScopeName())
	}

	p.MoveCursor(1)
	if p.ScopeName() != "prod" {
		t.Errorf("got %q, want prod", p.ScopeName())
	}
}

func TestEnvPanel_MoveCursorClampsAtBounds(t *testing.T) {
	var p envPanelState
	p.Open([]string{"staging"})

	p.MoveCursor(-1)
	if !p.IsGlobalsSelected() {
		t.Error("expected cursor clamped at Globals (top), not wrapped past it")
	}

	p.MoveCursor(5)
	if p.ScopeName() != "staging" {
		t.Errorf("got %q, want clamped at the last environment (bottom)", p.ScopeName())
	}
}

func TestEnvPanel_SetEnvNamesClampsCursorWhenListShrinks(t *testing.T) {
	var p envPanelState
	p.Open([]string{"staging", "prod"})
	p.MoveCursor(2) // select "prod", the last one

	p.SetEnvNames([]string{"staging"})

	if p.ScopeName() != "staging" {
		t.Errorf("got %q, want cursor clamped onto the remaining environment", p.ScopeName())
	}
}

func TestEnvPanel_ToggleFocusMovesBetweenScopesAndVars(t *testing.T) {
	var p envPanelState
	p.Open(nil)
	if p.focusVars {
		t.Fatal("expected scope list focused on open")
	}

	p.ToggleFocus()
	if !p.focusVars {
		t.Error("expected vars focused after toggle")
	}

	p.ToggleFocus()
	if p.focusVars {
		t.Error("expected scopes focused after toggling back")
	}
}
