package tui

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	history "github.com/dhitalkamal/parley/internal/history/domain"
)

// TestComputeDashboardOverview_AggregatesAcrossEveryRunRegardlessOfCaps
// guards the scope decision the Overview section relies on: its stats
// always reflect every recorded run, not just the maxDashboardRunsShown/
// trendRunWindow-capped subset the panels below it show.
func TestComputeDashboardOverview_AggregatesAcrossEveryRunRegardlessOfCaps(t *testing.T) {
	var runs []history.CollectionRunEntry
	for i := 0; i < 25; i++ {
		result := execution.RunResult{RequestPath: "010_get.json", StatusCode: 200}
		if i < 5 {
			result = execution.RunResult{RequestPath: "010_get.json", Err: "boom"}
		}
		runs = append(runs, history.CollectionRunEntry{
			Time: time.Now(), Results: []execution.RunResult{result}, TotalMS: int64(10 * (i + 1)),
		})
	}

	ov := computeDashboardOverview(runs)
	if ov.total != 25 {
		t.Errorf("total = %d, want 25 (unbounded by the 20-run display cap)", ov.total)
	}
	if ov.passed != 20 || ov.failed != 5 {
		t.Errorf("passed/failed = %d/%d, want 20/5", ov.passed, ov.failed)
	}
	wantAvg := int64(0)
	for i := 0; i < 25; i++ {
		wantAvg += int64(10 * (i + 1))
	}
	wantAvg /= 25
	if ov.avgMS != wantAvg {
		t.Errorf("avgMS = %d, want %d", ov.avgMS, wantAvg)
	}
}

// TestComputeDashboardOverview_EmptyRunsIsAllZeroesNotDivideByZero guards
// the zero-runs edge case (a fresh install, or every run filtered/deleted).
func TestComputeDashboardOverview_EmptyRunsIsAllZeroesNotDivideByZero(t *testing.T) {
	ov := computeDashboardOverview(nil)
	if ov.total != 0 || ov.passed != 0 || ov.failed != 0 || ov.passRate != 0 || ov.avgMS != 0 {
		t.Errorf("got %+v, want every field zero", ov)
	}
}

// TestRunPassed_FailsOnAnyRequestError guards the run-level rollup: one
// failing request fails the whole run, matching requestPassed's own rule
// one level up.
func TestRunPassed_FailsOnAnyRequestError(t *testing.T) {
	run := history.CollectionRunEntry{Results: []execution.RunResult{
		{StatusCode: 200},
		{Err: "boom"},
	}}
	if runPassed(run) {
		t.Error("expected a run with one failing request to count as failed")
	}
}

// TestFormatDurationMS_SwitchesUnitAtOneSecond guards the mockup's own
// convention: sub-second figures read as ms, one second and up read as a
// fractional-second figure.
func TestFormatDurationMS_SwitchesUnitAtOneSecond(t *testing.T) {
	cases := map[int64]string{
		0:    "0ms",
		999:  "999ms",
		1000: "1.00s",
		2310: "2.31s",
	}
	for ms, want := range cases {
		if got := formatDurationMS(ms); got != want {
			t.Errorf("formatDurationMS(%d) = %q, want %q", ms, got, want)
		}
	}
}

// TestDashboardScreenView_OverviewShowsGlobalStatsNotJustTheVisibleWindow
// is the view-level counterpart to the aggregation test above: seed more
// runs than the log/performance panels ever display, and confirm the
// Overview row still reports the true total.
func TestDashboardScreenView_OverviewShowsGlobalStatsNotJustTheVisibleWindow(t *testing.T) {
	var runs []history.CollectionRunEntry
	for i := 0; i < 30; i++ {
		runs = append(runs, history.CollectionRunEntry{
			Time:    time.Now(),
			Results: []execution.RunResult{{RequestPath: "010_get.json", StatusCode: 200}},
		})
	}
	m := New(t.TempDir(), t.TempDir())
	m.screen = ScreenDashboard
	m.width, m.height = 140, 40
	m.dashboard = dashboardState{runs: runs}

	out := m.dashboardScreenView()
	if !strings.Contains(out, "30") {
		t.Errorf("expected the Overview row to show all 30 runs (not the 20-run display cap), got:\n%s", out)
	}
}

// TestDashboardScreenView_ManyDistinctRequestsStillRendersExactlyTheTerminalHeight
// stress-tests the exact-height contract with far more data than the
// earlier, smaller fixtures ever exercised: many runs, each with many
// distinct requests, feeding both the run log and the performance table -
// this is the shape of bug this session's Request-screen height-budget
// fixes were chasing (content taller than its box, silently overflowing).
func TestDashboardScreenView_ManyDistinctRequestsStillRendersExactlyTheTerminalHeight(t *testing.T) {
	var runs []history.CollectionRunEntry
	for run := 0; run < 15; run++ {
		var results []execution.RunResult
		for req := 0; req < 12; req++ {
			results = append(results, execution.RunResult{
				RequestPath: fmt.Sprintf("%03d_get.json", req),
				Method:      "GET",
				URL:         "https://example.com/resource/" + strconv.Itoa(req),
				StatusCode:  200,
				ElapsedMS:   int64(10 + req),
			})
		}
		runs = append(runs, history.CollectionRunEntry{
			Time: time.Now(), Path: fmt.Sprintf("run-%d", run), Results: results, TotalMS: 200,
		})
	}

	for _, size := range []struct{ w, h int }{{100, 24}, {120, 30}, {140, 40}, {160, 60}} {
		m := New(t.TempDir(), t.TempDir())
		m.screen = ScreenDashboard
		m.width, m.height = size.w, size.h
		m.dashboard = dashboardState{runs: runs}

		got := m.View()
		lines := strings.Split(got, "\n")
		if len(lines) != m.height {
			t.Errorf("w=%d h=%d: got %d lines, want %d", size.w, size.h, len(lines), m.height)
		}
	}
}

// TestDashboardScreenView_ManyRequestsInOneRunStillRendersExactlyTheTerminalHeight
// is the same stress test focused on the Selected Run pane specifically: one
// run with far more requests than any reasonable pane height can show.
func TestDashboardScreenView_ManyRequestsInOneRunStillRendersExactlyTheTerminalHeight(t *testing.T) {
	var results []execution.RunResult
	for i := 0; i < 40; i++ {
		results = append(results, execution.RunResult{
			RequestPath: fmt.Sprintf("%03d_get.json", i), Method: "GET",
			URL: "https://example.com/x", StatusCode: 200, ElapsedMS: 10,
		})
	}
	m := New(t.TempDir(), t.TempDir())
	m.screen = ScreenDashboard
	// Height 30 (not 24): the Selected Run pane needs enough rows to reach its
	// "...and N more" overflow line for 40 requests - at 24 the pane is too
	// short to render it, which isn't what this test is guarding.
	m.width, m.height = 140, 30
	m.dashboard = dashboardState{runs: []history.CollectionRunEntry{{Time: time.Now(), Results: results}}}

	got := m.View()
	lines := strings.Split(got, "\n")
	if len(lines) != m.height {
		t.Errorf("got %d lines, want %d", len(lines), m.height)
	}
	if !strings.Contains(got, "more") {
		t.Errorf("expected the request overflow to be surfaced explicitly, got:\n%s", got)
	}
}
