package tui

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
)

// envPanelState is the toggle-open, Postman-like right-side panel: a list of
// scopes (globals always first, then every named environment) and a variable
// editor for whichever scope is selected. Picking a named environment (not
// globals) also makes it the active one used for request substitution - the
// panel replaces the old blind ctrl+e cycle with a visible, navigable list
// doing the same job.
//
// This type only holds UI state (which scope is selected, which pane has
// focus). Loading/saving variables against environment.EnvironmentStore is done
// by Model methods in envpanel_actions.go, the same split prompt/confirm use
// between their own dumb state structs and Model's store-backed handlers.
type envPanelState struct {
	active    bool
	envNames  []string // sorted named environments, NOT including globals
	cursor    int      // 0 = Globals, 1..len(envNames) = envNames[cursor-1]
	focusVars bool

	// Variable editor for whichever scope is selected - see envpanel_vars.go.
	rows    []envVarRow
	tbl     table.Model
	editing bool
	// adding distinguishes an in-progress "new variable" edit (append on
	// commit) from editing an existing row (overwrite the selected row on
	// commit). Only the side rail's inline add uses it - the ctrl+e panel
	// adds through the kvAdd modal instead, so it never sets this.
	adding    bool
	keyInput  textinput.Model
	valInput  textinput.Model
	editField int // 0 = key field focused, 1 = value field focused
}

// Open shows the panel with the given (sorted) environment names, always
// starting on the Globals scope.
func (p *envPanelState) Open(envNames []string) {
	p.active = true
	p.envNames = envNames
	p.cursor = 0
	p.focusVars = false
}

func (p *envPanelState) Close() {
	p.active = false
	p.focusVars = false
}

// scopeCount is Globals plus every named environment.
func (p envPanelState) scopeCount() int {
	return len(p.envNames) + 1
}

// MoveCursor shifts the selected scope by delta, clamped to the scope list's
// bounds rather than wrapping - Globals at the top and the last environment
// at the bottom are natural stopping points, not a cycle.
func (p *envPanelState) MoveCursor(delta int) {
	p.cursor += delta
	if p.cursor < 0 {
		p.cursor = 0
	}
	if max := p.scopeCount() - 1; p.cursor > max {
		p.cursor = max
	}
}

// SetEnvNames replaces the environment list (after a create/delete), keeping
// the cursor in bounds if the list got shorter.
func (p *envPanelState) SetEnvNames(envNames []string) {
	p.envNames = envNames
	if max := p.scopeCount() - 1; p.cursor > max {
		p.cursor = max
	}
}

// IsGlobalsSelected reports whether the top ("Globals") scope is selected.
func (p envPanelState) IsGlobalsSelected() bool {
	return p.cursor == 0
}

// ScopeName is the currently selected environment's name, or "" for Globals.
func (p envPanelState) ScopeName() string {
	if p.cursor == 0 {
		return ""
	}
	return p.envNames[p.cursor-1]
}

// ToggleFocus moves keyboard focus between the scope list and the variable
// editor - the two panes of this panel.
func (p *envPanelState) ToggleFocus() {
	p.focusVars = !p.focusVars
}
