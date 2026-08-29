package tui

import (
	"fmt"
	"strings"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	history "github.com/dhitalkamal/parley/internal/history/domain"

	"github.com/charmbracelet/lipgloss"
)

// maxAboutPanelHistory caps the Recent History list the same
// surfaced-not-silent way every other capped list in this app does - see
// maxDashboardRunsShown's own doc comment.
const maxAboutPanelHistory = 5

// aboutPanelView renders the Collections screen's right-hand panel: a
// read-only summary of whichever request is currently selected in the
// sidebar tree, plus its own recent send history - so browsing the
// collection surfaces what a request does and how it's been behaving
// without opening it into the full Request screen first.
func (m Model) aboutPanelView(width, height int) string {
	item, ok := m.sidebar.Selected()
	if !ok || item.kind != collection.KindRequest {
		return aboutPanelPlaceholder(width, height)
	}
	req, err := m.store.LoadRequest(item.path)
	if err != nil {
		return aboutPanelPlaceholder(width, height)
	}

	matching := historyForPath(m.historyStore, item.path, maxAboutPanelHistory)

	var b strings.Builder
	fmt.Fprintln(&b, activeTabStyle.Render("About "+item.name))
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "%s  %s\n", methodBadgeStyle(string(req.Method)).Render(string(req.Method)), req.URL)
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, labelStyle.Render(authSummaryLine(req)))
	fmt.Fprintln(&b, labelStyle.Render(testsSummaryLine(req)))
	fmt.Fprintln(&b, labelStyle.Render(lastRunSummaryLine(matching)))

	fmt.Fprintln(&b)
	fmt.Fprintln(&b, activeTabStyle.Render("Recent History"))
	if len(matching) == 0 {
		fmt.Fprintln(&b, labelStyle.Render("(never run)"))
	}
	for _, e := range matching {
		fmt.Fprintln(&b, historyLineFor(e))
	}

	style := borderStyle
	return style.Width(width - 2).Height(height - 2).Render(strings.TrimRight(b.String(), "\n"))
}

// aboutPanelPlaceholder is shown before anything request-shaped is
// selected - nothing selected at all, or a folder, which has no method/
// URL/tests/history of its own to summarize.
func aboutPanelPlaceholder(width, height int) string {
	msg := labelStyle.Render("Select a request to see its details")
	return borderStyle.Width(width - 2).Height(height - 2).
		Render(lipgloss.Place(width-4, height-2, lipgloss.Center, lipgloss.Center, msg))
}

// authSummaryLine describes the request's auth-capture configuration (the
// Auth tab's actual feature - capturing and auto-refreshing a bearer token
// from a login response, not a manual Basic/Bearer/API-key picker), which is
// the only auth-shaped data collection.Request actually carries.
func authSummaryLine(req collection.Request) string {
	if !req.AuthCapture.Enabled() {
		return "Auth: none"
	}
	line := fmt.Sprintf("Auth: capture -> %s", req.AuthCapture.TokenVar)
	if req.Refresh.Enabled() {
		line += " (auto-refresh)"
	}
	return line
}

// testsSummaryLine reports whether a test script exists at all - a count of
// individual pm.test() assertions would need parsing the script itself,
// which this panel deliberately doesn't do.
func testsSummaryLine(req collection.Request) string {
	if strings.TrimSpace(req.TestScript) == "" {
		return "Tests: none"
	}
	return "Tests: configured"
}

// lastRunSummaryLine describes the most recent (matching is newest-first)
// send for this request, or "never run" before the first one.
func lastRunSummaryLine(matching []history.HistoryEntry) string {
	if len(matching) == 0 {
		return "Last run: never run"
	}
	last := matching[0]
	when := last.Time.Format("2006-01-02 15:04:05")
	if last.Err != "" {
		return fmt.Sprintf("Last run: %s - error: %s", when, last.Err)
	}
	return fmt.Sprintf("Last run: %s - %s", when, last.Status)
}

// historyLineFor renders one Recent History row.
func historyLineFor(e history.HistoryEntry) string {
	when := e.Time.Format("2006-01-02 15:04:05")
	if e.Err != "" {
		return labelStyle.Render(when) + "  " + errStyle.Render("ERROR: "+e.Err)
	}
	return labelStyle.Render(when) + "  " + statusClassStyle(e.StatusCode).Render(e.Status) +
		labelStyle.Render(fmt.Sprintf("  %dms", e.ElapsedMS))
}

// historyForPath loads every history entry and filters it down to the ones
// recorded against path, already-newest-first (see HistoryStore.ListHistory),
// capped at limit - the Collections screen's per-request narrowing of the
// same log the History overlay (ctrl+y) shows in full.
func historyForPath(store history.HistoryStore, path string, limit int) []history.HistoryEntry {
	all, err := store.ListHistory()
	if err != nil {
		return nil
	}
	var matching []history.HistoryEntry
	for _, e := range all {
		if e.RequestPath != path {
			continue
		}
		matching = append(matching, e)
		if len(matching) == limit {
			break
		}
	}
	return matching
}
