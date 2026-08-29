package tui

// updateFocus blurs every focusable widget, then focuses whichever one
// m.focus (and, inside the Request panel, m.reqTab) now points at - called
// after every focus/tab change so exactly one widget is ever focused.
func (m *Model) updateFocus() {
	m.urlInput.Blur()
	m.params.SetFocus(false)
	m.headers.SetFocus(false)
	m.body.SetFocus(false)
	m.scripts.SetFocus(false)
	m.auth.SetFocus(false)
	m.reqSettings.SetFocus(false)
	m.sidebar.SetFocused(m.focus == focusSidebar)
	// Focusing the rail's env view reloads it from the active environment, so
	// starting to edit always sees current values even if they changed via
	// the ctrl+e panel since the rail was last looked at. refreshRailEnv is a
	// no-op mid-edit, so this can't clobber a row being typed.
	if m.focus == focusRail && m.railEnvMode() {
		m.refreshRailEnv()
	}
	switch m.focus {
	case focusURL:
		m.urlInput.Focus()
	case focusRequest:
		switch m.reqTab {
		case reqTabParams:
			m.params.SetFocus(true)
		case reqTabHeaders:
			m.headers.SetFocus(true)
		case reqTabBody:
			m.body.SetFocus(true)
		case reqTabScripts:
			// kind isn't forced here (unlike the old two-tab split, which
			// each hardcoded which script it meant) - ctrl+b inside
			// scriptEditor already owns which of pre-request/test is
			// showing, so switching back to this tab just resumes wherever
			// that toggle last left it.
			m.scripts.SetFocus(true)
		case reqTabAuth:
			m.auth.SetFocus(true)
		case reqTabSettings:
			m.reqSettings.SetFocus(true)
		}
	}
}

// tabStops is the ordered list of focus zones Tab/shift+tab currently
// cycle through: the fixed six (Method, URL, Env, Send, Request, Response),
// matching the screen's own left-to-right/top-to-bottom order, plus
// Sidebar spliced in right after Send whenever the drawer is visible -
// there's nothing to tab into there otherwise. Visibility no longer
// implies focus (see drawer.go's drawerOpen doc comment), so tabbing
// through Sidebar never closes the drawer, and tabbing away from it never
// opens one that wasn't already open.
func (m Model) tabStops() []int {
	// Left column first, top to bottom: Collections then Environment (only the
	// expanded ones - a collapsed section is a header bar with nothing to focus,
	// reached via its toggle key instead). Then the center: method -> url ->
	// send -> request -> response.
	stops := []int{}
	if m.leftSidebarShown() {
		if m.collectionsExpanded() {
			stops = append(stops, focusSidebar)
		}
		if m.environmentExpanded() {
			stops = append(stops, focusRail)
		}
	}
	stops = append(stops, focusMethod, focusURL, focusSend)
	// Request doesn't even render while the Response is maximized (unlike a
	// merely collapsed zone, which is still an on-screen header bar worth
	// tabbing to) - nothing there to focus.
	if m.responseMaximized {
		stops = append(stops, focusResponse)
	} else {
		stops = append(stops, focusRequest, focusResponse)
	}
	return stops
}

// advanceFocus moves current to the next (dir=1) or previous (dir=-1) stop
// in stops, wrapping around. Falls back to the first stop if current isn't
// actually one of them - shouldn't happen, but a safe default beats an
// out-of-range index.
func advanceFocus(stops []int, current, dir int) int {
	idx := 0
	for i, s := range stops {
		if s == current {
			idx = i
			break
		}
	}
	n := len(stops)
	return stops[(idx+dir+n)%n]
}

// currentlyEditingRow reports whether the Request panel is mid-edit of a
// params/headers row - Tab must stay swallowed by that inline form rather
// than changing focus out from under it. Body/script textareas have no such
// lock: they already consume every key except the ones root.go intercepts
// first (ctrl+b/ctrl+g for body, ctrl+b for scripts).
func (m Model) currentlyEditingRow() bool {
	if m.focus != focusRequest {
		return false
	}
	switch m.reqTab {
	case reqTabParams:
		return m.params.editing
	case reqTabHeaders:
		return m.headers.editing
	}
	return false
}

// modalActive reports whether anything is currently drawn as a floating
// overlay and owns all keyboard input (see handleKey's early-return chain) -
// used to suppress the background's own focus border so it doesn't keep
// glowing "focused" behind a modal that actually has the keyboard.
func (m Model) modalActive() bool {
	return m.confirm.active || m.prompt.active || m.kvAdd.active || m.palette.active ||
		m.history.active || m.runner.active || m.envPanel.active ||
		m.envDropdown.active || m.workspace.active ||
		m.showVariables || m.showCodeSnippet ||
		m.bodyTypeDropdown.active || m.contentTypeDropdown.active
}

// effectiveFocus is what rendering code should check instead of m.focus
// directly - the real focus zone normally, or focusNone while a modal is
// open, so the 3-panel grid (and the method/url boxes) never show a stale
// "this is focused" border for a panel that isn't actually receiving input
// anymore.
func (m Model) effectiveFocus() int {
	if m.modalActive() {
		return focusNone
	}
	return m.focus
}

// textEntryFocused reports whether the keystroke about to be handled would
// otherwise land in a live multi-line text-editing widget - a body/script
// textarea, or a params/headers row mid-edit. bubbles/textarea binds
// several of this app's global shortcuts as its own emacs-style editing
// keys by default (ctrl+e line-end, ctrl+w delete-word, ctrl+u/ctrl+k
// delete-before/after-cursor, ctrl+t transpose) - firing the global command
// instead of the widget's own action broke ordinary line/word editing every
// time someone reached for that muscle memory while composing a body or
// script. The URL bar is deliberately excluded: it's a short single-line
// field where quick access to these global shortcuts already has an
// established, tested contract (e.g. ctrl+e opening the environment
// dropdown from the app's default startup focus) and word-jump editing
// inside a URL is rare enough not to be worth breaking that.
func (m Model) textEntryFocused() bool {
	// On the WebSocket screen a URL/composer field (or a header row mid-edit)
	// owns typing - otherwise a bare "1" would switch primary tabs instead of
	// going into the ws:// URL. The transcript isn't a text field, so digits
	// there still navigate (an exit path off the screen).
	if m.screen == ScreenWebSocket {
		f := m.ws.active().focus
		if f == wsFocusURL || f == wsFocusComposer {
			return true
		}
		return f == wsFocusHeaders && m.ws.active().headers.editing
	}
	// The rail's env editor owns every key while a row is being edited, same
	// as the request panel's editors below - so bare digits (tab-nav) and
	// single-key globals stay out of the key/value fields.
	if m.focus == focusRail && m.railEnv.editing {
		return true
	}
	// URL/Body/Scripts are modal now (see mode.go): keys only flow into them in
	// INSERT. In NORMAL they're navigable and single-letter commands apply.
	if m.insertableFocus() {
		return m.typingNow()
	}
	if m.focus != focusRequest {
		return false
	}
	// Auth fields and open params/headers row editors aren't modal yet - they
	// still consume keys whenever focused/editing.
	switch m.reqTab {
	case reqTabAuth:
		return true
	}
	return m.currentlyEditingRow()
}
