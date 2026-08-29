package tui

import (
	"fmt"
	"strings"

	history "github.com/dhitalkamal/parley/internal/history/domain"
)

// dashboardPerfRow is one request's aggregate across a window of recent
// runs: its label, pass rate within the window, every latency in the window
// (oldest to newest, for the sparkline), and the most recent one.
type dashboardPerfRow struct {
	label    string
	passRate int
	ms       []int64
	lastMS   int64
}

// computeDashboardPerfRows groups every result across the most recent
// window runs (out of runs, which is newest-first) by RequestPath, so a
// regression (a request getting less reliable or slower) is visible without
// opening each run.
func computeDashboardPerfRows(runs []history.CollectionRunEntry, window int) []dashboardPerfRow {
	if len(runs) > window {
		runs = runs[:window]
	}

	type point struct {
		passed bool
		label  string
		ms     int64
	}
	order := []string{}
	byPath := map[string][]point{}
	// Walk oldest-to-newest (runs is newest-first) so each path's sparkline
	// reads left-to-right as a timeline ending at its most recent run.
	for i := len(runs) - 1; i >= 0; i-- {
		for _, r := range runs[i].Results {
			if _, ok := byPath[r.RequestPath]; !ok {
				order = append(order, r.RequestPath)
			}
			label := strings.TrimSpace(fmt.Sprintf("%s %s", r.Method, r.URL))
			byPath[r.RequestPath] = append(byPath[r.RequestPath], point{passed: requestPassed(r), label: label, ms: r.ElapsedMS})
		}
	}

	rows := make([]dashboardPerfRow, 0, len(order))
	for _, path := range order {
		points := byPath[path]
		passed := 0
		ms := make([]int64, len(points))
		for i, p := range points {
			if p.passed {
				passed++
			}
			ms[i] = p.ms
		}
		rows = append(rows, dashboardPerfRow{
			label:    points[len(points)-1].label,
			passRate: passed * 100 / len(points),
			ms:       ms,
			lastMS:   ms[len(ms)-1],
		})
	}
	return rows
}

// dashboardPerformanceSection renders the "Request Performance" table as a
// standalone, full-width section (below the Run History/Selected Run split -
// see dashboardScreenView): its own label, then one row per request in the
// window, capped at maxRows with an explicit "...and N more" overflow line,
// never silently.
func (m Model) dashboardPerformanceSection(width, maxRows int) string {
	var b strings.Builder
	fmt.Fprintln(&b, labelStyle.Render(fmt.Sprintf("REQUEST PERFORMANCE - LAST %d RUNS", trendRunWindow)))
	fmt.Fprintln(&b)

	rows := computeDashboardPerfRows(m.dashboard.runs, trendRunWindow)
	shown := rows
	truncatedBy := 0
	if len(shown) > maxRows {
		truncatedBy = len(shown) - maxRows
		shown = shown[:maxRows]
	}
	for _, row := range shown {
		line := fmt.Sprintf("%-30s  %3d%%  %s  %s",
			truncate(row.label, 30), row.passRate, latencySparkline(row.ms), formatDurationMS(row.lastMS))
		fmt.Fprintln(&b, clipLine(line, width))
	}
	if truncatedBy > 0 {
		fmt.Fprintf(&b, "... and %d more\n", truncatedBy)
	}
	return strings.TrimRight(b.String(), "\n")
}
