package tui

import (
	"strings"
	"testing"
	"time"

	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	history "github.com/dhitalkamal/parley/internal/history/domain"
	historystore "github.com/dhitalkamal/parley/internal/history/infrastructure"

	tea "github.com/charmbracelet/bubbletea"
)

func TestOpenDashboard_LoadsRunsAndSwitchesScreen(t *testing.T) {
	projectRoot := t.TempDir()
	runStore := historystore.NewRunStore(projectRoot)
	if err := runStore.AppendRun(history.CollectionRunEntry{Time: time.Now(), Path: "users"}); err != nil {
		t.Fatalf("seed append error: %v", err)
	}

	m := New(t.TempDir(), projectRoot)
	m.screen = ScreenRequest
	got, _ := m.openDashboard()
	if got.screen != ScreenDashboard {
		t.Errorf("screen = %v, want ScreenDashboard", got.screen)
	}
	if got.previousScreen != ScreenRequest {
		t.Errorf("previousScreen = %v, want ScreenRequest", got.previousScreen)
	}
	if len(got.dashboard.runs) != 1 {
		t.Fatalf("got %d runs, want 1", len(got.dashboard.runs))
	}
}

func TestHandleDashboardKey_EscReturnsToPreviousScreen(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.screen = ScreenDashboard
	m.previousScreen = ScreenRequest
	got, _ := m.handleDashboardKey(tea.KeyMsg{Type: tea.KeyEsc})
	if got.(Model).screen != ScreenRequest {
		t.Errorf("screen = %v, want ScreenRequest after esc", got.(Model).screen)
	}
}

func TestHandleDashboardKey_F5ReturnsToPreviousScreen(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.screen = ScreenDashboard
	m.previousScreen = ScreenCollections
	got, _ := m.handleDashboardKey(tea.KeyMsg{Type: tea.KeyF5})
	if got.(Model).screen != ScreenCollections {
		t.Errorf("screen = %v, want ScreenCollections after f5 toggles back", got.(Model).screen)
	}
}

func TestF5_OpensDashboardFromRequestScreen(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.focus = focusResponse

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyF5})
	got := next.(Model)
	if got.screen != ScreenDashboard {
		t.Errorf("screen = %v, want ScreenDashboard", got.screen)
	}
}

func TestDashboardScreenView_EmptyStateWhenNoRuns(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.screen = ScreenDashboard
	m.width, m.height = 100, 30
	out := m.dashboardScreenView()
	if !strings.Contains(strings.ToLower(out), "no runs") {
		t.Errorf("expected an empty-state message, got:\n%s", out)
	}
}

func TestDashboardScreenView_ShowsRunLogRow(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 100, 30
	m.screen = ScreenDashboard
	m.dashboard = dashboardState{runs: []history.CollectionRunEntry{
		{
			Time: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
			Path: "users",
			Results: []execution.RunResult{
				{RequestPath: "010_get.json", Method: "GET", URL: "https://example.com/users", StatusCode: 200, ElapsedMS: 12},
			},
			TotalMS: 12,
		},
	}}
	out := m.dashboardScreenView()
	if !strings.Contains(out, "users") {
		t.Errorf("expected the run's path in the log, got:\n%s", out)
	}
	if !strings.Contains(out, "1/0") {
		t.Errorf("expected a 1 passed/0 failed request count, got:\n%s", out)
	}
}

func TestDashboardScreenView_PerformanceTableShowsPassRateAcrossRuns(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 100, 30
	m.screen = ScreenDashboard
	// ListRuns returns newest-first; the oldest run passed, the newest failed
	// - one pass out of two runs is a 50% pass rate.
	m.dashboard = dashboardState{runs: []history.CollectionRunEntry{
		{
			Time: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
			Results: []execution.RunResult{
				{RequestPath: "010_get.json", Method: "GET", URL: "https://example.com/users", Err: "connection refused", ElapsedMS: 5},
			},
		},
		{
			Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			Results: []execution.RunResult{
				{RequestPath: "010_get.json", Method: "GET", URL: "https://example.com/users", StatusCode: 200, ElapsedMS: 10},
			},
		},
	}}
	out := m.dashboardScreenView()
	if !strings.Contains(out, "https://example.com/users") {
		t.Errorf("expected the performance table's request url, got:\n%s", out)
	}
	if !strings.Contains(out, "50%") {
		t.Errorf("expected a 50%% pass rate (1 of 2 runs passed), got:\n%s", out)
	}
}

