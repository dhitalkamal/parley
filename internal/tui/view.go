package tui

import (
	"fmt"
	collection "github.com/dhitalkamal/parley/internal/collection/domain"

	"github.com/charmbracelet/lipgloss"
)

// topBarHeight/urlRowHeight are the fixed-height rows above the 3-panel grid.
// helpBarHeight assumes the collapsed (default) help bar - pressing "?" to
// expand it isn't accounted for, the same kind of minor approximation every
// render budget in this file already makes. tabStripHeight is kept at 0 (the
// top tab strip was removed - navigation lives in the command palette and the
// dedicated keys now) so all the layout/mouse math that still adds it stays
// self-consistent without touching every call site.
const (
	// topBarHeight is 0: the top bar (brand/theme/clock/workspace/env) was
	// removed on request. Workspace switching lives in the command palette
	// (ctrl+k) and F4, theme in the palette, and the environment in the rail
	// (e) and the url-row env box (ctrl+e). Kept as a named 0 so the layout
	// and mouse math that still adds it stays self-consistent.
	topBarHeight   = 0
	tabStripHeight = 0
	urlRowHeight   = 3
	// helpBarHeight is the footer/status row the full-screen views (dashboard,
	// collections, settings) still render at their bottom. The Request screen
	// no longer has any bottom bar (removed on request - see panelContentHeight
	// and mainView), so it does NOT subtract this; full key help there is a "?"
	// overlay and the command palette instead.
	helpBarHeight = 1
)

// urlRowWidth is how wide the method+url+send row renders - it spans the
// full terminal width above the 3-panel grid, not just what's left after
// the sidebar (the sidebar starts below it, not beside it).
func urlRowWidth(totalWidth int) int {
	w := totalWidth - 3
	if w < 20 {
		w = 20
	}
	return w
}

// panelContentHeight is how many rows the workspace region gets on the Request
// screen once the top bar and url row are subtracted. It deliberately does NOT
// subtract helpBarHeight: the Request screen has no bottom bar anymore, so the
// workspace runs all the way to the last row.
func panelContentHeight(totalHeight int) int {
	h := totalHeight - topBarHeight - tabStripHeight - urlRowHeight
	if h < 10 {
		h = 10
	}
	return h
}

// paletteModalWidth is a fixed width for the command palette modal -
// independent of the 3-panel grid, since it floats centered over the whole
// screen rather than living inside any one panel.
const paletteModalWidth = 70

// methodBoxContentWidth/sendBoxWidth size the url row's fixed boxes (method
// and send) as separate bordered elements instead of one shared border - each
// needs its own visible focus state, and a fixed width keeps the row from
// jiggling as their text changes length. The environment is no longer one of
// these boxes; it lives in its own panel now.
const (
	methodBoxContentWidth = 9
	sendBoxWidth          = 10
)

func methodBoxOuterWidth() int { return methodBoxContentWidth + 2 }

// urlBoxOuterWidth is whatever's left of the url row once the fixed-width
// method and send boxes, and the two 1-column gaps between the three boxes,
// are subtracted.
func urlBoxOuterWidth(totalWidth int) int {
	w := totalWidth - methodBoxOuterWidth() - sendBoxWidth - 2
	if w < 12 {
		w = 12
	}
	return w
}

