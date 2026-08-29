package tui

import (
	"fmt"
	"strings"

	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	history "github.com/dhitalkamal/parley/internal/history/domain"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// maxDashboardRunsShown caps the run log the same way maxRunnerResultsShown
// caps a single run's results - explicitly surfaced ("...and N more"),
// never silently.
const maxDashboardRunsShown = 20

// trendRunWindow is how many of the most recent runs (across all of them,
// not per request) the Request Performance table considers - enough to
// show a real trend without the table growing unbounded as the log fills up.
const trendRunWindow = 10

// minWidthForDashboardSplit is the terminal width below which Selected Run
// collapses out of its own side-by-side pane and the dashboard falls back
// to one stacked column, full width - the same graceful-degradation floor
// pattern as minWidthForShortcutsRail/minWidthForCollectionsAboutPanel.
const minWidthForDashboardSplit = 100

const dashboardStatusBar = "up/down select  r rerun  d delete  / search  esc back  f5 back  ? help"

// dashboardRunRow pairs a recorded run with its 1-based sequence number in
// the full (unfiltered) history - oldest run is #1, newest is #len(runs) -
// so the number a user sees for a given run stays stable across filtering
// instead of shifting with whatever subset is currently shown.
type dashboardRunRow struct {
	entry  history.CollectionRunEntry
	number int
}

type dashboardState struct {
	// runs is newest-first, the same order RunHistoryStore.ListRuns returns.
	runs   []history.CollectionRunEntry
	cursor int
	// filter narrows the run log to entries whose label contains it
	// (case-insensitive) - see filteredRuns. Empty means "show everything".
	filter string
}

// filteredRuns is every run matching d.filter (all of them if it's empty),
// each tagged with its stable position in the full history.
func (d dashboardState) filteredRuns() []dashboardRunRow {
	rows := make([]dashboardRunRow, len(d.runs))
	for i, r := range d.runs {
		rows[i] = dashboardRunRow{entry: r, number: len(d.runs) - i}
	}
	if d.filter == "" {
		return rows
	}
	q := strings.ToLower(d.filter)
	kept := rows[:0]
	for _, row := range rows {
		if strings.Contains(strings.ToLower(dashboardRunLabel(row.entry)), q) {
			kept = append(kept, row)
		}
	}
	return kept
}

// shownRuns applies the same maxDashboardRunsShown cap the run log render
// itself uses - shared so the cursor's bounds and the Selected Run pane's
// lookup can never disagree with what's actually on screen.
func (d dashboardState) shownRuns() []dashboardRunRow {
	rows := d.filteredRuns()
	if len(rows) > maxDashboardRunsShown {
		return rows[:maxDashboardRunsShown]
	}
	return rows
}

// selectedRun is the run under the cursor, among shownRuns - ok is false
// with no runs at all (an empty or fully-filtered-out dashboard) or a stale
// cursor past the end.
func (d dashboardState) selectedRun() (dashboardRunRow, bool) {
	shown := d.shownRuns()
	if d.cursor < 0 || d.cursor >= len(shown) {
		return dashboardRunRow{}, false
	}
	return shown[d.cursor], true
}

// openDashboard loads every recorded collection run (interactive or via
// `parley run`) so the dashboard can show pass/fail and latency trends
// across them - see runner.go's handleRunResult for where they're recorded -
// and switches to the Dashboard screen, remembering whichever screen was
// active so its own back-key returns to it.
func (m Model) openDashboard() (Model, tea.Cmd) {
	runs, err := m.runHistoryStore.ListRuns()
	if err != nil {
		m.status = "Dashboard failed: " + err.Error()
		return m, nil
	}
	if m.screen != ScreenDashboard {
		m.previousScreen = m.screen
	}
	m.screen = ScreenDashboard
	m.dashboard = dashboardState{runs: runs}
	return m, nil
}

// handleDashboardKey moves the cursor with up/down - Selected Run (see
// dashboardSelectedRunPane) always mirrors whichever run that cursor is
// currently on, live, so there's no separate "open"/"close" step the way an
// Enter-driven drill-down would need. r/d/"/" act on whichever run the
// cursor is currently on.
func (m Model) handleDashboardKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(k, keys.Quit) {
		return m, tea.Quit
	}
	switch k.String() {
	case "up", "k":
		if m.dashboard.cursor > 0 {
			m.dashboard.cursor--
		}
		return m, nil
	case "down", "j":
		if m.dashboard.cursor < len(m.dashboard.shownRuns())-1 {
			m.dashboard.cursor++
		}
		return m, nil
	case "r":
		if row, ok := m.dashboard.selectedRun(); ok {
			return m.startRerunFromHistory(row.entry)
		}
		return m, nil
	case "d":
		if row, ok := m.dashboard.selectedRun(); ok {
			m.confirm.Open(confirmDeleteRun, "Delete run '"+dashboardRunLabel(row.entry)+"'?", "")
		}
		return m, nil
	case "/":
		m.prompt.Open(promptDashboardSearch, "", "Filter runs", m.dashboard.filter)
		return m, nil
	}
	if k.String() == "esc" || key.Matches(k, keys.Dashboard) {
		m.screen = m.previousScreen
	}
	return m, nil
}