func TestDashboardScreenView_CapsRunLogAtTwentyWithExplicitCount(t *testing.T) {
	var runs []history.CollectionRunEntry
	for i := 0; i < 25; i++ {
		runs = append(runs, history.CollectionRunEntry{Time: time.Now(), Path: "ping"})
	}
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 100, 30
	m.screen = ScreenDashboard
	m.dashboard = dashboardState{runs: runs}
	out := m.dashboardScreenView()
	if !strings.Contains(out, "and 5 more") {
		t.Errorf("expected the overflow to be surfaced explicitly, got:\n%s", out)
	}
}

func threeSeedRuns() []history.CollectionRunEntry {
	return []history.CollectionRunEntry{
		{Time: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC), Path: "run-c", Results: []execution.RunResult{
			{RequestPath: "010_get.json", Method: "GET", URL: "https://example.com/c", StatusCode: 200, ElapsedMS: 9},
		}, TotalMS: 9},
		{Time: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), Path: "run-b", Results: []execution.RunResult{
			{RequestPath: "010_get.json", Method: "GET", URL: "https://example.com/b", Err: "boom", ElapsedMS: 4},
		}, TotalMS: 4},
		{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Path: "run-a", Results: []execution.RunResult{
			{RequestPath: "010_get.json", Method: "GET", URL: "https://example.com/a", StatusCode: 200, ElapsedMS: 7},
		}, TotalMS: 7},
	}
}

// TestHandleDashboardKey_DownMovesCursorForward guards the actual ask: the
// status bar has always claimed "up/down select" even though nothing
// implemented it - this is that implementation.
func TestHandleDashboardKey_DownMovesCursorForward(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.screen = ScreenDashboard
	m.dashboard = dashboardState{runs: threeSeedRuns()}

	got, _ := m.handleDashboardKey(tea.KeyMsg{Type: tea.KeyDown})
	if got.(Model).dashboard.cursor != 1 {
		t.Errorf("cursor = %d, want 1 after one down press", got.(Model).dashboard.cursor)
	}
}

// TestHandleDashboardKey_UpDoesNotGoNegative guards the same floor every
// other cursor-based list in this app (history.go) already has.
func TestHandleDashboardKey_UpDoesNotGoNegative(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.screen = ScreenDashboard
	m.dashboard = dashboardState{runs: threeSeedRuns()}

	got, _ := m.handleDashboardKey(tea.KeyMsg{Type: tea.KeyUp})
	if got.(Model).dashboard.cursor != 0 {
		t.Errorf("cursor = %d, want 0 (clamped)", got.(Model).dashboard.cursor)
	}
}

// TestHandleDashboardKey_DownStopsAtLastShownRun guards the upper bound.
func TestHandleDashboardKey_DownStopsAtLastShownRun(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.screen = ScreenDashboard
	m.dashboard = dashboardState{runs: threeSeedRuns()}

	for i := 0; i < 10; i++ {
		next, _ := m.handleDashboardKey(tea.KeyMsg{Type: tea.KeyDown})
		m = next.(Model)
	}
	if m.dashboard.cursor != 2 {
		t.Errorf("cursor = %d, want 2 (clamped to the last of 3 runs)", m.dashboard.cursor)
	}
}

// TestDashboardScreenView_HighlightsTheSelectedRun guards the visible half
// of selection - a cursor with no on-screen marker is invisible to a user.
func TestDashboardScreenView_HighlightsTheSelectedRun(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	// Wide enough that the run-log line's own fixed-width columns (date,
	// padded label, "X/Y passed", latency) all fit in the left column's own
	// budget without clipLine cutting any of them off - see
	// dashboardLogAndPerformanceContent's doc comment on why they're clipped
	// rather than wrapped at all.
	m.width, m.height = 160, 34
	m.screen = ScreenDashboard
	m.dashboard = dashboardState{runs: threeSeedRuns(), cursor: 1}

	out := m.dashboardScreenView()
	lines := strings.Split(out, "\n")
	// A run's passed/failed fraction (see dashboardRunLine) only ever
	// appears on its own run-log row - the run's number and path also show
	// up unmarked elsewhere on this same combined terminal row once
	// Selected Run sits beside the log (see dashboardSelectedRunContent's
	// own meta line, "#N  label"), so the log row itself has to be
	// identified by its fraction, not just "#N"/path alone.
	found := false
	for _, line := range lines {
		if strings.Contains(line, "run-b") && strings.Contains(line, "0/1") {
			if !strings.Contains(line, ">") {
				t.Errorf("selected run's line = %q, want a selection marker", line)
			}
			found = true
		}
		if strings.Contains(line, "run-c") && strings.Contains(line, "1/0") && strings.Contains(line, ">") {
			t.Errorf("unselected run's line = %q, want no selection marker", line)
		}
	}
	if !found {
		t.Fatal("expected to find the run-b log line at all")
	}
}