// View renders the full frame, layering floating pieces on top of the
// still-rendered background one at a time (see overlay.go) instead of
// blanking the screen to a backdrop - lipgloss has no built-in layer
// compositing, so each layer is spliced on by hand. History/runner/palette/
// merged-vars/code-snippet/environments are mutually exclusive (root.go's
// handleKey routes to at most one at a time), so exactly one of them - if
// any - composites onto the background before confirm/prompt, which can
// still float on top of any of them (e.g. "delete this environment?" opened
// from inside the environments modal).
func (m Model) View() string {
	var background string
	switch m.screen {
	case ScreenCollections:
		background = m.collectionsScreenView()
	case ScreenDashboard:
		background = m.dashboardScreenView()
	case ScreenSettings:
		background = m.settingsScreenView()
	case ScreenWebSocket:
		background = m.wsView()
	default: // ScreenRequest
		background = m.mainView()
	}

	// Vim mode indicator, bottom-right of the Request screen (hidden while a
	// modal owns input - the mode doesn't apply then).
	if m.screen == ScreenRequest && !m.modalActive() {
		ind := m.modeIndicator()
		background = overlayAt(background, ind, m.width, m.height, m.width-lipgloss.Width(ind), m.height-1)
	}

	switch {
	case m.envPanel.active:
		background = overlay(background, m.envPanel.View(m.activeEnvName), m.width, m.height)
	case m.envDropdown.active:
		x, y := m.envDropdownAnchor()
		background = overlayAt(background, m.envDropdown.View(m.activeEnvName), m.width, m.height, x, y)
	case m.bodyTypeDropdown.active:
		x, y := m.bodyTypeDropdownAnchor()
		background = overlayAt(background, m.bodyTypeDropdown.View(bodyTypeLabels), m.width, m.height, x, y)
	case m.contentTypeDropdown.active:
		x, y := m.contentTypeDropdownAnchor()
		background = overlayAt(background, m.contentTypeDropdown.View(rawContentTypes), m.width, m.height, x, y)
	case m.workspace.active:
		background = overlay(background, m.workspace.View(m.activeWorkspaceName), m.width, m.height)
	case m.history.active:
		background = overlay(background, m.history.View(), m.width, m.height)
	case m.runner.active:
		background = overlay(background, m.runnerView(), m.width, m.height)
	case m.palette.active:
		background = overlay(background, m.palette.View(paletteModalWidth), m.width, m.height)
	case m.showVariables:
		background = overlay(background, m.variablesPanelView(), m.width, m.height)
	case m.showCodeSnippet:
		// Cap the snippet to whatever the terminal can actually show - a
		// request with many headers or a large body can otherwise produce
		// curl+Go output taller than the screen itself.
		maxLines := m.height - 6
		if maxLines < 5 {
			maxLines = 5
		}
		header := activeTabStyle.Render("Code Snippet") + labelStyle.Render("  (esc close)")
		background = overlay(background, borderStyle.Render(header+"\n"+truncateLines(m.codeSnippetView(), maxLines)), m.width, m.height)
	case m.help.ShowAll:
		// Full key help is a centered overlay now, opened with "?" - the old
		// always-on bottom help bar was removed on request.
		m.help.Width = m.width - 8
		header := activeTabStyle.Render("Keyboard Shortcuts") + labelStyle.Render("  (? or esc to close)")
		background = overlay(background, borderStyle.Render(header+"\n"+m.help.View(keys)), m.width, m.height)
	}

	// kvAdd floats on top of the switch above rather than being one of its
	// cases - it can open either over the plain background (adding a param/
	// header) or over an already-open envPanel (adding a variable).
	if m.kvAdd.active {
		background = overlay(background, m.kvAdd.View(), m.width, m.height)
	}

	if m.confirm.active {
		return overlay(background, m.confirm.View(), m.width, m.height)
	}
	if m.prompt.active {
		return overlay(background, m.prompt.View(), m.width, m.height)
	}
	return background
}

// envDropdownAnchor positions the quick-switch dropdown just under the
// environment pill in the top bar - the pill is the primary switcher now
// (the url row keeps a color-matched echo instead, see urlRowView). x
// mirrors where colorizeEnvSegment finds and colors that same segment, so
// the dropdown always opens directly under it regardless of workspace/
// theme name length shifting its position. The reference time passed to
// topBarContent doesn't affect this - the clock segment is always exactly
// 8 characters ("15:04:05" - width, not value), so the pill's position
// never drifts with the actual time.
func (m Model) envDropdownAnchor() (x, y int) {
	// Aligned with the Active selector line (row 1: past the panel's top
	// border), and opened to the LEFT of the Environment panel rather than
	// inside it - a dropdown dropping down within the panel would cover the
	// panel's own variables, which is exactly what you want to keep seeing
	// while picking an environment. Falls back to the top-left when the panel
	// isn't on screen (terminal too narrow).
	if !m.leftSidebarShown() || !m.environmentExpanded() {
		return 0, topBarHeight + tabStripHeight + 1
	}
	// The Environment panel sits at the bottom of the left column now; anchor
	// the quick-switch dropdown just to its right, a few rows into the section
	// (past the VPN line and the Active selector), so it doesn't cover the
	// variables it's helping pick between.
	colH, _ := m.sidebarSectionHeights()
	x = m.leftSidebarWidth() + 1
	y = colH + 4
	if y >= m.height {
		y = m.height - 1
	}
	return x, y
}

