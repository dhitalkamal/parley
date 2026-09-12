package tui

import (
	workspace "github.com/dhitalkamal/parley/internal/workspace/domain"

	"github.com/charmbracelet/lipgloss"
)

// zoneRect is a zone's position and size within the workspace region (which
// itself starts at gridTop() on screen - these coordinates are workspace-
// relative, not absolute screen coordinates).
type zoneRect struct {
	x, y, width, height int
}

// contains reports whether the workspace-relative point (x, y) falls
// within r.
func (r zoneRect) contains(x, y int) bool {
	return x >= r.x && x < r.x+r.width && y >= r.y && y < r.y+r.height
}

// workspaceGeom is the request/response workspace's current arrangement -
// computed once and shared by workspace_view.go (rendering) and mouse.go
// (click hit-testing) so the two can never disagree about where a zone
// actually is, the same reasoning the old 3-panel grid's panelWidths
// followed.
type workspaceGeom struct {
	responseVisible bool
	// requestHidden is true only while the response is maximized (see
	// Model.responseMaximized) - the zero value (false, request visible)
	// matches every other construction site below, none of which need to
	// set it explicitly.
	requestHidden bool
	orientation   workspace.Orientation // the effective one, after the narrow-terminal fallback
	request       zoneRect
	response      zoneRect // zero value when !responseVisible
}

// computeWorkspaceGeom lays out the request/response zones for a
// width x height workspace region, given the model's current orientation
// and collapse state.
func (m Model) computeWorkspaceGeom(width, height int) workspaceGeom {
	if !m.responseZoneVisible() {
		hint := labelStyle.Render(responseHintText)
		reqHeight := height - lipgloss.Height(hint)
		if reqHeight < 5 {
			reqHeight = 5
		}
		return workspaceGeom{request: zoneRect{0, 0, width, reqHeight}}
	}
	if m.responseMaximized {
		return workspaceGeom{
			responseVisible: true,
			requestHidden:   true,
			orientation:     m.orientation,
			response:        zoneRect{0, 0, width, height},
		}
	}

	orientation := m.orientation
	if orientation == workspace.OrientationHorizontal && width < workspaceHorizontalFloor {
		orientation = workspace.OrientationVertical
	}
	if orientation == workspace.OrientationHorizontal {
		return m.horizontalGeom(width, height)
	}
	return m.verticalGeom(width, height)
}

// currentWorkspaceGeom is computeWorkspaceGeom for the workspace's actual
// current width and screen x-origin, accounting for the collections drawer
// (see layoutColumns) occupying real columns on the left while open. Every
// click/hit-test/anchor computation that needs to know where the
// request/response zones really are calls this instead of
// computeWorkspaceGeom(m.width, ...) directly, so a stale full-width
// assumption can't creep back into one call site while the rest account
// for the drawer.
func (m Model) currentWorkspaceGeom() (g workspaceGeom, xOffset int) {
	// The request/response zones live in the center column now (Explorer left,
	// Environment right - see layoutColumns), so xOffset is that column's left
	// edge and the width is the center column's width. gridTop() (the url row's
	// height) is still their y-origin, since the url row sits above them inside
	// the same center column.
	_, centerStart, centerW := m.layoutColumns()
	g = m.computeWorkspaceGeom(centerW, panelContentHeight(m.height))
	return g, centerStart
}

func (m Model) verticalGeom(width, height int) workspaceGeom {
	g := workspaceGeom{responseVisible: true, orientation: workspace.OrientationVertical}
	switch {
	case m.requestCollapsed && m.responseCollapsed:
		g.request = zoneRect{0, 0, width, zoneHeaderHeight}
		g.response = zoneRect{0, zoneHeaderHeight, width, zoneHeaderHeight}
	case m.requestCollapsed:
		g.request = zoneRect{0, 0, width, zoneHeaderHeight}
		g.response = zoneRect{0, zoneHeaderHeight, width, height - zoneHeaderHeight}
	case m.responseCollapsed:
		g.request = zoneRect{0, 0, width, height - zoneHeaderHeight}
		g.response = zoneRect{0, height - zoneHeaderHeight, width, zoneHeaderHeight}
	default:
		reqHeight := height / 2
		g.request = zoneRect{0, 0, width, reqHeight}
		g.response = zoneRect{0, reqHeight, width, height - reqHeight}
	}
	return g
}

func (m Model) horizontalGeom(width, height int) workspaceGeom {
	g := workspaceGeom{responseVisible: true, orientation: workspace.OrientationHorizontal}
	switch {
	case m.requestCollapsed && m.responseCollapsed:
		g.request = zoneRect{0, 0, collapsedZoneColumnWidth, height}
		g.response = zoneRect{collapsedZoneColumnWidth + 1, 0, width - collapsedZoneColumnWidth - 1, height}
	case m.requestCollapsed:
		g.request = zoneRect{0, 0, collapsedZoneColumnWidth, height}
		g.response = zoneRect{collapsedZoneColumnWidth + 1, 0, width - collapsedZoneColumnWidth - 1, height}
	case m.responseCollapsed:
		g.request = zoneRect{0, 0, width - collapsedZoneColumnWidth - 1, height}
		g.response = zoneRect{width - collapsedZoneColumnWidth, 0, collapsedZoneColumnWidth, height}
	default:
		reqWidth := width / 2
		g.request = zoneRect{0, 0, reqWidth, height}
		g.response = zoneRect{reqWidth + 1, 0, width - reqWidth - 1, height}
	}
	return g
}