// TestDashboardScreenView_RunDetailsPaneLiveMirrorsTheSelectedRun guards
// the split-pane redesign's whole premise: moving the cursor alone updates
// the Run Details pane immediately, on a wide enough terminal - no
// Enter-to-open step, and every request in the run shows, not just the log's
// one summary line.
func TestDashboardScreenView_RunDetailsPaneLiveMirrorsTheSelectedRun(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 34
	m.screen = ScreenDashboard
	m.dashboard = dashboardState{runs: threeSeedRuns(), cursor: 1}

	out := m.dashboardScreenView()
	if !strings.Contains(out, "https://example.com/b") {
		t.Errorf("expected the selected run's request URL, got:\n%s", out)
	}
	if !strings.Contains(out, "ERROR: boom") {
		t.Errorf("expected the selected run's error shown, got:\n%s", out)
	}
	if !strings.Contains(out, "Failed") {
		t.Errorf("expected a Failed stat row, got:\n%s", out)
	}

	got, _ := m.handleDashboardKey(tea.KeyMsg{Type: tea.KeyDown})
	after := got.(Model)
	out = after.dashboardScreenView()
	if !strings.Contains(out, "https://example.com/c") {
		t.Errorf("expected Run Details to follow the cursor to run-c after down, got:\n%s", out)
	}
	if strings.Contains(out, "ERROR: boom") {
		t.Errorf("expected run-b's details gone once the cursor moved off it, got:\n%s", out)
	}
}

// TestDashboardScreenView_OmitsRunDetailsPaneOnNarrowTerminal guards the
// graceful-degradation floor - a narrow terminal falls back to just the run
// log/performance column instead of squeezing a second pane.
func TestDashboardScreenView_OmitsRunDetailsPaneOnNarrowTerminal(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 80, 24
	m.screen = ScreenDashboard
	m.dashboard = dashboardState{runs: threeSeedRuns(), cursor: 1}

	out := m.dashboardScreenView()
	if strings.Contains(out, "Run Details") {
		t.Errorf("dashboardScreenView() at width 80 = %q, want no Run Details pane", out)
	}
}

func TestDashboardScreenView_RendersExactlyTheTerminalHeight(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.screen = ScreenDashboard
	m.width, m.height = 100, 30

	got := m.View()
	lines := strings.Split(got, "\n")
	if len(lines) != m.height {
		t.Errorf("got %d lines, want %d", len(lines), m.height)
	}
}

// TestDashboard_OverviewCachedOnSetRuns guards that the overview aggregate is
// computed when the run list changes, not re-scanned over the whole history on
// every 1s render frame. setRuns must keep dashboardState.overview in sync.
func TestDashboard_OverviewCachedOnSetRuns(t *testing.T) {
	runs := []history.CollectionRunEntry{
		{Path: "a", TotalMS: 100, Results: []execution.RunResult{{}}},
		{Path: "b", TotalMS: 300, Results: nil},
	}
	var d dashboardState
	d.setRuns(runs)
	if got, want := d.overview, computeDashboardOverview(runs); got != want {
		t.Fatalf("after setRuns overview = %+v, want %+v", got, want)
	}
	if d.overview.total != 2 {
		t.Errorf("overview.total = %d, want 2", d.overview.total)
	}
	// prepending a run must refresh the cached overview.
	grown := append([]history.CollectionRunEntry{{Path: "c", TotalMS: 50, Results: nil}}, d.runs...)
	d.setRuns(grown)
	if got, want := d.overview, computeDashboardOverview(grown); got != want {
		t.Fatalf("after growth overview = %+v, want %+v", got, want)
	}
	if d.overview.total != 3 {
		t.Errorf("overview.total = %d, want 3", d.overview.total)
	}
}
