package tui

import (
	"testing"
	"time"

	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	history "github.com/dhitalkamal/parley/internal/history/domain"
	historystore "github.com/dhitalkamal/parley/internal/history/infrastructure"

	tea "github.com/charmbracelet/bubbletea"
)

// TestHandleDashboardKey_ROpensRerunAgainstTheSelectedRunsOwnRequests guards
// the rerun action's whole point: it replays exactly the requests the
// original run executed (by RequestPath), not whatever's currently selected
// in the sidebar - the run's own Path (folder label) is threaded through so
// the freshly appended history entry keeps the same label.
func TestHandleDashboardKey_ROpensRerunAgainstTheSelectedRunsOwnRequests(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.screen = ScreenDashboard
	m.dashboard = dashboardState{runs: threeSeedRuns()}

	next, cmd := m.handleDashboardKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	got := next.(Model)
	if !got.runner.active || !got.runner.running {
		t.Fatalf("expected the runner to be active and running, got %+v", got.runner)
	}
	if cmd == nil {
		t.Fatal("expected a command to actually run the request")
	}
}

// TestHandleDashboardKey_RWithNoSelectionIsANoOp guards the empty-dashboard
// edge case - nothing to rerun shouldn't panic or arm the runner.
func TestHandleDashboardKey_RWithNoSelectionIsANoOp(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.screen = ScreenDashboard
	m.dashboard = dashboardState{}

	next, _ := m.handleDashboardKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	if next.(Model).runner.active {
		t.Error("expected the runner to stay inactive with no run selected")
	}
}

// TestHandleDashboardKey_DOpensDeleteConfirmForTheSelectedRun guards that
// deleting a run is confirmed first, like every other destructive action in
// this app (sidebar delete, environment delete, workspace delete).
func TestHandleDashboardKey_DOpensDeleteConfirmForTheSelectedRun(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.screen = ScreenDashboard
	m.dashboard = dashboardState{runs: threeSeedRuns()}

	next, _ := m.handleDashboardKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	got := next.(Model)
	if !got.confirm.active || got.confirm.purpose != confirmDeleteRun {
		t.Fatalf("expected an armed confirmDeleteRun modal, got %+v", got.confirm)
	}
}

// TestDeleteSelectedDashboardRun_RemovesItFromTheStoreAndTheLog guards the
// actual deletion: it must reach the real store (so it survives a restart),
// not just the in-memory slice, and the in-memory log must reflect that
// immediately.
func TestDeleteSelectedDashboardRun_RemovesItFromTheStoreAndTheLog(t *testing.T) {
	projectRoot := t.TempDir()
	runStore := historystore.NewRunStore(projectRoot)
	kept := history.CollectionRunEntry{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Path: "keep-me"}
	target := history.CollectionRunEntry{Time: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), Path: "delete-me"}
	for _, e := range []history.CollectionRunEntry{kept, target} {
		if err := runStore.AppendRun(e); err != nil {
			t.Fatalf("seed append error: %v", err)
		}
	}

	m := New(t.TempDir(), projectRoot)
	m.screen = ScreenDashboard
	got, _ := m.openDashboard()
	got.dashboard.cursor = 0 // ListRuns is newest-first: "delete-me" is first

	after, _ := got.deleteSelectedDashboardRun()
	if len(after.dashboard.runs) != 1 || after.dashboard.runs[0].Path != "keep-me" {
		t.Errorf("in-memory log = %+v, want only keep-me left", after.dashboard.runs)
	}

	onDisk, err := runStore.ListRuns()
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(onDisk) != 1 || onDisk[0].Path != "keep-me" {
		t.Errorf("on-disk log = %+v, want only keep-me left", onDisk)
	}
}

// TestHandleDashboardKey_SlashOpensSearchPrompt guards the search action's
// entry point.
func TestHandleDashboardKey_SlashOpensSearchPrompt(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.screen = ScreenDashboard
	m.dashboard = dashboardState{runs: threeSeedRuns()}

	next, _ := m.handleDashboardKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	got := next.(Model)
	if !got.prompt.active || got.prompt.purpose != promptDashboardSearch {
		t.Fatalf("expected an armed promptDashboardSearch, got %+v", got.prompt)
	}
}

// TestSubmitPrompt_DashboardSearchFiltersTheRunLogByLabel guards the actual
// filtering: only runs whose label contains the query (case-insensitively)
// remain, and the cursor resets so it can't point past the new, shorter
// list.
func TestSubmitPrompt_DashboardSearchFiltersTheRunLogByLabel(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.screen = ScreenDashboard
	m.dashboard = dashboardState{runs: threeSeedRuns(), cursor: 2}
	m.prompt.Open(promptDashboardSearch, "", "Filter runs", "")
	m.prompt.input.SetValue("RUN-B")

	got, _ := m.submitPrompt()
	if got.dashboard.filter != "RUN-B" {
		t.Errorf("filter = %q, want RUN-B", got.dashboard.filter)
	}
	if got.dashboard.cursor != 0 {
		t.Errorf("cursor = %d, want reset to 0", got.dashboard.cursor)
	}
	shown := got.dashboard.shownRuns()
	if len(shown) != 1 || shown[0].entry.Path != "run-b" {
		t.Errorf("shownRuns() = %+v, want only run-b", shown)
	}
}

// TestHandleRunResult_PrependsToTheDashboardLog guards the rerun ->
// dashboard sync: a run that finishes while the Dashboard's in-memory log
// is already loaded should show up immediately, without requiring the
// screen to be closed and reopened.
func TestHandleRunResult_PrependsToTheDashboardLog(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.screen = ScreenDashboard
	m.dashboard = dashboardState{runs: threeSeedRuns()}
	before := len(m.dashboard.runs)

	got := m.handleRunResult(runResultMsg{
		path:    "rerun-target",
		results: []execution.RunResult{{RequestPath: "010_get.json", Method: "GET", URL: "https://example.com/x", StatusCode: 200}},
	})
	if len(got.dashboard.runs) != before+1 {
		t.Fatalf("got %d runs, want %d", len(got.dashboard.runs), before+1)
	}
	if got.dashboard.runs[0].Path != "rerun-target" {
		t.Errorf("newest run = %+v, want it prepended first", got.dashboard.runs[0])
	}
}
