package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// collectionsStatusBar is this screen's own compact bottom bar - see
// keys.go/handlekey.go for why every screen gets its own instead of one
// shared global list.
const collectionsStatusBar = "up/down j/k navigate  enter open  n new  N folder  r rename  d delete  R run  / search  : palette  ? help  q quit"

// minWidthForCollectionsAboutPanel is the terminal width below which the
// About/Recent History panel is dropped and the tree goes back to full
// width - same graceful-degradation-on-narrow-terminals floor pattern as
// minWidthForShortcutsRail.
const minWidthForCollectionsAboutPanel = 80

func (m Model) collectionsAboutPanelVisible() bool {
	return m.width >= minWidthForCollectionsAboutPanel
}

// collectionsSidebarWidth splits the screen roughly evenly between the tree
// and the About panel, matching the design reference's near-50/50 layout.
func collectionsSidebarWidth(totalWidth int) int {
	return totalWidth / 2
}

// openCollectionsScreen switches to the full-screen Collections view,
// remembering whichever screen was active so its own back-key (see
// handleCollectionsScreenKey's RestEsc case) returns to it - same
// convention as openDashboard/openSettings. Day-to-day browsing normally
// goes through the drawer instead (see drawer.go); this is the escape
// hatch for when the drawer's narrower column isn't enough.
func (m Model) openCollectionsScreen() (Model, tea.Cmd) {
	if m.screen != ScreenCollections {
		m.previousScreen = m.screen
	}
	m.screen = ScreenCollections
	return m, nil
}

// collectionsScreenView renders the Collections screen: the collections
// tree, plus (on a wide enough terminal) a right-hand panel summarizing
// whichever request is currently selected and its recent send history -
// unlike the drawer overlay, which shares the frame with the Request screen
// behind it (see drawer_view.go), this is the screen itself, full width and
// full height.
func (m Model) collectionsScreenView() string {
	contentHeight := m.height - topBarHeight - tabStripHeight - helpBarHeight
	if contentHeight < 3 {
		contentHeight = 3
	}
	m.help.Width = m.width - 2
	status := labelStyle.Render(collectionsStatusBar)

	if !m.collectionsAboutPanelVisible() {
		m.sidebar.SetFocused(true)
		m.sidebar.SetSize(m.width-2, contentHeight-2)
		body := padLinesTo(titledBox(m.sidebar.View(), zoneHeaderText("Collections", true, true)), contentHeight)
		return padLinesTo(strings.Join([]string{body, status}, "\n"), m.height)
	}

	sidebarWidth := collectionsSidebarWidth(m.width)
	aboutWidth := m.width - sidebarWidth - 1

	m.sidebar.SetFocused(true)
	m.sidebar.SetSize(sidebarWidth-2, contentHeight-2)
	sidebarBox := titledBox(m.sidebar.View(), zoneHeaderText("Collections", true, true))
	aboutBox := m.aboutPanelView(aboutWidth, contentHeight)

	body := padLinesTo(lipgloss.JoinHorizontal(lipgloss.Top, sidebarBox, " ", aboutBox), contentHeight)
	return padLinesTo(strings.Join([]string{body, status}, "\n"), m.height)
}

// handleCollectionsScreenKey handles every key while the Collections screen
// is active: sidebar browsing/management (new/rename/delete/move/run/enter),
// quitting, and the same global overlays reachable from any screen (palette,
// help, environments, workspaces, dashboard, settings, import/export,
// history). Request-screen-only keys (send, save, tab-stop cycling, body/
// response shortcuts) don't apply here and are deliberately absent.
func (m Model) handleCollectionsScreenKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.sidebar.IsFiltering() {
		var cmd tea.Cmd
		m.sidebar, cmd = m.sidebar.Update(k)
		return m, cmd
	}

	switch {
	case k.String() == "q":
		m.confirm.Open(confirmQuit, "Quit parley?", "")
		return m, nil
	case key.Matches(k, keys.Quit):
		return m, tea.Quit
	case key.Matches(k, keys.RestEsc):
		// Collections is no longer the home screen (see screen.go/
		// openCollectionsScreen) - esc backs out to whichever screen was
		// active before it was opened, the same convention
		// handleDashboardKey/handleSettingsKey already use.
		m.screen = m.previousScreen
		return m, nil
	case key.Matches(k, keys.Palette):
		m.palette.Open()
		return m, nil
	case key.Matches(k, keys.Help):
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
	case key.Matches(k, keys.Environments):
		m.openEnvDropdown()
		return m, nil
	case key.Matches(k, keys.Workspaces):
		m.openWorkspaceSwitcher()
		return m, nil
	case key.Matches(k, keys.Dashboard):
		return m.openDashboard()
	case key.Matches(k, keys.Settings):
		return m.openSettings()
	case key.Matches(k, keys.Import):
		return m.startImport()
	case key.Matches(k, keys.Export):
		return m.startExport()
	case key.Matches(k, keys.History):
		return m.openHistory()
	}

	if newM, cmd, handled := m.handleSidebarKey(k); handled {
		return newM, cmd
	}

	// Mirrors dispatchKeyToFocusedWidget's focusSidebar case: a selection
	// change (not every keystroke - "/" to start filtering doesn't move the
	// cursor) previews the request under the cursor, the same file-explorer-
	// style preview the drawer already gives.
	before, hadBefore := m.sidebar.Selected()
	var cmd tea.Cmd
	m.sidebar, cmd = m.sidebar.Update(k)
	after, hadAfter := m.sidebar.Selected()
	if hadAfter && (!hadBefore || after.path != before.path) {
		m.sidebar.resetScroll()
		m = m.previewSidebarSelection()
	}
	return m, cmd
}
