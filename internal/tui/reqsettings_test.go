package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRequestSettingsEditor_DefaultsMatchZeroValueRequest(t *testing.T) {
	e := newRequestSettingsEditor()
	if e.Timeout() != 0 {
		t.Errorf("Timeout() = %v, want 0 (defers to execapp.DefaultTimeout downstream)", e.Timeout())
	}
	if !e.FollowRedirects() {
		t.Error("FollowRedirects() = false, want true by default (matches buildRequest's prior hardcoded default)")
	}
	if e.InsecureSkipVerify() {
		t.Error("InsecureSkipVerify() = true, want false by default")
	}
}

func TestRequestSettingsEditor_SetSettingsRoundTrips(t *testing.T) {
	e := newRequestSettingsEditor()
	e.SetSettings(45*time.Second, false, true)

	if e.Timeout() != 45*time.Second {
		t.Errorf("Timeout() = %v, want 45s", e.Timeout())
	}
	if e.FollowRedirects() {
		t.Error("FollowRedirects() = true, want false")
	}
	if !e.InsecureSkipVerify() {
		t.Error("InsecureSkipVerify() = false, want true")
	}
}

func TestRequestSettingsEditor_InvalidTimeoutTextParsesAsZero(t *testing.T) {
	e := newRequestSettingsEditor()
	e.timeout.SetValue("not a duration")
	if e.Timeout() != 0 {
		t.Errorf("Timeout() = %v, want 0 for unparseable text (defers to the default rather than blocking)", e.Timeout())
	}
}

func TestRequestSettingsEditor_SpaceTogglesFollowRedirectsWhenFocused(t *testing.T) {
	e := newRequestSettingsEditor()
	e.SetFocus(true)
	e.fieldIdx = reqSettingsFieldFollowRedirects

	got, _, _ := e.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if got.FollowRedirects() {
		t.Error("expected space to toggle FollowRedirects from its true default to false")
	}
}

func TestRequestSettingsEditor_SpaceTogglesInsecureSkipVerifyWhenFocused(t *testing.T) {
	e := newRequestSettingsEditor()
	e.SetFocus(true)
	e.fieldIdx = reqSettingsFieldInsecureSkipVerify

	got, _, _ := e.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if !got.InsecureSkipVerify() {
		t.Error("expected space to toggle InsecureSkipVerify from false to true")
	}
}

func TestRequestSettingsEditor_DownCyclesFields(t *testing.T) {
	e := newRequestSettingsEditor()
	e.SetFocus(true)

	got, _, _ := e.Update(tea.KeyMsg{Type: tea.KeyDown})
	if got.fieldIdx != reqSettingsFieldFollowRedirects {
		t.Errorf("fieldIdx = %v, want FollowRedirects after one down", got.fieldIdx)
	}
}

func TestRequestSettingsEditor_View_ShowsCompactControls(t *testing.T) {
	e := newRequestSettingsEditor()
	e.SetSettings(30*time.Second, true, false)
	out := stripANSI(e.View())

	if !strings.Contains(out, "Timeout") || !strings.Contains(out, "30s") {
		t.Errorf("expected timeout row, got:\n%s", out)
	}
	if !strings.Contains(out, "Follow redirects") || !strings.Contains(out, "[x]") {
		t.Errorf("expected follow-redirects checked, got:\n%s", out)
	}
	if !strings.Contains(out, "Skip TLS verify") || !strings.Contains(out, "[ ]") {
		t.Errorf("expected skip-tls-verify unchecked, got:\n%s", out)
	}
}
