package tui

import (
	workspace "github.com/dhitalkamal/parley/internal/workspace/domain"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// responseHintText nudges toward how to populate the (currently entirely
// absent - see workspaceView) response zone, instead of leaving a user to
// guess. ctrl+r is the keyboard path; Send is the mouse one.
const responseHintText = "No response yet - press ctrl+r or Send to run this request."

// zoneHeaderHeight is how many total rows a collapsed zone's header-only
// bar renders as in vertical orientation (border + one content row +
// border).
const zoneHeaderHeight = 3

// collapsedZoneColumnWidth is how wide a collapsed zone renders in
// horizontal orientation - just enough for its chevron and title, so the
// other zone reclaims as much width as possible. The vertical counterpart
// (collapsedZoneViewBar) shrinks height instead, since side-by-side zones
// collapse by width, not height.
const collapsedZoneColumnWidth = 20

// workspaceHorizontalFloor is the narrowest terminal width horizontal
// orientation (request and response side by side, each with the same
// 24-column floor the old 3-panel grid used) can render without either zone
// overflowing - the horizontal-orientation counterpart to the old 3-panel
// grid's narrowGridWidth fallback. Below it, workspaceView renders vertical
// instead, regardless of the user's saved orientation preference; the
// preference itself is untouched - it's just not usable stacked side by
// side at this width today.
const workspaceHorizontalFloor = 24 + 1 + 24

// zoneChevron is the expand/collapse indicator shown in a zone's header - a
// filled down-triangle when expanded, a filled right-triangle when collapsed
// (see glyphs.go: built from code points so the source stays ASCII while the
// UI renders solid triangles, matching the dropdown affordance).
func zoneChevron(expanded bool) string {
	if expanded {
		return glyphTriangleDown
	}
	return glyphTriangleRight
}

// zoneHeaderText renders a zone's header: chevron + title, accent-colored
// and bold when focused, dim otherwise - the same visual language
// panelTitleText used for the old numbered panel titles, minus the [N]
// prefix (there's no numbered panel-jump scheme to reference once
// Collections moved out of the permanent grid into a drawer - see
// drawer.go).
func zoneHeaderText(title string, expanded, focused bool) string {
	text := zoneChevron(expanded) + " " + title
	if focused {
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(activeTheme.FocusedBorder)).Render(text)
	}
	return labelStyle.Render(text)
}

// collapsedZoneViewBar renders a collapsed zone as a full-width, header-
// only bar - vertical orientation's collapse shape: the other zone stacks
// above/below it and reclaims the rows this one gives up.
func collapsedZoneViewBar(width int, title string, focused bool) string {
	style := borderStyle
	if focused {
		style = focusedBorder
	}
	return style.Width(width - 2).Render(zoneHeaderText(title, false, focused))
}

// collapsedZoneViewColumn renders a collapsed zone as a narrow, full-height
// column - horizontal orientation's collapse shape: the other zone sits
// beside it and reclaims the columns this one gives up.
func collapsedZoneViewColumn(height int, title string, focused bool) string {
	style := borderStyle
	if focused {
		style = focusedBorder
	}
	return style.Width(collapsedZoneColumnWidth - 2).Height(height - 2).Render(zoneHeaderText(title, false, focused))
}

// requestZoneBox renders the (expanded) Request zone's box, with the body-
// type/content-type dropdown labels spliced into the right side of its
// header when the Body tab is active (see bodyHeaderRightText) - shared by
// both the request-only and the two-zone rendering paths below.
func (m Model) requestZoneBox(width, height int, focused bool) string {
	return titledBoxWithRight(
		m.requestPanelView(width, height),
		zoneHeaderText("Request", true, focused),
		m.bodyHeaderRightText(),
	)
}

// workspaceView renders the request/response workspace from computeWorkspaceGeom's
// layout: no response zone at all until one exists, otherwise the two zones
// stacked per orientation, each independently collapsible to a header-only
// bar/column.
func (m Model) workspaceView(width, height int, focus int) string {
	g := m.computeWorkspaceGeom(width, height)

	if !g.responseVisible {
		// The request zone's own collapsed state doesn't apply here - with
		// no response zone to hand the freed space to, "collapsing" the
		// only zone that exists would leave nothing on screen at all.
		hint := labelStyle.Render(responseHintText)
		box := m.requestZoneBox(g.request.width, g.request.height, focus == focusRequest)
		return lipgloss.JoinVertical(lipgloss.Left, box, hint)
	}

	if g.requestHidden {
		// Zoomed: the Request zone (and its own collapsed/expanded state)
		// doesn't render at all - Response gets the entire content frame,
		// the same way a terminal multiplexer's pane zoom works.
		return titledBoxWithRight(
			m.responsePanelView(g.response.width, g.response.height),
			zoneHeaderText("Response", true, true),
			labelStyle.Render("ctrl+z restore"),
		)
	}

	reqFocused := focus == focusRequest
	respFocused := focus == focusResponse

	var reqBox, respBox string
	if m.requestCollapsed {
		if g.orientation == workspace.OrientationHorizontal {
			reqBox = collapsedZoneViewColumn(g.request.height, "Request", reqFocused)
		} else {
			reqBox = collapsedZoneViewBar(g.request.width, "Request", reqFocused)
		}
	} else {
		reqBox = m.requestZoneBox(g.request.width, g.request.height, reqFocused)
	}
	if m.responseCollapsed {
		if g.orientation == workspace.OrientationHorizontal {
			respBox = collapsedZoneViewColumn(g.response.height, "Response", respFocused)
		} else {
			respBox = collapsedZoneViewBar(g.response.width, "Response", respFocused)
		}
	} else {
		// Status/timing/size ride the title's right side (see responseStatusMeta);
		// titledBoxWithRight drops them if the panel's too narrow to fit both.
		respBox = titledBoxWithRight(
			m.responsePanelView(g.response.width, g.response.height),
			zoneHeaderText("Response", true, respFocused),
			m.responseStatusMeta(),
		)
	}

	if g.orientation == workspace.OrientationHorizontal {
		return lipgloss.JoinHorizontal(lipgloss.Top, reqBox, " ", respBox)
	}

	result := lipgloss.JoinVertical(lipgloss.Left, reqBox, respBox)
	if m.requestCollapsed && m.responseCollapsed {
		// Both collapsed leaves rows below the two header bars that
		// neither zone wants to expand into (there's nothing to expand FOR
		// when both are deliberately minimized) - pad with blank lines so
		// the total still fills height exactly. Rendering short here
		// leaves the whole frame one or more rows short of the terminal,
		// which is the exact bug TestModelView_RendersExactlyTheTerminalHeight
		// guards against: bubbletea's alt-screen renderer then scrolls the
		// top bar off-screen, since nothing above it absorbs the shortfall.
		if used := lipgloss.Height(result); used < height {
			result += strings.Repeat("\n", height-used)
		}
	}
	return result
}