// urlRowView renders the method/url/env/send row as four separately-
// bordered boxes instead of one shared border - focusMethod and focusURL
// used to look almost identical (only a background fill inside one shared
// box told them apart), which a user reported as "unclear which panel/field
// is focused." Env and Send are both real Tab stops too (ctrl+e/ctrl+r and
// a mouse click already reach them from anywhere, but a keyboard-only user
// tabbing through the row needs to as well), each with its own focus
// border like method/url.
func (m Model) urlRowView(width int) string {
	// Keep the url input's visible width in sync with the box it's rendered in
	// (the center column, not the full terminal) - otherwise the input renders
	// wider than its narrower box and wraps, making the url row taller than its
	// fixed urlRowHeight and pushing the whole frame past the terminal height.
	m.urlInput.Width = urlBoxOuterWidth(width) - 4
	focus := m.effectiveFocus()
	currentMethod := string(collection.Methods[m.methodIdx])

	methStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(methodColor(currentMethod)))
	methBorder := borderStyle
	if focus == focusMethod {
		methBorder = focusedBorder
		methStyle = methStyle.Background(lipgloss.Color(methodColor(currentMethod))).Foreground(lipgloss.Color(activeTheme.AccentFg))
	}
	// methBorder already carries Padding(0, 1), and lipgloss's own Width()
	// counts that padding as part of the budget it fits the content into -
	// padding the method name out to the box's full width double-counts
	// those 2 padding columns and pushes the text 2 columns past budget,
	// which word-wraps it onto a second line instead of clipping (the
	// "bleeding" box seen when a 4-letter method got centered oddly). Pad
	// to content-minus-padding instead so the text plus the box's own
	// padding lands exactly on methodBoxContentWidth, never over it.
	methText := fmt.Sprintf("%-*s", methodBoxContentWidth-2, currentMethod)
	methBox := methBorder.Width(methodBoxContentWidth).Render(methStyle.Render(methText))

	urlBorder := borderStyle
	if focus == focusURL {
		urlBorder = focusedBorder
	}
	urlBoxW := urlBoxOuterWidth(width)
	urlBox := urlBorder.Width(urlBoxW - 2).Render(m.urlInput.View())

	// The primary environment switcher is the top-bar pill now (see
	// envDropdownAnchor) - this box is a read-only echo in the pill's own
	// color, so a user building a request sees at a glance which
	// environment it'll actually run against without looking away to the
	// top bar. Still a real focus/click target (ctrl+e, Tab, or clicking it
	// all still open the dropdown), just no longer the dropdown's anchor
	// point. -4 (not -2) for the text budget: this box, like method's,
	// carries Padding(0, 1) on top of its border, and lipgloss's Width()
	// counts that padding as part of the budget it fits content into - see
	// methBox's own comment above for the wrap bug that skipping this step
	// causes.
	sendBox := sendBoxView(focus == focusSend)

	// The environment lives in its own full-height panel now (see railView),
	// so the url row is just method, url, and send - no env box, no second
	// place to switch the environment from. At only 1 line of content tall,
	// sendBox needs vertical centering to sit level with the taller bordered
	// method/url boxes - lipgloss.Top would otherwise stick it to the row's
	// top border line.
	return lipgloss.JoinHorizontal(lipgloss.Center, methBox, " ", urlBox, " ", sendBox)
}

// sendBoxView renders the send button as a single-line filled pill, no
// border and no background fill, matching the reference screenshot's own
// action button ("[ stay ]") - a user first asked for the button "filled
// fully" (a seamless solid block), then reported that block as "too big";
// plain bracket text sidesteps both complaints; there's no fill to leave
// hollow, and no solid block to loom over its neighbors. Focus swaps in
// focusMarker (no border to swap shape on, and color alone doesn't survive
// stripANSI/a non-truecolor terminal). ctrl+r already sends from anywhere
// regardless of focus, so the label stays plain "Send" - repeating that
// shortcut inside the button itself would be redundant clutter.
func sendBoxView(focused bool) string {
	style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(activeTheme.Accent))
	if focused {
		style = style.Foreground(lipgloss.Color(activeTheme.FocusedBorder))
	}
	text := focusMarker(focused) + "[ Send ]"
	return lipgloss.NewStyle().Width(sendBoxWidth).Render(style.Render(text))
}

// layoutColumns computes the x-geometry of the two full-height columns: the
// left sidebar (Collections stacked over Environment - see leftsidebar.go) and
// the center column (url row over request/response). Shared by mainView
// (rendering) and hitTestZone (clicks) so the two can never disagree. leftW is
// 0 on a terminal too narrow for the sidebar; a single-column gap sits between
// the sidebar and the center when the sidebar is shown.
func (m Model) layoutColumns() (leftW, centerStart, centerW int) {
	leftW = m.leftSidebarWidth()
	gap := 0
	if leftW > 0 {
		gap = 1
	}
	centerStart = leftW + gap
	centerW = m.width - centerStart
	if centerW < 20 {
		centerW = 20
	}
	return
}

// mainView is the Request screen: the left sidebar (Collections over
// Environment, full height) beside the center column (url row over
// request/response). No top or bottom bar, and no right column anymore - the
// center runs to the right edge.
func (m Model) mainView() string {
	focus := m.effectiveFocus()
	_, _, centerW := m.layoutColumns()

	urlRow := m.urlRowView(centerW)
	workspace := m.workspaceView(centerW, panelContentHeight(m.height), focus)
	center := lipgloss.JoinVertical(lipgloss.Left, urlRow, workspace)

	if !m.leftSidebarShown() {
		return center
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, m.leftSidebarView(focus), " ", center)
}
