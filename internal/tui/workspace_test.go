package tui

import (
	workspace "github.com/dhitalkamal/parley/internal/workspace/domain"
	"strings"
	"testing"
)

func sampleWorkspaces() []workspace.Workspace {
	return []workspace.Workspace{
		{Name: "Personal", CollectionsRoot: "/a/collections", ProjectRoot: "/a"},
		{Name: "TeamX", CollectionsRoot: "/b/collections", ProjectRoot: "/b"},
	}
}

func TestWorkspaceState_OpenSelectsTheActiveWorkspace(t *testing.T) {
	var p workspaceState
	p.Open(sampleWorkspaces(), "TeamX")
	if !p.active {
		t.Fatal("expected active after Open")
	}
	got, ok := p.Selected()
	if !ok || got.Name != "TeamX" {
		t.Errorf("got %+v, want TeamX selected (it's the active workspace)", got)
	}
}

func TestWorkspaceState_OpenDefaultsToFirstWhenActiveNameNotFound(t *testing.T) {
	var p workspaceState
	p.Open(sampleWorkspaces(), "DoesNotExist")
	got, ok := p.Selected()
	if !ok || got.Name != "Personal" {
		t.Errorf("got %+v, want Personal (first entry) when the active name isn't in the list", got)
	}
}

func TestWorkspaceState_Close(t *testing.T) {
	var p workspaceState
	p.Open(sampleWorkspaces(), "Personal")
	p.Close()
	if p.active {
		t.Error("expected inactive after Close")
	}
}

func TestWorkspaceState_MoveCursorClampsAtBounds(t *testing.T) {
	var p workspaceState
	p.Open(sampleWorkspaces(), "Personal")

	p.MoveCursor(-1)
	if got, _ := p.Selected(); got.Name != "Personal" {
		t.Error("expected cursor clamped at the top")
	}

	p.MoveCursor(5)
	if got, _ := p.Selected(); got.Name != "TeamX" {
		t.Error("expected cursor clamped at the bottom")
	}
}

func TestWorkspaceState_SetWorkspacesKeepsCursorInBounds(t *testing.T) {
	var p workspaceState
	p.Open(sampleWorkspaces(), "TeamX")
	p.SetWorkspaces(sampleWorkspaces()[:1]) // TeamX removed
	if got, _ := p.Selected(); got.Name != "Personal" {
		t.Errorf("got %+v, want cursor clamped back onto Personal", got)
	}
}

func TestWorkspaceState_ViewShowsEveryWorkspaceAndMarksActive(t *testing.T) {
	var p workspaceState
	p.Open(sampleWorkspaces(), "Personal")
	view := stripANSI(p.View("Personal"))
	for _, want := range []string{"Personal", "TeamX", "(active)"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q:\n%s", want, view)
		}
	}
}
