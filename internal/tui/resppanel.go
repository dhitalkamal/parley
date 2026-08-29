package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// responsePanelView renders the Response panel's bordered box: the tab bar
// (body/headers/cookies/tests/timeline) on top and the viewport content below,
// both full width. The status/timing/size meta lives on the panel's title bar
// (see responseStatusMeta, spliced onto the border in workspace_view.go), so
// this body no longer carries a status row or a side column.
func (m Model) responsePanelView(width, height int) string {
	innerWidth := width - 4
	innerHeight := height - 2

	m.response.SetSize(innerWidth, height)
	var topLine string
	topRows := 0
	if m.sending {
		topLine = sendingStatusLine(m.sendStartedAt)
		topRows = 1
	} else if m.response.HasContent() {
		topLine = m.response.TabBarText()
		topRows = lipgloss.Height(topLine)
	}

	vpHeight := innerHeight - topRows
	if vpHeight < 3 {
		vpHeight = 3
	}
	m.response.SetSize(innerWidth, vpHeight)

	content := m.response.ContentView()
	if topLine != "" {
		content = topLine + "\n" + content
	}

	style := borderStyle
	if m.effectiveFocus() == focusResponse {
		style = focusedBorder
	}
	return style.Width(width - 2).Height(height - 2).Render(content)
}

// responseStatusMeta is the "404 Not Found  |  1218 ms  |  19 bytes" string
// shown on the Response panel's title border (right-aligned, see
// workspace_view.go): the status code in its class color, timing and size dim.
// Empty while sending, before a response, or on a bare network error - the body
// shows those cases full width.
func (m Model) responseStatusMeta() string {
	rv := m.response
	if m.sending || rv.status == "" || rv.err != nil {
		return ""
	}
	return statusClassStyle(rv.statusCode).Render(rv.status) +
		labelStyle.Render(fmt.Sprintf("  |  %d ms  |  %s", rv.elapsedMS, humanSize(rv.sizeBytes)))
}

// sendingStatusLine shows a live-ticking elapsed time while a request is in
// flight - accent-colored (like an active tab) rather than the dim label
// style "Ready"/errors use, so it reads as "in progress" at a glance.
func sendingStatusLine(startedAt time.Time) string {
	elapsedMS := time.Since(startedAt).Milliseconds()
	return activeTabStyle.Render(fmt.Sprintf("Sending...  %d ms", elapsedMS))
}
