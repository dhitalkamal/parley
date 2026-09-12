package tui

// drawerOpen reports whether the collections drawer is currently shown -
// its own stored flag (drawerVisible), not derived from focus. An earlier
// version used focus == focusSidebar directly, on the theory that "open"
// and "sidebar focused" were the same fact by definition - but that meant
// Tab-ing out of the drawer's list into Request/Response necessarily
// closed it too, since focus leaving Sidebar was the only way the drawer
// ever closed. A user asked to browse Request/Response with the drawer
// still visible beside it, so visibility now outlives any single focus
// zone; New sets the initial state and only toggleDrawer changes it after.
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
