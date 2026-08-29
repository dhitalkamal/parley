package tui

import (
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	environment "github.com/dhitalkamal/parley/internal/environment/domain"
	workspace "github.com/dhitalkamal/parley/internal/workspace/domain"
	workspacestore "github.com/dhitalkamal/parley/internal/workspace/infrastructure"
)

func environmentWithVar(name, key, value string) environment.Environment {
	return environment.Environment{
		Name:      name,
		Variables: []environment.Variable{{Key: key, Value: value, Enabled: true}},
	}
}

// runeKey builds a single-rune key message - space and letters both arrive
// this way, matching how the app compares against k.String().
func runeKey(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func railStopsContain(stops []int, want int) bool {
	for _, s := range stops {
		if s == want {
			return true
		}
	}
	return false
}

// TestRail_DefaultsToEnvView guards the headline decision: the rail shows its
// editable environment view by default, not the shortcuts list.
func TestRail_DefaultsToEnvView(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	if !m.railEnvMode() {
		t.Fatalf("rail should default to the env view, got railMode %v", m.railMode)
	}
}

// TestRail_F1TogglesVisibility guards the direct hide/unhide keybinding - the
// discoverability gap that made the rail feel static.
func TestRail_F1TogglesVisibility(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 40
	m.screen = ScreenRequest
	if !m.shortcutsRailVisible() {
		t.Fatal("precondition: rail should be visible at width 160")
	}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyF1})
	got := next.(Model)
	if !got.shortcutsRailHidden {
		t.Error("f1 should hide the rail")
	}
	next, _ = got.Update(tea.KeyMsg{Type: tea.KeyF1})
	got = next.(Model)
	if got.shortcutsRailHidden {
		t.Error("f1 again should show the rail")
	}
}

// TestRail_HidingWhileFocusedReleasesFocus guards against stranding focus on
// an off-screen zone.
func TestRail_HidingWhileFocusedReleasesFocus(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 40
	m.screen = ScreenRequest
	m.focus = focusRail
	m.updateFocus()

	m.toggleSideRail()
	if m.focus == focusRail {
		t.Error("hiding the rail while it has focus should move focus off focusRail")
	}
}

// TestRail_VSwitchesViewWhileFocused guards the in-rail view toggle: v flips
// env <-> shortcuts and back, both directions, while the rail has focus.
func TestRail_VSwitchesViewWhileFocused(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 40
	m.screen = ScreenRequest
	m.focus = focusRail
	m.updateFocus()

	next, _ := m.Update(runeKey('v'))
	got := next.(Model)
	if got.railEnvMode() {
		t.Error("v should switch the rail out of the env view")
	}
	next, _ = got.Update(runeKey('v'))
	got = next.(Model)
	if !got.railEnvMode() {
		t.Error("v again should switch back to the env view")
	}
}

// TestRail_FocusStopOnlyWhenVisible guards that focusRail joins the Tab cycle
// exactly when the rail is on screen - the same visibility gate focusSidebar
// uses for the drawer.
func TestRail_FocusStopOnlyWhenVisible(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.screen = ScreenRequest
	m.closeDrawer()

	m.width = minWidthForSidebar - 1
	if railStopsContain(m.tabStops(), focusRail) {
		t.Error("focusRail should not be a tab stop below the width floor")
	}
	m.width = minWidthForSidebar
	if !railStopsContain(m.tabStops(), focusRail) {
		t.Error("focusRail should be a tab stop at the width floor")
	}
	m.shortcutsRailHidden = true
	if railStopsContain(m.tabStops(), focusRail) {
		t.Error("focusRail should not be a tab stop while the rail is hidden")
	}
}

// TestRail_JumpFocusesRail guards the dedicated jump key: f10 moves focus
// straight to the rail from anywhere, the same way f2/f3 jump to
// Request/Response.
func TestRail_JumpFocusesRail(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 40
	m.screen = ScreenRequest
	m.focus = focusURL

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyF10})
	got := next.(Model)
	if got.focus != focusRail {
		t.Errorf("f10 should focus the rail, got focus %v", got.focus)
	}
}

