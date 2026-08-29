package tui

// drawerOpen reports whether the collections drawer is currently shown -
// its own stored flag (drawerVisible), not derived from focus. An earlier
// version used focus == focusSidebar directly, on the theory that "open"
// and "sidebar focused" were the same fact by definition - but that meant
// Tab-ing out of the drawer's list into Request/Response necessarily
// closed it too, since focus leaving Sidebar was the only way the drawer
// ever closed. A user asked to browse Request/Response with the drawer
// still visible beside it (see workspaceXOffset), so visibility now
// outlives any single focus zone; only openDrawer/closeDrawer/toggleDrawer
// change it.
func (m Model) drawerOpen() bool {
	return m.drawerVisible
}

// collapseCollectionsIfFocused releases focus off the Collections section when
// it's being collapsed, so focus never lands on a section that's shrunk to a
// header bar.
func (m *Model) releaseFocusFrom(zone int) {
	if m.focus != zone {
		return
	}
	m.focus = focusRequest
	m.updateFocus()
}

// openDrawer shows the drawer and moves focus into it, remembering the
// current focus so closeDrawer can restore it instead of always landing
// back on Request regardless of where the user actually was.
func (m *Model) openDrawer() {
	if m.drawerVisible {
		return
	}
	m.preDrawerFocus = m.focus
	m.drawerVisible = true
	m.focus = focusSidebar
	m.updateFocus()
}

// closeDrawer hides the drawer. Focus is only restored to preDrawerFocus
// when it's still on the drawer's own list - if the user already tabbed or
// clicked away to Request/Response while the drawer stayed open beside it,
// hiding the drawer now shouldn't also yank focus somewhere else. A no-op
// while the drawer isn't shown.
func (m *Model) closeDrawer() {
	if !m.drawerVisible {
		return
	}
	m.drawerVisible = false
	if m.focus == focusSidebar {
		m.focus = m.preDrawerFocus
		m.updateFocus()
	}
}

// toggleDrawer is ctrl+\'s binding: collapse the Collections section to a header
// bar if it's expanded, expand it if collapsed. It no longer hides the whole
// column (Environment still lives below it) - collapsing just hands Collections'
// rows to the Environment section. Collapsing while Collections is focused
// releases focus so it doesn't stick on a header bar.
func (m *Model) toggleDrawer() {
	m.drawerVisible = !m.drawerVisible
	if !m.drawerVisible {
		m.releaseFocusFrom(focusSidebar)
	}
	m.persistLayout()
}
