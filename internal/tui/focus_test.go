package tui

import (
	"strings"
	"testing"
)

// assertFocusVisiblyDiffers is the shared assertion for every focus-indicator
// test below: focusing a widget must visibly change its border (shape, not
// just color - color alone disappears over a non-truecolor terminal and,
// not coincidentally, in this test environment too).
func assertFocusVisiblyDiffers(t *testing.T, name string, unfocused, focused string) {
	t.Helper()
	if stripANSI(unfocused) == stripANSI(focused) {
		t.Errorf("%s: View() identical whether focused or not, want a visible border change", name)
	}
}

// TestEffectiveFocus_IsRealFocusWhenNoModalIsOpen and the following test
// guard a real bug: mainView always highlighted whichever panel m.focus
// pointed to even while a modal had drawn over the whole screen and owned
// every keystroke - a stale, misleading "this panel is focused" signal
// showing through behind the modal.
func TestEffectiveFocus_IsRealFocusWhenNoModalIsOpen(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.focus = focusResponse
	if got := m.effectiveFocus(); got != focusResponse {
		t.Errorf("effectiveFocus() = %d, want %d (real focus, no modal open)", got, focusResponse)
	}
}

func TestEffectiveFocus_IsNoneWhenAModalIsOpen(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.focus = focusResponse
	m.palette.active = true
	if got := m.effectiveFocus(); got == focusResponse {
		t.Errorf("effectiveFocus() = %d, want it to NOT report a panel focused while the palette is open", got)
	}
}

// TestSendBox_FocusChangesBorder mirrors TestEnvBox_FocusChangesBorder for
// the send button, now that it's a real tab stop too. Baseline is
// focusSidebar (not focusURL) deliberately - it leaves every box in the
// url row in its own default unfocused state, so the only thing that can
// possibly differ from focusSend is the send button itself, not the url
// box losing its own highlight.
func TestSendBox_FocusChangesBorder(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44

	m.focus = focusSidebar
	unfocused := m.urlRowView(urlRowWidth(m.width))
	m.focus = focusSend
	focused := m.urlRowView(urlRowWidth(m.width))

	assertFocusVisiblyDiffers(t, "sendBox", unfocused, focused)
}

// TestSendBox_LabelHasNoShortcutText guards the cleaned-up button label -
// it just says "Send" now, not "Send ^R" - ctrl+r still sends from
// anywhere regardless of focus, so cluttering the button itself with the
// shortcut was redundant.
func TestSendBox_LabelHasNoShortcutText(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44

	got := stripANSI(m.urlRowView(urlRowWidth(m.width)))
	if !strings.Contains(got, "Send") {
		t.Errorf("got %q, want it to still say \"Send\"", got)
	}
	if strings.Contains(got, "^R") {
		t.Errorf("got %q, want no \"^R\" shortcut text inside the button", got)
	}
}

func TestSidebar_FocusChangesBorderNotContent(t *testing.T) {
	sb := newSidebar()
	sb.SetSize(30, 10)

	sb.SetFocused(false)
	unfocused := sb.View()
	sb.SetFocused(true)
	focused := sb.View()

	assertFocusVisiblyDiffers(t, "sidebar", unfocused, focused)
}

func TestSidebar_EmptyStateShowsCreateHintOnce(t *testing.T) {
	sb := newSidebar()
	sb.SetSize(30, 10)

	view := stripANSI(sb.View())
	if strings.Count(view, "No items") > 1 {
		t.Errorf("expected the empty-state message to appear once, got:\n%s", view)
	}
	if !strings.Contains(view, "new request") {
		t.Errorf("expected empty sidebar to hint how to create a request, got:\n%s", view)
	}
}

func TestKVTable_FocusChangesBorderNotContent(t *testing.T) {
	kv := newKVTable("Query params", "Key", "explanation")
	kv.SetWidth(60)

	kv.SetFocus(false)
	unfocused := kv.View()
	kv.SetFocus(true)
	focused := kv.View()

	assertFocusVisiblyDiffers(t, "kvTable", unfocused, focused)
}

// TestRequestPanelView_BodyTabFocusChangesBorder replaces the old
// bodyEditor-local focus test - bodyEditor no longer nests its own border
// (see body.go's View doc comment: at this panel's typical height, a
// second border ran nearly floor to ceiling and read as a stray line
// rather than a deliberate box), so the outer Request panel's own border is
// now the only focus indicator while on the Body tab.
func TestRequestPanelView_BodyTabFocusChangesBorder(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.reqTab = reqTabBody

	m.focus = focusSidebar
	unfocused := m.requestPanelView(60, 30)
	m.focus = focusRequest
	focused := m.requestPanelView(60, 30)

	assertFocusVisiblyDiffers(t, "requestPanelView (Body tab)", unfocused, focused)
}

// TestRequestPanelView_ScriptsTabFocusChangesBorder replaces the old
// scriptEditor-local focus test: scriptEditor no longer nests its own border
// (see scriptEditor.View - it renders the textarea flush now, same as the
// Body Raw editor and the params/headers tables, so the Request panel stops
// looking like a box-in-a-box next to the flush Response body). The outer
// Request panel's own border is the only focus indicator on the Scripts tab.
func TestRequestPanelView_ScriptsTabFocusChangesBorder(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.reqTab = reqTabScripts

	m.focus = focusSidebar
	unfocused := m.requestPanelView(60, 30)
	m.focus = focusRequest
	focused := m.requestPanelView(60, 30)

	assertFocusVisiblyDiffers(t, "requestPanelView (Scripts tab)", unfocused, focused)
}

// TestScriptEditor_RendersFlushNoBorder locks in the redesign: the script
// editor draws no border of its own (it used to), so it never renders a
// box-in-a-box inside the Request panel.
func TestScriptEditor_RendersFlushNoBorder(t *testing.T) {
	s := newScriptEditor()
	s.SetSize(60, 5)
	s.SetFocus(true)

	view := stripANSI(s.View())
	// corner glyphs a bordered widget would draw: rounded top-left (0x256D)
	// and thick top-left (0x250F). A flush textarea draws neither.
	for _, r := range []rune{0x256D, 0x250F} {
		if strings.ContainsRune(view, r) {
			t.Errorf("scriptEditor.View drew border glyph %q, want flush:\n%s", string(r), view)
		}
	}
}