// TestRail_JumpUnhidesThenFocuses guards that jumping to the rail while it's
// manually hidden brings it back and focuses it - "take me to the env config"
// should always land there, not silently no-op.
func TestRail_JumpUnhidesThenFocuses(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 40
	m.screen = ScreenRequest
	m.shortcutsRailHidden = true
	m.focus = focusURL

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyF10})
	got := next.(Model)
	if got.shortcutsRailHidden {
		t.Error("f10 should unhide the rail")
	}
	if got.focus != focusRail {
		t.Errorf("f10 should focus the rail, got focus %v", got.focus)
	}
}

// TestRail_JumpNoopWhenTooNarrow guards the width floor: with no room for the
// rail, the jump does nothing rather than focusing an off-screen zone.
func TestRail_JumpNoopWhenTooNarrow(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = minWidthForSidebar-1, 40
	m.screen = ScreenRequest
	m.focus = focusURL

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyF10})
	got := next.(Model)
	if got.focus != focusURL {
		t.Errorf("f10 below the width floor should be a no-op, got focus %v", got.focus)
	}
}

// TestRail_AddVariableUsesInlineBottomForm walks the add flow end to end: 'a'
// opens an inline form (not the centered modal), typing fills the key, and
// enter appends the row into the editor and persists it to the store (globals,
// since no environment is active).
func TestRail_AddVariableUsesInlineBottomForm(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 40
	m.focus = focusRail
	m.updateFocus()
	m.railSection = railGlobalVars // add to the globals section (no env active)

	m, _, handled := m.handleRailKey(runeKey('a'))
	if !handled {
		t.Fatal("rail should handle 'a'")
	}
	if m.kvAdd.active {
		t.Error("'a' should not open the centered kvAdd modal anymore")
	}
	if !m.railGlobals.IsEditing() || !m.railGlobals.IsAdding() {
		t.Fatalf("'a' should open the inline add form, got editing=%v adding=%v", m.railGlobals.IsEditing(), m.railGlobals.IsAdding())
	}

	for _, r := range "baseUrl" {
		m, _, _ = m.handleRailKey(runeKey(r))
	}
	m, _, _ = m.handleRailKey(tea.KeyMsg{Type: tea.KeyEnter})

	if got := len(m.railGlobals.Rows()); got != 1 {
		t.Fatalf("globals editor should have 1 row after add, got %d", got)
	}
	g, err := m.envStore.LoadGlobals()
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Variables) != 1 || g.Variables[0].Key != "baseUrl" {
		t.Errorf("globals not persisted, got %+v", g.Variables)
	}
}

// TestRail_AddCancelledOnBlankKey guards that leaving the key empty and
// committing adds nothing - a cancelled add, not a nameless variable.
func TestRail_AddCancelledOnBlankKey(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 40
	m.focus = focusRail
	m.updateFocus()
	m.railSection = railGlobalVars

	m, _, _ = m.handleRailKey(runeKey('a'))
	m, _, _ = m.handleRailKey(tea.KeyMsg{Type: tea.KeyEnter})

	if got := len(m.railGlobals.Rows()); got != 0 {
		t.Errorf("committing a blank add should add nothing, got %d rows", got)
	}
}

// TestRail_ToggleEnabledAndDeletePersist guards the two in-place edits that
// save immediately, the same as the ctrl+e panel: space flips enabled, d
// deletes, both written straight through to the store.
func TestRail_ToggleEnabledAndDeletePersist(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 40
	m.focus = focusRail
	m.updateFocus()
	m.railGlobals.AddRowWithValues("a", "1")
	m.railGlobals.AddRowWithValues("b", "2") // cursor lands on this one
	m.saveRailGlobals()
	m.railSection = railGlobalVars // cursor on the globals list, not the selector

	m, _, _ = m.handleRailKey(runeKey(' '))
	g, _ := m.envStore.LoadGlobals()
	if len(g.Variables) != 2 || g.Variables[1].Enabled {
		t.Fatalf("space should disable the selected row and persist it, got %+v", g.Variables)
	}

	m, _, _ = m.handleRailKey(runeKey('d'))
	g, _ = m.envStore.LoadGlobals()
	if len(g.Variables) != 1 || g.Variables[0].Key != "a" {
		t.Fatalf("d should delete the selected row and persist it, got %+v", g.Variables)
	}
}

