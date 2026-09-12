package tui

import (
	"fmt"
	"strings"

	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	history "github.com/dhitalkamal/parley/internal/history/domain"

	"github.com/charmbracelet/lipgloss"
)

// dashboardStatColWidth sizes the Overview section's five stat columns
// (Runs/Passed/Failed/Pass Rate/Avg Duration) - fixed rather than measured,
// since the header labels and their values are both known in advance.
const dashboardStatColWidth = 16

// dashboardOverviewFixedHeight is exactly how many lines dashboardOverviewLines
// always returns - a pure aggregate, not a per-item list, so unlike the run
// log/performance table below it this never varies with how much history
// exists, and dashboardScreenView's own budget below relies on that.
const dashboardOverviewFixedHeight = 4

// dashboardScreenView renders the Dashboard as a full frame: the global top
// bar, a title, the Overview stat row, the Run History/Selected Run split
// (on a wide enough terminal) with Request Performance beneath it, and its
// own status bar - matching every other screen's header+content+status-bar
// shape. Every section's height is either fixed (Overview) or explicitly
// clipped to its own computed budget (clipToLines) rather than assumed, so
// the whole frame renders exactly m.height lines regardless of how much
// history exists - see clipToLines' own doc comment for why that matters.
func (m Model) dashboardScreenView() string {
	status := labelStyle.Render(dashboardStatusBar)

	if len(m.dashboard.runs) == 0 {
		content := borderStyle.Render(dashboardHeader() + "\n" + labelStyle.Render("No runs recorded yet"))
		return m.screenFrame(content, status)
	}

	contentHeight := m.screenContentHeight()

	if m.width < minWidthForDashboardSplit {
		return m.dashboardNarrowView(status, contentHeight)
	}

	// fixedTop: title(1) + divider(1) + blank(1) + overview(dashboardOverviewFixedHeight) + blank(1).
	const fixedTop = 4 + dashboardOverviewFixedHeight
	const minSplitHeight = 6
	const minPerfHeight = 4

	available := contentHeight - fixedTop
	splitHeight := available
	perfHeight := 0
	if available-1-minSplitHeight >= minPerfHeight {
		candidate := (available - 1) * 3 / 10
		if candidate >= minPerfHeight {
			perfHeight = candidate
			splitHeight = available - 1 - perfHeight
		}
	}
	if splitHeight < 1 {
		splitHeight = 1
	}

	lines := []string{
		clipLine(activeTabStyle.Render("RUN DASHBOARD"), m.width),
		strings.Repeat("-", m.width),
		"",
	}
	lines = append(lines, dashboardOverviewLines(m.dashboard.runs)...)
	lines = append(lines, "")
	body := strings.Join(lines, "\n") + "\n" + m.dashboardSplitView(m.width, splitHeight)
	if perfHeight > 0 {
		body += "\n\n" + clipToLines(m.dashboardPerformanceSection(m.width, perfHeight-1), perfHeight)
	}

	return m.screenFrame(body, status)
}

// dashboardNarrowView is the fallback below minWidthForDashboardSplit: one
// bordered column stacking the title, Overview, Run History, and Request
// Performance - no Selected Run pane, matching the graceful-degradation
// floor pattern every other side-by-side layout in this app uses.
func (m Model) dashboardNarrowView(status string, contentHeight int) string {
	innerWidth := m.width - 4
	innerHeight := contentHeight - 2

	lines := []string{
		clipLine(activeTabStyle.Render("RUN DASHBOARD"), innerWidth),
		strings.Repeat("-", innerWidth),
		"",
	}
	lines = append(lines, dashboardOverviewLines(m.dashboard.runs)...)
	lines = append(lines, "", labelStyle.Render("RUN HISTORY"), "")
	lines = append(lines, strings.Split(m.dashboardRunHistoryContent(innerWidth, maxDashboardRunsShown), "\n")...)
	lines = append(lines, "", m.dashboardPerformanceSection(innerWidth, 8))

	content := clipToLines(strings.Join(lines, "\n"), innerHeight)
	box := borderStyle.Width(m.width - 2).Height(innerHeight).Render(content)
	return m.screenFrame(box, status)
}

