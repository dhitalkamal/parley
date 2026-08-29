package tui

// drawerWidth is the Collections section's width - the same fixed left-sidebar
// width the Environment section uses, since they share one column now (see
// leftsidebar.go).
func (m Model) drawerWidth() int {
	return m.leftSidebarWidth()
}

// drawerTop is the row the Collections section starts on - the very top of the
// left column (see leftsidebar.go). Used by mouse.go to map a click row to a
// tree item.
func drawerTop() int {
	return topBarHeight + tabStripHeight
}

// workspaceXOffset is how many columns the request/response workspace is
// shifted right to make room for the collections drawer beside it (its own
// width, plus a 1-column gap) - zero while the drawer is closed. An earlier
// version rendered the drawer as an overlay floating on top of a full-width,
// never-resized workspace instead; a user found that still counted as
// hiding the request/response zones (just via compositing instead of a
// scrim) and asked for a real side-by-side layout, like a permanent sidebar
// panel while the drawer happens to be open - the same shape the old grid
// column used, minus always being there. Every click/hit-test/anchor
// computation that needs to know where the request/response zones actually
// are goes through this (see workspace_geom.go's currentWorkspaceGeom), so
// they can never drift out of sync with what mainView() actually rendered.
func (m Model) workspaceXOffset() int {
	_, centerStart, _ := m.layoutColumns()
	return centerStart
}
