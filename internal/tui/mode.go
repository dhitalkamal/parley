package tui

import "github.com/charmbracelet/lipgloss"

// editorMode is the vim-style input mode for the Request screen's editing
// surfaces. NORMAL is the zero value, so a fresh model starts in command mode;
// single letters are actions there. INSERT is text entry - keys flow into the
// focused field until esc returns to NORMAL. Modals (palette, prompts, the
// collections/response navigation) have their own input handling and don't
// consult this - the mode governs the always-visible editors (URL bar, request
// body/scripts/auth, and open row edits).
type editorMode int

const (
	modeNormal editorMode = iota
	modeInsert
)

func (m editorMode) String() string {
	if m == modeInsert {
		return "INSERT"
	}
	return "NORMAL"
}

// insertableFocus reports whether the focused zone is a free-text field INSERT
// mode types into. This first stage covers the clean free-text editors - the
// URL bar and the request Body/Scripts editors. The params/headers row editors
// and Auth fields keep their existing always-type behavior for now (they have
// their own inline enter/esc semantics) and are converted in a later stage; the
// response/sidebar lists are navigation-only and never insertable.
func (m Model) insertableFocus() bool {
	switch m.focus {
	case focusURL:
		return true
	case focusRequest:
		switch m.reqTab {
		case reqTabBody, reqTabScripts:
			return true
		}
	}
	return false
}

// enterInsert switches to INSERT when the focus can accept text; a no-op
// otherwise (so pressing i on the response viewer does nothing).
func (m *Model) enterInsert() {
	if m.insertableFocus() {
		m.mode = modeInsert
		m.pendingG = false // an armed goto never survives into typing
	}
}

// enterNormal returns to NORMAL. Callers that were mid-row-edit cancel it first;
// free-text editors just stop receiving keys.
func (m *Model) enterNormal() {
	m.mode = modeNormal
}

// typingNow reports whether keys should flow into the focused text field right
// now - true only in INSERT with an insertable focus. This is what the rest of
// the app checks (via textEntryFocused) to keep single-letter commands and
// tab-digits out of a field being typed into.
func (m Model) typingNow() bool {
	return m.mode == modeInsert && m.insertableFocus()
}

// modeIndicator is the "-- NORMAL --"/"-- INSERT --" tag shown bottom-right on
// the Request screen: dim in NORMAL, bold accent in INSERT. It keys off
// textEntryFocused rather than the raw mode field, so it reads INSERT for every
// context where keys are actually flowing into a field - the explicitly-modal
// URL/Body/Scripts editors and the editing-derived Auth fields and open
// params/headers row editors alike.
func (m Model) modeIndicator() string {
	label := "-- NORMAL --"
	style := labelStyle
	accent := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(activeTheme.Accent))
	switch {
	case m.focus == focusResponse && m.response.visualActive:
		label, style = "-- VISUAL --", accent
	case m.textEntryFocused():
		label, style = "-- INSERT --", accent
	}
	return style.Render(label)
}