// dashboardOverviewLines is always exactly dashboardOverviewFixedHeight
// lines: the section label, a blank, and a header+value row pair of the
// five aggregate stats - see computeDashboardOverview.
func dashboardOverviewLines(runs []history.CollectionRunEntry) []string {
	ov := computeDashboardOverview(runs)
	header := fmt.Sprintf("%-*s%-*s%-*s%-*s%-*s",
		dashboardStatColWidth, "Runs", dashboardStatColWidth, "Passed", dashboardStatColWidth, "Failed",
		dashboardStatColWidth, "Pass Rate", dashboardStatColWidth, "Avg Duration")
	values := fmt.Sprintf("%-*d%-*d%-*d%-*s%-*s",
		dashboardStatColWidth, ov.total, dashboardStatColWidth, ov.passed, dashboardStatColWidth, ov.failed,
		dashboardStatColWidth, fmt.Sprintf("%.1f%%", ov.passRate), dashboardStatColWidth, formatDurationMS(ov.avgMS))
	return []string{
		labelStyle.Render("OVERVIEW"),
		"",
		labelStyle.Render(strings.TrimRight(header, " ")),
		strings.TrimRight(values, " "),
	}
}

// dashboardSplitView is the Run History/Selected Run split, each pane given
// exactly height rows - both built via the same border(2)+content(height-2)
// shape every other bordered zone in this app uses, so lipgloss.JoinHorizontal
// never has to pad one side taller than the other.
func (m Model) dashboardSplitView(width, height int) string {
	leftWidth := width / 2
	rightWidth := width - leftWidth - 1
	leftContentWidth := leftWidth - 4
	innerHeight := height - 2
	if innerHeight < 1 {
		innerHeight = 1
	}

	leftContent := clipToLines(m.dashboardRunHistoryContent(leftContentWidth, innerHeight), innerHeight)
	leftBody := borderStyle.Width(leftWidth - 2).Height(innerHeight).Render(leftContent)
	leftBox := titledBox(leftBody, zoneHeaderText("Run History", true, true))

	rightBox := m.dashboardSelectedRunPane(rightWidth, height)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftBox, " ", rightBox)
}

// dashboardRunHistoryContent is the run log: one line per visible run (see
// dashboardRunLine), cursor-highlighted. Two independent limits apply: the
// hard maxDashboardRunsShown cap (surfaced with an explicit "...and N more"
// overflow line, never silently - see its own doc comment), and maxRows,
// however many rows the pane's current height actually has room for. When
// maxRows can't fit every capped row, the window scrolls to keep the
// cursor visible instead of just chopping off whatever's past the fold, the
// same as any ordinary scrollable list - only the hard cap's own overflow
// is explicit, not this ordinary scrolling. Every line is clipped (not left
// to wrap) to width, the same class of bug response.go's StatusLine guards
// against.
func (m Model) dashboardRunHistoryContent(width, maxRows int) string {
	all := m.dashboard.filteredRuns()
	rows := m.dashboard.shownRuns()
	truncatedByCap := len(all) - len(rows)

	capacity := maxRows
	if truncatedByCap > 0 && capacity > 1 {
		capacity-- // reserve a row for the cap's own overflow line
	}
	if capacity < 1 {
		capacity = 1
	}
	start := 0
	if len(rows) > capacity {
		start = m.dashboard.cursor - capacity + 1
		if start < 0 {
			start = 0
		}
		if maxStart := len(rows) - capacity; start > maxStart {
			start = maxStart
		}
		rows = rows[start : start+capacity]
	}

	var b strings.Builder
	for i, row := range rows {
		fmt.Fprintln(&b, clipLine(dashboardRunLine(row, start+i == m.dashboard.cursor), width))
	}
	if truncatedByCap > 0 {
		fmt.Fprintf(&b, "... and %d more\n", truncatedByCap)
	}
	return strings.TrimRight(b.String(), "\n")
}

// dashboardRunLine renders one run log row: its stable sequence number,
// label, passed/failed request counts, and total duration.
func dashboardRunLine(row dashboardRunRow, selected bool) string {
	cursor := "  "
	if selected {
		cursor = "> "
	}
	passed := 0
	for _, r := range row.entry.Results {
		if requestPassed(r) {
			passed++
		}
	}
	failed := len(row.entry.Results) - passed
	return fmt.Sprintf("%s#%-4d %-24s %d/%-4d %s",
		cursor, row.number, truncate(dashboardRunLabel(row.entry), 24), passed, failed, formatDurationMS(row.entry.TotalMS))
}