// TestRail_EditRowCommitPersists guards the inline edit form: enter opens it,
// typing changes the key, and a second enter commits and saves.
func TestRail_EditRowCommitPersists(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 40
	m.focus = focusRail
	m.updateFocus()
	m.railGlobals.AddRowWithValues("k", "v")
	m.saveRailGlobals()
	m.railSection = railGlobalVars // cursor on the globals variable

	m, _, _ = m.handleRailKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.railGlobals.IsEditing() {
		t.Fatal("enter should open the edit form")
	}
	m, _, _ = m.handleRailKey(runeKey('X')) // append to the key field (focused first)
	m, _, _ = m.handleRailKey(tea.KeyMsg{Type: tea.KeyEnter})
	if m.railGlobals.IsEditing() {
		t.Fatal("enter should commit and close the edit form")
	}

	g, _ := m.envStore.LoadGlobals()
	if len(g.Variables) != 1 || g.Variables[0].Key != "kX" {
		t.Errorf("edit should persist the changed key, got %+v", g.Variables)
	}
}

// TestRail_LeftRightCyclesNamedEnvsOnly guards inline switching: right/left step
// the active environment through the NAMED environments only - Globals is not in
// the ring (it's shared, shown in its own section). No dropdown.
func TestRail_LeftRightCyclesNamedEnvsOnly(t *testing.T) {
	project := t.TempDir()
	m := New(t.TempDir(), project)
	if err := m.envStore.SaveEnvironment(environmentWithVar("dev", "d", "1")); err != nil {
		t.Fatal(err)
	}
	if err := m.envStore.SaveEnvironment(environmentWithVar("prod", "p", "2")); err != nil {
		t.Fatal(err)
	}
	m.envNames, _ = m.envStore.ListEnvironments() // dev, prod
	m.width, m.height = 160, 40
	m.focus = focusRail
	m.updateFocus() // cursor defaults to the selector

	// none -> dev -> prod -> dev (wraps, never Globals); left -> prod.
	m, _, _ = m.handleRailKey(tea.KeyMsg{Type: tea.KeyRight})
	if m.activeEnvName != "dev" {
		t.Fatalf("right should activate dev, got %q", m.activeEnvName)
	}
	if m.envDropdown.active {
		t.Error("right should switch inline, never open a dropdown")
	}
	m, _, _ = m.handleRailKey(tea.KeyMsg{Type: tea.KeyRight})
	if m.activeEnvName != "prod" {
		t.Fatalf("right should activate prod, got %q", m.activeEnvName)
	}
	m, _, _ = m.handleRailKey(tea.KeyMsg{Type: tea.KeyRight})
	if m.activeEnvName != "dev" {
		t.Fatalf("right should wrap to dev (not Globals), got %q", m.activeEnvName)
	}
	m, _, _ = m.handleRailKey(tea.KeyMsg{Type: tea.KeyLeft})
	if m.activeEnvName != "prod" {
		t.Fatalf("left should go back to prod, got %q", m.activeEnvName)
	}
}

// TestRail_EnterOnSelectorDropsToVariables guards that enter on the selector
// moves into the sections (not a dropdown, not an edit).
func TestRail_EnterOnSelectorDropsToVariables(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 40
	m.focus = focusRail
	m.updateFocus()
	m.railGlobals.AddRowWithValues("k", "v") // a globals row to land on

	m, _, _ = m.handleRailKey(tea.KeyMsg{Type: tea.KeyEnter})
	if m.envDropdown.active {
		t.Error("enter on the selector should not open a dropdown")
	}
	if m.railSection == railSelector {
		t.Error("enter on the selector should move the cursor into a section")
	}
}

