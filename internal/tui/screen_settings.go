package tui

import (
	"fmt"
	"strings"

	workspace "github.com/dhitalkamal/parley/internal/workspace/domain"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

const settingsStatusBar = "t theme  o orientation  esc back  ? help"

// openSettings switches to the Settings screen, remembering whichever
// screen was active so its own back-key returns to it - same convention as
// openDashboard.
func (m Model) openSettings() (Model, tea.Cmd) {
	if m.screen != ScreenSettings {
		m.previousScreen = m.screen
	}
	m.screen = ScreenSettings
	return m, nil
}

// handleSettingsKey handles the Settings screen's small, fixed set of
// preferences - deliberately minimal for now (theme, layout orientation):
// see the Phase 1 scope note in dashboard.go's sibling screens for why
// deeper settings (keyboard config, persistence) are deferred.
func (m Model) handleSettingsKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(k, keys.Quit):
		return m, tea.Quit
	case k.String() == "esc" || key.Matches(k, keys.Settings):
		m.screen = m.previousScreen
		return m, nil
	case k.String() == "t":
		m.status = "Theme: " + cycleTheme()
		m.sidebar.RefreshDelegate()
		m.palette.RefreshDelegate()
		return m, nil
	case k.String() == "o" || key.Matches(k, keys.ToggleOrientation):
		if m.orientation == workspace.OrientationVertical {
			m.orientation = workspace.OrientationHorizontal
		} else {
			m.orientation = workspace.OrientationVertical
		}
		m.persistLayout()
		return m, nil
	}
	return m, nil
}

func settingsHeader() string {
	return activeTabStyle.Render("Settings") + labelStyle.Render("  (esc close)")
}

// settingsScreenView renders the Settings screen as a full frame, matching
// every other screen's header+content+status-bar shape.
func (m Model) settingsScreenView() string {
	orientationLabel := "Vertical"
	if m.orientation == workspace.OrientationHorizontal {
		orientationLabel = "Horizontal"
	}

	var b strings.Builder
	fmt.Fprintln(&b, settingsHeader())
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, labelStyle.Render("Appearance"))
	fmt.Fprintf(&b, "  Theme                %s\n", activeTheme.Name)
	fmt.Fprintf(&b, "  Layout               %s\n", orientationLabel)

	content := borderStyle.Render(strings.TrimRight(b.String(), "\n"))
	status := labelStyle.Render(settingsStatusBar)
	return padLinesTo(strings.Join([]string{content, status}, "\n"), m.height)
}
