package tui

import (
	"fmt"
	history "github.com/dhitalkamal/parley/internal/history/domain"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// maxHistoryEntries caps how many past sends the history modal shows.
const maxHistoryEntries = 50

type historyState struct {
	active  bool
	entries []history.HistoryEntry
	cursor  int
}

func (m Model) openHistory() (Model, tea.Cmd) {
	entries, err := m.historyStore.ListHistory()
	if err != nil {
		m.status = "History load failed: " + err.Error()
		return m, nil
	}
	if len(entries) > maxHistoryEntries {
		entries = entries[:maxHistoryEntries]
	}
	m.history = historyState{active: true, entries: entries}
	return m, nil
}

func (m Model) handleHistoryKey(k tea.KeyMsg) (Model, tea.Cmd) {
	switch k.String() {
	case "esc", "ctrl+y":
		m.history.active = false
		return m, nil
	case "up", "k":
		if m.history.cursor > 0 {
			m.history.cursor--
		}
		return m, nil
	case "down", "j":
		if m.history.cursor < len(m.history.entries)-1 {
			m.history.cursor++
		}
		return m, nil
	case "enter":
		if len(m.history.entries) == 0 {
			return m, nil
		}
		entry := m.history.entries[m.history.cursor]
		m.history.active = false
		m.status = "Re-running..."
		m.beginSending()
		return m, tea.Batch(sendReqCmd(m.client, entry.Request, entry.Request, false), sendTickCmd())
	}
	return m, nil
}

// historyHeader is the same title+esc-hint convention every modal in the app
// uses (see envPanelState.View's "Manage Environments  (esc close)").
func historyHeader() string {
	return activeTabStyle.Render("History") + labelStyle.Render("  (enter re-run - esc close)")
}

func (h historyState) View() string {
	if len(h.entries) == 0 {
		return borderStyle.Render(historyHeader() + "\n" + labelStyle.Render("No history yet"))
	}
	var b strings.Builder
	for i, e := range h.entries {
		cursor := "  "
		if i == h.cursor {
			cursor = "> "
		}
		statusText := e.Status
		if e.Err != "" {
			statusText = "ERROR: " + e.Err
		}
		// Method is colored after padding, not before - padding an
		// already-ANSI-wrapped string with %-6s would count escape bytes
		// toward the width and misalign every column after it.
		method := methodBadgeStyle(string(e.Request.Method)).Render(fmt.Sprintf("%-6s", e.Request.Method))
		fmt.Fprintf(&b, "%s%s %-40s %-16s %dms\n",
			cursor, method, truncate(e.Request.URL, 40), statusText, e.ElapsedMS)
	}
	return borderStyle.Render(historyHeader() + "\n" + strings.TrimRight(b.String(), "\n"))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
}
