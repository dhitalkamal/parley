package tui

// drawerTop is the row the Collections section starts on - the very top of the
// left column (see leftsidebar.go). Used by mouse.go to map a click row to a
// tree item.
func drawerTop() int {
	return topBarHeight + tabStripHeight
}