// dashboardSelectedRunPane is the right column: every request in whichever
// run the cursor currently sits on, live - moving the cursor in the left
// column updates this pane immediately, no separate drill-down step.
func (m Model) dashboardSelectedRunPane(width, height int) string {
	innerHeight := height - 2
	if innerHeight < 1 {
		innerHeight = 1
	}
	body := labelStyle.Render("Select a run to see its details")
	if row, ok := m.dashboard.selectedRun(); ok {
		// dashboardRunHeaderLines: label(1)+blank(1)+4 stat rows+blank(1)+
		// "REQUESTS"(1)+blank(1) - everything dashboardSelectedRunContent
		// renders before its per-request rows.
		const dashboardRunHeaderLines = 9
		maxReqRows := innerHeight - dashboardRunHeaderLines
		if maxReqRows < 0 {
			maxReqRows = 0
		}
		body = dashboardSelectedRunContent(row, width-4, maxReqRows)
	}
	body = clipToLines(body, innerHeight)
	boxed := borderStyle.Width(width - 2).Height(innerHeight).Render(body)
	return titledBox(boxed, zoneHeaderText("Selected Run", true, false))
}

// dashboardSelectedRunContent is a single run's full detail: its label and
// number, Started/Duration/Passed/Failed stat rows, then every request in
// it under a REQUESTS heading (see dashboardRequestLine) - reusing the same
// per-result data the just-finished collection runner shows (runner.go), a
// past run and a live one are the same kind of data. Requests are capped at
// maxRequestRows with an explicit "...and N more" overflow line, the same
// never-silently convention as the run log's own maxDashboardRunsShown cap -
// a run with more requests than fit shouldn't just quietly lose the tail.
func dashboardSelectedRunContent(row dashboardRunRow, width, maxRequestRows int) string {
	run := row.entry
	passed := 0
	for _, r := range run.Results {
		if requestPassed(r) {
			passed++
		}
	}
	failed := len(run.Results) - passed

	var b strings.Builder
	fmt.Fprintf(&b, "#%d  %s\n\n", row.number, dashboardRunLabel(run))
	fmt.Fprintf(&b, "%-14s%s\n", "Started", run.Time.Format("15:04:05"))
	fmt.Fprintf(&b, "%-14s%s\n", "Duration", formatDurationMS(run.TotalMS))
	fmt.Fprintf(&b, "%-14s%d\n", "Passed", passed)
	fmt.Fprintf(&b, "%-14s%d\n\n", "Failed", failed)
	fmt.Fprintln(&b, labelStyle.Render("REQUESTS"))
	fmt.Fprintln(&b)

	results := run.Results
	truncatedBy := 0
	if maxRequestRows >= 0 && len(results) > maxRequestRows {
		capacity := maxRequestRows
		if capacity > 0 {
			capacity-- // reserve a row for the overflow line
		}
		truncatedBy = len(results) - capacity
		results = results[:capacity]
	}
	for _, r := range results {
		fmt.Fprintln(&b, clipLine(dashboardRequestLine(r), width))
	}
	if truncatedBy > 0 {
		fmt.Fprintf(&b, "... and %d more\n", truncatedBy)
	}
	return strings.TrimRight(b.String(), "\n")
}

// dashboardRequestLine renders one request's outcome within a run: a plain
// ASCII pass/fail marker (this project avoids decorative Unicode glyphs -
// see zoneChevron's own doc comment for the same call elsewhere), method,
// URL, latency, and its tests-passed ratio if it ran any.
func dashboardRequestLine(r execution.RunResult) string {
	mark := "[OK] "
	if !requestPassed(r) {
		mark = "[X]  "
	}
	method := methodBadgeStyle(string(r.Method)).Render(fmt.Sprintf("%-6s", r.Method))
	name := fmt.Sprintf("%-30s", truncate(r.URL, 30))
	if r.Err != "" {
		return mark + method + " " + name + "  " + errStyle.Render("ERROR: "+r.Err)
	}
	line := fmt.Sprintf("%s%s %s  %6s", mark, method, name, formatDurationMS(r.ElapsedMS))
	if len(r.Tests) > 0 {
		p := 0
		for _, tr := range r.Tests {
			if tr.Passed {
				p++
			}
		}
		line += fmt.Sprintf("  %d/%d", p, len(r.Tests))
	}
	return line
}
