package tui

import (
	"fmt"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	"strings"
)

// runResultLine renders one executed request's outcome - shared by the
// collection runner's own live summary (runner.go) and the dashboard's Run
// Details drill-down (dashboard.go), which show the same per-request
// information for a just-finished run and a previously recorded one.
func runResultLine(r execution.RunResult) string {
	// Method is colored after padding, not before - see history.go's View
	// for why (padding an already-ANSI-wrapped string misaligns columns).
	method := methodBadgeStyle(string(r.Method)).Render(fmt.Sprintf("%-6s", r.Method))
	line := fmt.Sprintf("%s %-40s", method, truncate(r.URL, 40))
	switch {
	case r.Err != "":
		line += "  " + errStyle.Render("ERROR: "+r.Err)
	default:
		line += "  " + statusClassStyle(r.StatusCode).Render(r.Status) + labelStyle.Render(fmt.Sprintf("  %dms", r.ElapsedMS))
		if len(r.Tests) > 0 {
			p := 0
			for _, tr := range r.Tests {
				if tr.Passed {
					p++
				}
			}
			line += labelStyle.Render(fmt.Sprintf("  tests %d/%d", p, len(r.Tests)))
		}
	}
	return line
}

// runResultsSummary is the "X/Y requests ok[, A/B tests passed], Nms total"
// line shared by the runner and the dashboard's Run Details view.
func runResultsSummary(results []execution.RunResult, totalMS int64) string {
	passedReqs, totalTests, passedTests := 0, 0, 0
	for _, r := range results {
		if r.Err == "" {
			passedReqs++
		}
		for _, tr := range r.Tests {
			totalTests++
			if tr.Passed {
				passedTests++
			}
		}
	}
	summary := fmt.Sprintf("%d/%d requests ok", passedReqs, len(results))
	if totalTests > 0 {
		summary += fmt.Sprintf(", %d/%d tests passed", passedTests, totalTests)
	}
	summary += fmt.Sprintf(", %dms total", totalMS)
	return summary
}

// runResultsBody renders results as one runResultLine per row, capped at
// max with an explicit "...and N more" overflow line - never silently
// truncated (see maxRunnerResultsShown/maxDashboardRunsShown's own doc
// comments for why).
func runResultsBody(results []execution.RunResult, max int) string {
	shown := results
	var truncatedBy int
	if len(shown) > max {
		truncatedBy = len(shown) - max
		shown = shown[:max]
	}
	var b strings.Builder
	for _, r := range shown {
		fmt.Fprintln(&b, runResultLine(r))
	}
	if truncatedBy > 0 {
		fmt.Fprintf(&b, "... and %d more\n", truncatedBy)
	}
	return strings.TrimRight(b.String(), "\n")
}