// deleteSelectedDashboardRun answers "y" on the confirmDeleteRun modal (see
// prompt.go) - removes the run under the cursor from the store and reloads
// the in-memory log from it, rather than just splicing the local slice, so
// the on-disk log and what's shown can never drift apart.
func (m Model) deleteSelectedDashboardRun() (Model, tea.Cmd) {
	row, ok := m.dashboard.selectedRun()
	if !ok {
		return m, nil
	}
	if err := m.runHistoryStore.DeleteRun(row.entry); err != nil {
		m.status = "Delete failed: " + err.Error()
		return m, nil
	}
	runs, err := m.runHistoryStore.ListRuns()
	if err != nil {
		m.status = "Delete failed: " + err.Error()
		return m, nil
	}
	m.dashboard.runs = runs
	if shown := m.dashboard.shownRuns(); m.dashboard.cursor >= len(shown) && m.dashboard.cursor > 0 {
		m.dashboard.cursor--
	}
	m.status = "Run deleted"
	return m, nil
}

// dashboardHeader matches every other screen's title convention (see
// settingsHeader) - used only by the empty state; the populated view has
// its own "RUN DASHBOARD" title (see dashboard_view.go).
func dashboardHeader() string {
	return activeTabStyle.Render("Run dashboard") + labelStyle.Render("  (esc close)")
}

// dashboardRunLabel is a run's display label: its recorded folder/request
// name (already a friendly display name, not a raw store path - see
// runner.go's recordedPath), or "(whole collection)" for a whole-collection
// run (Path is "" then).
func dashboardRunLabel(run history.CollectionRunEntry) string {
	if run.Path == "" {
		return "(whole collection)"
	}
	return run.Path
}

// requestPassed reports whether a request's result counts as passing: it
// sent without error and every pm.test assertion in it (if any) passed -
// the same rule internal/cli.Success uses for `parley run`'s exit code.
func requestPassed(r execution.RunResult) bool {
	if r.Err != "" {
		return false
	}
	for _, tr := range r.Tests {
		if !tr.Passed {
			return false
		}
	}
	return true
}

// runPassed applies requestPassed to every request in a run: a run only
// counts as passed if all of them did. A run with no results at all (should
// not happen in practice - startCollectionRun refuses to run zero requests)
// counts as failed rather than vacuously passed.
func runPassed(run history.CollectionRunEntry) bool {
	if len(run.Results) == 0 {
		return false
	}
	for _, r := range run.Results {
		if !requestPassed(r) {
			return false
		}
	}
	return true
}

// dashboardOverview is the Overview section's five aggregate stats, computed
// across every recorded run (not just whatever's currently shown/filtered
// below it) - a stable, always-global summary.
type dashboardOverview struct {
	total, passed, failed int
	passRate              float64 // percent, 0-100
	avgMS                 int64
}

func computeDashboardOverview(runs []history.CollectionRunEntry) dashboardOverview {
	var ov dashboardOverview
	ov.total = len(runs)
	var totalMS int64
	for _, r := range runs {
		if runPassed(r) {
			ov.passed++
		} else {
			ov.failed++
		}
		totalMS += r.TotalMS
	}
	if ov.total > 0 {
		ov.passRate = float64(ov.passed) * 100 / float64(ov.total)
		ov.avgMS = totalMS / int64(ov.total)
	}
	return ov
}

// formatDurationMS is "Nms" under a second, "X.XXs" from a second up - the
// Overview/run-total figures read as fractional seconds in the mockup this
// screen follows, while individual request latencies elsewhere stay in ms
// (see dashboardRequestLine/computeDashboardPerfRows) - both conventions
// already exist independently elsewhere in the app, this just picks one
// per figure the way the mockup does.
func formatDurationMS(ms int64) string {
	if ms >= 1000 {
		return fmt.Sprintf("%.2fs", float64(ms)/1000)
	}
	return fmt.Sprintf("%dms", ms)
}
