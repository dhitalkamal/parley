package tui

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	workspace "github.com/dhitalkamal/parley/internal/workspace/domain"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// handleKey is root.go's tea.KeyMsg entry point - split out on its own
// since it's the single largest routine in the app (every modal's early-
// return, then every global keybinding, then per-focus-zone handling)
// and was pushing root.go itself past this project's file-size guidance.
func (m Model) handleKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.confirm.active {
		return m.handleConfirmKey(k)
	}
	if m.prompt.active {
		return m.handlePromptKey(k)
	}
	if m.kvAdd.active {
		return m.handleKVAddKey(k)
	}
	if m.palette.active {
		return m.handlePaletteKey(k)
	}
	if m.history.active {
		return m.handleHistoryKey(k)
	}
	if m.runner.active {
		return m.handleRunnerKey(k)
	}
	if m.envPanel.active {
		return m.handleEnvPanelKey(k)
	}
	if m.envDropdown.active {
		return m.handleEnvDropdownKey(k)
	}
	if m.bodyTypeDropdown.active {
		return m.handleBodyTypeDropdownKey(k)
	}
	if m.contentTypeDropdown.active {
		return m.handleContentTypeDropdownKey(k)
	}
	if m.workspace.active {
		return m.handleWorkspaceKey(k)
	}
	// Pane jump: a bare 1-5 (when nothing is being typed into) focuses a region
	// on the Request screen - 1 Collections, 2 Environment, 3 URL, 4 Request,
	// 5 Response (the vim/lazygit digit-to-region convention). Screens are
	// reached via the palette / their function keys instead.
	if nm, cmd, handled := m.handlePaneDigit(k); handled {
		return nm, cmd
	}
	// Dashboard/Settings/Collections are full screens, not overlays, but get
	// the same early-return treatment: none of the Request screen's tab-stop
	// cycling or grid-specific keys apply to them - see screen.go.
	if m.screen == ScreenDashboard {
		return m.handleDashboardKey(k)
	}
	if m.screen == ScreenSettings {
		return m.handleSettingsKey(k)
	}
	if m.screen == ScreenCollections {
		return m.handleCollectionsScreenKey(k)
	}
	if m.screen == ScreenWebSocket {
		return m.handleWSKey(k)
	}
	if m.help.ShowAll {
		// The full-help overlay swallows keys while open; "?" or esc closes it.
		if k.String() == "esc" || key.Matches(k, keys.Help) {
			m.help.ShowAll = false
		}
		return m, nil
	}
	if m.showVariables {
		if k.String() == "esc" || key.Matches(k, keys.ToggleVars) {
			m.showVariables = false
		}
		return m, nil
	}
	if m.showCodeSnippet {
		if k.String() == "esc" || key.Matches(k, keys.CodeSnippet) {
			m.showCodeSnippet = false
		}
		return m, nil
	}
	// Unlike showVariables/showCodeSnippet above, a maximized Response only
	// intercepts esc (and the same key that maximized it) here - every other
	// key falls through to the rest of handleKey and eventually
	// dispatchKeyToFocusedWidget as normal, since a zoomed Response still
	// needs to scroll/copy/search/switch tabs exactly like a non-zoomed one.
	if m.responseMaximized && (k.String() == "esc" || key.Matches(k, keys.MaximizeResponse)) {
		m.responseMaximized = false
		return m, nil
	}
	if m.focus == focusSidebar && m.sidebar.IsFiltering() {
		// While the user is mid-filter, esc/ctrl+c must cancel the filter
		// (the list widget's own binding), not hit the global quit binding.
		var cmd tea.Cmd
		m.sidebar, cmd = m.sidebar.Update(k)
		return m, cmd
	}
	// esc does NOT close the drawer: a single esc should never hide a panel,
	// only cancel a modal-like sub-state (the filter above). The drawer hides
	// only via its own toggle (ctrl+\, keys.ToggleDrawer). esc while the
	// sidebar has focus falls through to the double-esc-to-quit handler below,
	// same as the side rail and the WebSocket screen.
	// Side rail routing sits ahead of the global bindings so its env editor
	// can own keys while a row is being edited (mirroring the ctrl+e panel),
	// but handleRailKey returns handled=false for anything it doesn't own -
	// Tab/shift+Tab and every global chord still work while the rail merely
	// has focus. Guarded by railFocusable() so a stale focusRail left over
	// from the rail going off-screen (a width shrink) can't swallow input.
	if m.focus == focusRail && m.railFocusable() {
		if newM, cmd, handled := m.handleRailKey(k); handled {
			return newM, cmd
		}
	}

	// Vim modes on the free-text editors (URL/Body/Scripts - see mode.go). In
	// INSERT, esc returns to NORMAL (caught before RestEsc's double-tap-quit);
	// the global chords below still run so ctrl+s/ctrl+r work while typing. In
	// NORMAL, i/a/c enter INSERT; every other key falls through to the command
	// routing, and dispatchKeyToFocusedWidget won't type into the field.
	if m.insertableFocus() {
		if m.mode == modeInsert {
			if k.String() == "esc" {
				m.enterNormal()
				return m, nil
			}
		} else {
			switch k.String() {
			case "i", "a", "c":
				m.enterInsert()
				return m, nil
			}
		}
	}

	// NORMAL-mode vim keys (never while typing into a field). These are added
	// alongside the existing chords/function keys, not replacing them, so muscle
	// memory keeps working while the vim scheme lands. A case that doesn't return
	// breaks out of the switch and falls through to the rest of handleKey (e.g.
	// g/G on the sidebar reach its own list, which already does top/bottom).
	if !m.textEntryFocused() {
		// g-prefix (vim goto): a bare g arms; the next key completes it - gg to
		// the top, gc/gd/gs/gw/gh jump to a screen. An unrecognized follow-up
		// just cancels the prefix and is processed normally below.
		if m.pendingG {
			m.pendingG = false
			switch k.String() {
			case "g":
				return m.gotoFocusedTop(), nil
			case "c":
				return m.openCollectionsScreen()
			case "d":
				return m.openDashboard()
			case "s":
				return m.openSettings()
			case "w":
				m.openWorkspaceSwitcher()
				return m, nil
			case "h":
				return m.openHistory()
			}
		}
		switch k.String() {
		case "g": // arm the goto prefix
			m.pendingG = true
			return m, nil
		case "G": // bottom of the focused list
			return m.gotoFocusedBottom(), nil
		case "q": // universal terminal-app quit (k9s/lazygit/yazi); ctrl+c still force-quits
			m.confirm.Open(confirmQuit, "Quit parley?", "")
			return m, nil
		case "z": // zoom the response (a terminal-safe alias for ctrl+z)
			if m.responseZoneVisible() {
				m.responseMaximized = true
				m.focus = focusResponse
				m.updateFocus()
			}
			return m, nil
		case "-": // collapse the focused sidebar section (safe alias for ctrl+\ / f1)
			if m.focus == focusSidebar {
				m.toggleDrawer()
			} else if m.focus == focusRail {
				m.toggleSideRail()
			}
			return m, nil
		case "/": // search the response (the sidebar handles its own / filter)
			if m.focus == focusResponse {
				m.prompt.Open(promptSearch, "", "Search response", m.response.searchTerm)
				return m, nil
			}
		}
	}

	// While a params/headers row's inline editor is open (an INSERT context),
	// esc cancels that edit - route it to the table before the global esc below
	// can arm the quit dialog instead. enter (commit) and tab (next field)
	// already reach the table since no global binding claims them here.
	if m.currentlyEditingRow() && k.String() == "esc" {
		return m.dispatchKeyToFocusedWidget(k)
	}
	// esc leaves the Response's Visual selection (route it to the viewer before
	// the global esc arms the quit dialog).
	if m.focus == focusResponse && m.response.visualActive && k.String() == "esc" {
		return m.dispatchKeyToFocusedWidget(k)
	}

	switch {
	case key.Matches(k, keys.Quit):
		return m, tea.Quit
	case key.Matches(k, keys.RestEsc):
		// Reached only on the Request screen (Collections/Dashboard/Settings
		// all early-return before this switch - see above), and Request is
		// the home screen now (see screen.go), so a bare esc with nothing
		// else open follows the same double-tap-to-arm-quit contract as
		// Collections used to when it was the default - there's nowhere
		// further back to go. See handleCollectionsScreenKey for the
		// Collections screen's own esc, which instead returns to
		// previousScreen now that it's a screen you navigate to, not from.
		return m.handleRestEsc()
	case key.Matches(k, keys.Send):
		return m.trySend()
	case key.Matches(k, keys.Save):
		return m.saveCurrentRequest()
	case key.Matches(k, keys.SaveExample):
		return m.startSaveExample()
	case key.Matches(k, keys.History):
		return m.openHistory()
	case key.Matches(k, keys.Import):
		return m.startImport()
	case key.Matches(k, keys.Export):
		return m.startExport()
	case key.Matches(k, keys.Help):
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
	case key.Matches(k, keys.JumpRequest):
		m.focus = focusRequest
		m.updateFocus()
		return m, nil
	case key.Matches(k, keys.JumpResponse):
		m.focus = focusResponse
		m.updateFocus()
		return m, nil
	case key.Matches(k, keys.JumpRail):
		m.jumpToRail()
		return m, nil
	case key.Matches(k, keys.ToggleDrawer):
		m.toggleDrawer()
		return m, nil
	case key.Matches(k, keys.ToggleOrientation):
		if m.orientation == workspace.OrientationVertical {
			m.orientation = workspace.OrientationHorizontal
		} else {
			m.orientation = workspace.OrientationVertical
		}
		m.persistLayout()
		return m, nil
	case key.Matches(k, keys.ToggleRequestZone):
		m.requestCollapsed = !m.requestCollapsed
		m.persistLayout()
		return m, nil
	case key.Matches(k, keys.ToggleResponseZone):
		m.responseCollapsed = !m.responseCollapsed
		m.persistLayout()
		return m, nil
	case key.Matches(k, keys.ToggleSideRail):
		m.toggleSideRail()
		return m, nil
	case key.Matches(k, keys.MaximizeResponse):
		// A no-op with nothing to zoom into yet - HasContent-backed, same
		// guard computeWorkspaceGeom itself uses (see responseZoneVisible).
		if m.responseZoneVisible() {
			m.responseMaximized = true
			m.focus = focusResponse
			m.updateFocus()
		}
		return m, nil
	case key.Matches(k, keys.Workspaces):
		m.openWorkspaceSwitcher()
		return m, nil
	case key.Matches(k, keys.Dashboard):
		return m.openDashboard()
	}

	// These share a ctrl-chord with a focused text widget's own default
	// editing keys (see textEntryFocused's doc comment) - skip them while
	// one has focus so the keystroke falls through to
	// dispatchKeyToFocusedWidget below instead of hijacking it.
	if !m.textEntryFocused() {
		switch {
		case key.Matches(k, keys.ToggleTab):
			m.response.CycleMode()
			return m, nil
		case key.Matches(k, keys.Search):
			m.prompt.Open(promptSearch, "", "Search response", m.response.searchTerm)
			return m, nil
		case key.Matches(k, keys.Environments):
			m.openEnvDropdown()
			return m, nil
		case key.Matches(k, keys.GoEnvPanel):
			// Jump into the Environment panel's env view - switching envs is
			// done inline there (left/right on the selector). Bound to alt+e
			// because Super+e can't be delivered to a terminal app (the window
			// manager grabs the Super key); f10 also jumps to this panel.
			m.railMode = railModeEnv
			m.jumpToRail()
			return m, nil
		case key.Matches(k, keys.ToggleVars):
			m.showVariables = !m.showVariables
			return m, nil
		case key.Matches(k, keys.CodeSnippet):
			m.showCodeSnippet = !m.showCodeSnippet
			return m, nil
		case key.Matches(k, keys.RevealSecrets):
			m.revealSecrets = !m.revealSecrets
			m.response.SetRevealSecrets(m.revealSecrets)
			m.body.formTable.SetRevealSecrets(m.revealSecrets)
			return m, nil
		case key.Matches(k, keys.Palette):
			m.palette.Open()
			return m, nil
		}
	}

	if m.focus == focusSidebar {
		if newM, cmd, handled := m.handleSidebarKey(k); handled {
			return newM, cmd
		}
	}

	// Global "n" = New (flow-aware): from anywhere that isn't a text field or
	// the Explorer itself, focus the Explorer and open the next-step create
	// prompt - so the flow bar's "n new" works without tabbing over first. The
	// Explorer's own handler owns "n" while it's focused (above). It's excluded
	// over the URL bar and every consume-all-keys editor (body/scripts/auth,
	// inline row edits) so an "n" being typed is never hijacked - textEntry-
	// Focused covers those editors but deliberately not the URL input, so that
	// one is named explicitly.
	if k.String() == "n" && m.focus != focusSidebar && m.focus != focusURL &&
		!m.textEntryFocused() && !m.currentlyEditingRow() {
		m.focus = focusSidebar
		m.updateFocus()
		newM, cmd, _ := m.newFromExplorer()
		return newM, cmd
	}

	if !m.currentlyEditingRow() {
		switch {
		case key.Matches(k, keys.NextFocus):
			m.mode = modeNormal // moving panes always lands in NORMAL
			m.focus = advanceFocus(m.tabStops(), m.focus, 1)
			m.updateFocus()
			return m, nil
		case key.Matches(k, keys.PrevFocus):
			m.mode = modeNormal
			m.focus = advanceFocus(m.tabStops(), m.focus, -1)
			m.updateFocus()
			return m, nil
		}
		if m.focus == focusRequest {
			switch {
			case key.Matches(k, keys.PrevReqTab):
				m.reqTab = nextReqTab(m.reqTab, -1)
				m.updateFocus()
				return m, nil
			case key.Matches(k, keys.NextReqTab):
				m.reqTab = nextReqTab(m.reqTab, 1)
				m.updateFocus()
				return m, nil
			case key.Matches(k, keys.AddRow) && m.reqTab == reqTabParams:
				m.kvAdd.Open(kvAddParam)
				return m, nil
			case key.Matches(k, keys.AddRow) && m.reqTab == reqTabHeaders:
				m.kvAdd.Open(kvAddHeader)
				return m, nil
			case k.String() == "i" && m.reqTab == reqTabParams:
				// i opens the selected row's inline editor - parity with the
				// URL/Body i-to-insert (enter does the same, below via dispatch).
				m.params, _, _ = m.params.Update(tea.KeyMsg{Type: tea.KeyEnter})
				return m, nil
			case k.String() == "i" && m.reqTab == reqTabHeaders:
				m.headers, _, _ = m.headers.Update(tea.KeyMsg{Type: tea.KeyEnter})
				return m, nil
			}
		}
		// Response-panel tab nav: the mirror of the Request block above, moving
		// between Body/Headers/Cookies/Tests/Timeline while the Response holds
		// focus. shift+left/right is the documented key (same as the Request
		// tabs), but a lot of terminals and tmux downgrade shift+arrow to a
		// plain arrow - so plain left/right and vim h/l are accepted too. They
		// have no other job on a read-only viewer (it scrolls with up/down/j/k),
		// and this is exactly the "shift+left/right does nothing" a user hit.
		// ctrl+t still cycles forward from anywhere as a last resort.
		if m.focus == focusResponse && !m.response.visualActive {
			switch {
			case key.Matches(k, keys.PrevReqTab), k.String() == "left", k.String() == "h":
				m.response.CycleModeBy(-1)
				return m, nil
			case key.Matches(k, keys.NextReqTab), k.String() == "right", k.String() == "l":
				m.response.CycleModeBy(1)
				return m, nil
			}
		}
	}

	if m.focus == focusMethod {
		switch k.String() {
		case "left", "h":
			m.methodIdx = (m.methodIdx + len(collection.Methods) - 1) % len(collection.Methods)
		case "right", "l", " ":
			m.methodIdx = (m.methodIdx + 1) % len(collection.Methods)
		}
		return m, nil
	}

	if m.focus == focusSend {
		switch k.String() {
		case "enter", " ":
			return m.trySend()
		}
		return m, nil
	}

	return m.dispatchKeyToFocusedWidget(k)
}

// gotoFocusedTop / gotoFocusedBottom implement gg / G for the focused list: the
// Response body jumps to its first/last line, the Collections list to its
// first/last row (via the list widget's own home/end bindings). Other focuses
// have no list to move, so they're no-ops.
func (m Model) gotoFocusedTop() Model {
	switch m.focus {
	case focusResponse:
		m.response.ScrollTop()
	case focusSidebar:
		m.sidebar, _ = m.sidebar.Update(tea.KeyMsg{Type: tea.KeyHome})
	}
	return m
}

func (m Model) gotoFocusedBottom() Model {
	switch m.focus {
	case focusResponse:
		m.response.ScrollBottom()
	case focusSidebar:
		m.sidebar, _ = m.sidebar.Update(tea.KeyMsg{Type: tea.KeyEnd})
	}
	return m
}