// TestRail_ShiftDownReachesEmptyThenGlobals guards section jumping with
// shift+down: it reaches an EMPTY Variables section (no active env) and then
// Globals, where a global can be edited. Plain down stays within a section.
func TestRail_ShiftDownReachesEmptyThenGlobals(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 160, 40
	m.focus = focusRail
	m.updateFocus() // no active env -> env (Variables) section is empty
	m.railGlobals.AddRowWithValues("k", "v")

	// shift+down: Active -> Expects (VPN) -> (empty) Variables -> Globals.
	m, _, _ = m.handleRailKey(tea.KeyMsg{Type: tea.KeyShiftDown})
	if m.railSection != railVPNSelector {
		t.Fatalf("first shift+down should land on the Expects/VPN selector, got %v", m.railSection)
	}
	m, _, _ = m.handleRailKey(tea.KeyMsg{Type: tea.KeyShiftDown})
	if m.railSection != railEnvVars {
		t.Fatalf("second shift+down should land on the (empty) Variables section, got %v", m.railSection)
	}
	m, _, _ = m.handleRailKey(tea.KeyMsg{Type: tea.KeyShiftDown})
	if m.railSection != railGlobalVars {
		t.Fatalf("third shift+down should land on the Globals section, got %v", m.railSection)
	}
	m, _, _ = m.handleRailKey(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.railGlobals.IsEditing() {
		t.Error("enter on a global should start editing it")
	}
	// shift+up walks all the way back to the Active selector.
	m.railGlobals.CancelEdit()
	for i := 0; i < 3; i++ {
		m, _, _ = m.handleRailKey(tea.KeyMsg{Type: tea.KeyShiftUp})
	}
	if m.railSection != railSelector {
		t.Errorf("shift+up back should return to the Active selector, got %v", m.railSection)
	}
}

// TestRail_SwitchingActiveEnvReloadsRailRows guards that the rail follows the
// active environment: making one active loads its variables into the rail.
func TestRail_SwitchingActiveEnvReloadsRailRows(t *testing.T) {
	project := t.TempDir()
	m := New(t.TempDir(), project)
	if err := m.envStore.SaveEnvironment(environmentWithVar("dev", "token", "abc")); err != nil {
		t.Fatal(err)
	}
	m.envNames, _ = m.envStore.ListEnvironments()

	m.setActiveEnvByName("dev")
	rows := m.railEnv.Rows()
	if len(rows) != 1 || rows[0].Key != "token" || rows[0].Value != "abc" {
		t.Errorf("rail should reload onto the newly active env, got %+v", rows)
	}
}

// TestRail_ModeTogglePersistsToWorkspace guards that the chosen rail view
// survives a restart, the same way the hide/show flag already does.
func TestRail_ModeTogglePersistsToWorkspace(t *testing.T) {
	wsStore := workspacestore.New(filepath.Join(t.TempDir(), "workspaces.json"))
	ws := workspace.Workspace{Name: "Team", CollectionsRoot: t.TempDir(), ProjectRoot: t.TempDir()}
	if err := wsStore.Save(ws); err != nil {
		t.Fatal(err)
	}
	m := New(t.TempDir(), t.TempDir())
	m.screen = ScreenRequest
	m.SetWorkspaces(wsStore, []workspace.Workspace{ws}, "Team", t.TempDir())

	cmd := findPaletteCommand(t, "Toggle rail view (env/shortcuts)")
	got, _ := cmd.run(m)
	if got.railEnvMode() {
		t.Error("toggling rail view should leave env mode")
	}
	saved, _ := wsStore.List()
	if saved[0].Layout.RailMode != int(railModeShortcuts) {
		t.Errorf("rail view should persist to the workspace registry, got RailMode=%d", saved[0].Layout.RailMode)
	}
}
