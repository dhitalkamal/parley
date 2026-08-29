package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// The left sidebar stacks two collapsible sections in one column: Collections
// (top) and Environment (bottom). Each section's expanded/collapsed state
// reuses an existing persisted flag - drawerVisible for Collections,
// shortcutsRailHidden (inverted) for Environment - so the collapse choice still
// survives a restart. A collapsed section shrinks to a single header bar and
// gives its rows to the other section; the whole sidebar drops out below
// minWidthForSidebar so a narrow terminal keeps a usable center.

const (
	minWidthForSidebar = 60
	// leftSidebarOuterWidth reuses the env panel's fixed width so its many
	// width-derived constants (shortcutsRailContentWidth etc.) stay valid
	// unchanged; Collections just renders at the same width.
	leftSidebarOuterWidth = shortcutsRailOuterWidth
	collapsedSectionH     = 1
)

func (m Model) leftSidebarShown() bool { return m.width >= minWidthForSidebar }

func (m Model) leftSidebarWidth() int {
	if !m.leftSidebarShown() {
		return 0
	}
	return leftSidebarOuterWidth
}

// collectionsExpanded / environmentExpanded reinterpret the old drawer/rail
// visibility flags as this section's collapse state.
func (m Model) collectionsExpanded() bool { return m.drawerVisible }
func (m Model) environmentExpanded() bool { return !m.shortcutsRailHidden }

// sidebarSectionHeights splits the full column height between Collections and
// Environment. Both open -> Collections gets ~55%; one collapsed -> it's a
// single header row and the other fills the rest.
func (m Model) sidebarSectionHeights() (colH, envH int) {
	total := m.height
	ce, ee := m.collectionsExpanded(), m.environmentExpanded()
	switch {
	case ce && ee:
		colH = total * 11 / 20
		if colH < 6 {
			colH = 6
		}
		envH = total - colH
		if envH < 8 {
			envH = 8
			colH = total - envH
		}
	case ce && !ee:
		envH, colH = collapsedSectionH, total-collapsedSectionH
	case !ce && ee:
		colH, envH = collapsedSectionH, total-collapsedSectionH
	default:
		colH, envH = collapsedSectionH, collapsedSectionH
	}
	return
}

// leftSidebarView composes the two stacked sections into the full-height left
// column.
func (m Model) leftSidebarView(focus int) string {
	w := m.leftSidebarWidth()
	colH, envH := m.sidebarSectionHeights()
	top := m.collectionsSectionView(w, colH, focus == focusSidebar)
	bottom := m.environmentSectionView(w, envH, focus == focusRail)
	return lipgloss.JoinVertical(lipgloss.Left, top, bottom)
}

// collectionsSectionView renders Collections: the sidebar tree in a titled box
// when expanded, or a one-line header bar when collapsed.
func (m Model) collectionsSectionView(w, h int, focused bool) string {
	if !m.collectionsExpanded() {
		return collapsedSectionBar(w, "Explorer", focused)
	}
	m.sidebar.SetSize(w-4, h-2)
	return titledBox(m.sidebar.View(), sectionTitle("Explorer", focused))
}

// environmentSectionView renders Environment via the existing rail view (which
// also honors the env/shortcuts sub-mode), or a header bar when collapsed.
func (m Model) environmentSectionView(w, h int, focused bool) string {
	if !m.environmentExpanded() {
		return collapsedSectionBar(w, "Environment", focused)
	}
	return m.railView(h)
}

// collapsedSectionBar is the single-row header shown for a collapsed section: a
// right-pointing filled triangle and the (upper-cased) title, accented when
// focused.
func collapsedSectionBar(w int, title string, focused bool) string {
	style := labelStyle
	if focused {
		style = activeTabStyle.Bold(true)
	}
	text := glyphTriangleRight + " " + strings.ToUpper(title)
	if lipgloss.Width(text) > w {
		text = ansi.Cut(text, 0, w)
	}
	return lipgloss.NewStyle().Width(w).Render(style.Render(text))
}

// sectionTitle is an expanded section's box title. zoneHeaderText already
// prepends the down-pointing filled triangle for an expanded zone, so this just
// forwards to it - an earlier version added a second triangle, the double-
// chevron bug the user reported.
func sectionTitle(title string, focused bool) string {
	return zoneHeaderText(title, true, focused)
}
