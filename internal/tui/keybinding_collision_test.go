package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestGlobalCtrlShortcuts_DoNotHijackTypingInBody guards a real bug: several
// global shortcuts (Environments ctrl+e, ToggleVars ctrl+w, Palette ctrl+k,
// RevealSecrets ctrl+u, ToggleTab ctrl+t, CodeSnippet ctrl+n) share a
// keystroke with an emacs-style editing shortcut bubbles/textarea already
// binds by default (LineEnd, DeleteWordBackward, DeleteAfterCursor,
// DeleteBeforeCursor, TransposeCharacterBackward, LineNext) - typing
// ordinary line/word-editing muscle memory into the Body tab kept popping
// open unrelated modals instead of editing the text.
func TestGlobalCtrlShortcuts_DoNotHijackTypingInBody(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.focus = focusRequest
	m.reqTab = reqTabBody
	m.updateFocus()
	m.mode = modeInsert // typing in the body requires INSERT now (see mode.go)
	m.body.typeIdx = rawTypeIdx()

	cases := []struct {
		name string
		key  tea.KeyMsg
	}{
		{"ctrl+e", tea.KeyMsg{Type: tea.KeyCtrlE}},
		{"ctrl+w", tea.KeyMsg{Type: tea.KeyCtrlW}},
		{"ctrl+k", tea.KeyMsg{Type: tea.KeyCtrlK}},
		{"ctrl+u", tea.KeyMsg{Type: tea.KeyCtrlU}},
		{"ctrl+t", tea.KeyMsg{Type: tea.KeyCtrlT}},
		{"ctrl+n", tea.KeyMsg{Type: tea.KeyCtrlN}},
	}
	for _, c := range cases {
		next, _ := m.Update(c.key)
		got := next.(Model)
		if got.envDropdown.active || got.showVariables || got.palette.active || got.showCodeSnippet || got.revealSecrets {
			t.Errorf("%s: opened a global modal/panel or toggled a global flag while typing in the body - want it to reach the textarea instead", c.name)
		}
		if got.response.mode != m.response.mode {
			t.Errorf("%s: changed the response tab while typing in the body - want it to reach the textarea instead", c.name)
		}
	}
}

// TestGlobalCtrlShortcuts_StillWorkOutsideAnyTextEntry checks the guard
// above doesn't accidentally swallow these shortcuts everywhere - only while
// a text-entry widget would otherwise receive the same keystroke.
func TestGlobalCtrlShortcuts_StillWorkOutsideAnyTextEntry(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.focus = focusResponse

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	got := next.(Model)
	if !got.envDropdown.active {
		t.Error("ctrl+e outside any text entry: want it to still open the environments dropdown")
	}
}

// TestRevealSecretsAndCodeSnippet_UseTheirOwnDedicatedChords guards the
// remap made when ctrl+u was needed for secret reveal: ctrl+u now toggles
// revealSecrets, and ctrl+n (freed up from nothing - it was previously
// unbound globally) opens the code snippet panel that used to live on
// ctrl+u.
func TestRevealSecretsAndCodeSnippet_UseTheirOwnDedicatedChords(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.screen = ScreenRequest
	m.focus = focusResponse

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	got := next.(Model)
	if !got.revealSecrets {
		t.Error("ctrl+u outside any text entry: want it to toggle revealSecrets")
	}

	next, _ = got.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
	got = next.(Model)
	if got.revealSecrets {
		t.Error("ctrl+u pressed again: want it to toggle revealSecrets back off")
	}

	next, _ = got.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	got = next.(Model)
	if !got.showCodeSnippet {
		t.Error("ctrl+n outside any text entry: want it to open the code snippet panel")
	}
}

// TestGlobalCtrlShortcuts_StillOpenFromTheURLBar guards the URL field's
// deliberate exclusion from the guard above: it's the app's default
// startup focus, and ctrl+e opening the environment dropdown from there is
// an established, separately-tested contract (envpanel_integration_test.go)
// that a blanket text-entry guard must not break.
func TestGlobalCtrlShortcuts_StillOpenFromTheURLBar(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.focus = focusURL
	m.updateFocus()

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlE})
	got := next.(Model)
	if !got.envDropdown.active {
		t.Error("ctrl+e while the URL bar has focus: want it to still open the environments dropdown")
	}
}
