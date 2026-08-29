package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// requestPanelView renders the Request panel's full bordered box: an
// unresolved-variables warning (if any), the tab bar, and whichever single
// tab (Params/Headers/Body/Auth/Scripts/Settings) is active - see
// reqtab.go. width
// and height are the panel's OUTER dimensions (border included); the active
// sub-widget is sized to exactly fill what's left after the warn/tab-bar
// rows, this panel's own border, and (for every tab except Params/Headers)
// that sub-widget's own nested border and label line.
func (m Model) requestPanelView(width, height int) string {
	// innerWidth is this panel's own content area; kvTable/bodyEditor/
	// scriptEditor each then nest their OWN border+padding on top of
	// whatever width they're given, so they need innerWidth minus another
	// 4 columns, or they render wider than this panel allotted them -
	// which does not get clipped, it gets wrapped, which silently doubles
	// the rendered height of every over-width line.
	innerWidth := width - 4
	nestedContentWidth := innerWidth - 4
	// params/headers are borderless now (see kvTable.borderless), so they get
	// the full innerWidth rather than nestedContentWidth - the same as the
	// Body None/Raw and Auth/Settings tabs, and matching the Response body.
	m.params.SetWidth(innerWidth)
	m.headers.SetWidth(innerWidth)

	var sections []string
	warnLines := 0
	if warn := m.unresolvedWarning(); warn != "" {
		sections = append(sections, warn)
		warnLines = 1
	}
	tabBar := reqTabBarText(m.reqTab, len(m.params.Rows()), len(m.headers.Rows()), m.body.HasBody(), innerWidth)
	sections = append(sections, tabBar)

	// Almost every tab now renders flush against this panel's own border - no
	// second box-in-a-box. bodyEditor's None/Raw did already; params/headers
	// (kvTable.borderless), Scripts (scriptEditor.View drops its border), and
	// Auth/Settings joined them so the whole Request panel matches the
	// Response body's flush treatment - each sets nestedBorder = 0 below. The
	// default stays 2 for the only holdouts: Body's Form/Multipart/GraphQL/
	// Binary sub-types still reuse a widget that draws its own border.
	// Scripts prefixes a 1-line label above its textarea; Body's own header
	// (type/content-type hint) varies in height, so it's measured via
	// headerLines() rather than assumed; Params/Headers add their own
	// variable number of lines (the empty-state hint, or the key/value edit
	// form) via kvTable.ExtraLines() - getting any of this wrong doesn't
	// wrap, it silently clips those lines off the bottom of the panel, which
	// is exactly what made "how do I add a header" so confusing: the edit
	// form was rendering, just never visible.
	nestedBorder := 2
	labelLine := 0
	switch m.reqTab {
	case reqTabBody:
		labelLine = m.body.headerLines() + m.body.ExtraLines()
		// None/Raw render flush against this panel's own border (see
		// bodyEditor.rendersBorderless); every other body type reuses a
		// widget that draws its own nested border, same as Params/Headers/
		// Pre-request/Tests below, so it keeps the default nestedBorder.
		if m.body.rendersBorderless() {
			nestedBorder = 0
		}
	case reqTabScripts:
		// scriptEditor renders its textarea flush now (just a label line above
		// it, no nested border) - same as the Body Raw editor.
		labelLine = 1
		nestedBorder = 0
	case reqTabParams:
		labelLine = m.params.ExtraLines()
		nestedBorder = 0
	case reqTabHeaders:
		labelLine = m.headers.ExtraLines()
		nestedBorder = 0
	case reqTabAuth, reqTabSettings:
		// authEditor/requestSettingsEditor's View() has no border of its own
		// (plain label:field rows) - same reasoning as Body's None/Raw case,
		// it renders flush against this panel's own border instead of
		// wasting a budget row pair on a border that's never actually drawn.
		nestedBorder = 0
	}
	// tabBar is now 2 rows (tab labels + underline, see tabbar.go) instead
	// of 1 - measuring it rather than assuming len(sections) == row count
	// avoids silently shrinking every tab's own height budget by 1 row too
	// few, which clips content off the bottom rather than visibly wrapping.
	budget := height - 2 /* this panel's own border */ - warnLines - lipgloss.Height(tabBar) - nestedBorder - labelLine
	if budget < 3 {
		budget = 3
	}

	switch m.reqTab {
	case reqTabParams:
		m.params.SetHeight(budget)
		sections = append(sections, m.params.View())
	case reqTabHeaders:
		m.headers.SetHeight(budget)
		sections = append(sections, m.headers.View())
	case reqTabBody:
		if m.body.rendersBorderless() {
			m.body.SetSize(innerWidth, budget)
		} else {
			m.body.SetSize(nestedContentWidth, budget)
		}
		sections = append(sections, m.body.View())
	case reqTabScripts:
		m.scripts.SetSize(innerWidth, budget)
		sections = append(sections, m.scripts.View())
	case reqTabAuth:
		m.auth.SetSize(innerWidth, budget)
		sections = append(sections, m.auth.View())
	case reqTabSettings:
		m.reqSettings.SetSize(innerWidth, budget)
		sections = append(sections, m.reqSettings.View())
	}

	content := strings.Join(sections, "\n")
	style := borderStyle
	if m.effectiveFocus() == focusRequest {
		style = focusedBorder
	}
	// Width() sizes the padding+content area together (border is the only
	// thing it adds on top of), confirmed empirically - -2 not -4.
	return style.Width(width - 2).Height(height - 2).Render(content)
}
