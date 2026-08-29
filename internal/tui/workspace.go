package tui

import (
	workspace "github.com/dhitalkamal/parley/internal/workspace/domain"
	"strings"
)

// workspaceState is the "Workspaces" switcher modal - a single list (switch/
// create/delete all in one place) rather than a separate quick-dropdown plus
// a full manage panel like the environment system has, since a user
// realistically has only a handful of workspaces and the extra split isn't
// worth the added surface here.
type workspaceState struct {
	active     bool
	workspaces []workspace.Workspace
	cursor     int
}

// Open shows the modal, starting the cursor on whichever workspace matches
// activeName, or the first entry if that name isn't found (e.g. it was
// renamed/deleted out from under an already-open session).
func (p *workspaceState) Open(workspaces []workspace.Workspace, activeName string) {
	p.active = true
	p.workspaces = workspaces
	p.cursor = 0
	for i, w := range workspaces {
		if w.Name == activeName {
			p.cursor = i
		}
	}
}

func (p *workspaceState) Close() {
	p.active = false
}

// MoveCursor shifts the selection by delta, clamped to the list's bounds -
// no wraparound, matching envPanelState's own scope-list navigation.
func (p *workspaceState) MoveCursor(delta int) {
	p.cursor += delta
	if p.cursor < 0 {
		p.cursor = 0
	}
	if max := len(p.workspaces) - 1; p.cursor > max {
		p.cursor = max
	}
}

// SetWorkspaces replaces the list (after a create/delete), keeping the
// cursor in bounds if the list got shorter.
func (p *workspaceState) SetWorkspaces(workspaces []workspace.Workspace) {
	p.workspaces = workspaces
	if p.cursor < 0 {
		p.cursor = 0
	}
	if max := len(workspaces) - 1; p.cursor > max {
		p.cursor = max
	}
}

// Selected is the workspace currently highlighted, or ok=false if the list
// is empty.
func (p workspaceState) Selected() (workspace.Workspace, bool) {
	if p.cursor < 0 || p.cursor >= len(p.workspaces) {
		return workspace.Workspace{}, false
	}
	return p.workspaces[p.cursor], true
}

// View renders every workspace, the active one marked, plus the available
// actions - same visual language as envPanelState/envDropdownState.
func (p workspaceState) View(activeName string) string {
	var b strings.Builder
	b.WriteString(activeTabStyle.Render("Workspaces") + "\n\n")
	if len(p.workspaces) == 0 {
		b.WriteString(labelStyle.Render("No workspaces yet.") + "\n")
	}
	for i, w := range p.workspaces {
		suffix := ""
		if w.Name == activeName {
			suffix = " (active)"
		}
		line := "  " + w.Name + suffix
		if i == p.cursor {
			line = selStyle.Bold(true).Render("> " + w.Name + suffix)
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\n" + labelStyle.Render("enter switch - n new - d delete - esc close"))
	return modalStyle.Width(modalWidth).Render(b.String())
}
