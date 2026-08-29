package tui

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestAuthEditor_RoundTripsAuthCaptureAndRefresh(t *testing.T) {
	a := newAuthEditor()
	a.SetAuth(
		collection.AuthCapture{TokenField: "data.token", TokenVar: "token", ExpiresInField: "expiresIn", ExpiresAtVar: "tokenExpiresAt"},
		collection.RefreshConfig{RequestPath: "Auth/refresh.json", ExpiresAtVar: "tokenExpiresAt"},
	)

	gotCapture := a.AuthCapture()
	if gotCapture.TokenField != "data.token" || gotCapture.TokenVar != "token" {
		t.Errorf("AuthCapture = %+v", gotCapture)
	}
	if gotCapture.ExpiresInField != "expiresIn" || gotCapture.ExpiresAtVar != "tokenExpiresAt" {
		t.Errorf("AuthCapture = %+v", gotCapture)
	}
	gotRefresh := a.RefreshConfig()
	if gotRefresh.RequestPath != "Auth/refresh.json" || gotRefresh.ExpiresAtVar != "tokenExpiresAt" {
		t.Errorf("RefreshConfig = %+v", gotRefresh)
	}
}

// TestAuthEditor_BothSectionsStartCollapsed guards the design spec's
// progressive disclosure: Token Capture and Refresh Flow fields are opt-in
// details a request may not even use, so they start hidden behind a
// section header rather than always cluttering the tab.
func TestAuthEditor_BothSectionsStartCollapsed(t *testing.T) {
	a := newAuthEditor()
	a.SetSize(60, 20)

	got := stripANSI(a.View())
	if !strings.Contains(got, "Token Capture") || !strings.Contains(got, "Refresh Flow") {
		t.Errorf("View() missing section headers:\n%s", got)
	}
	for _, label := range authFieldLabels {
		if strings.Contains(got, label) {
			t.Errorf("View() shows field %q while collapsed, want it hidden:\n%s", label, got)
		}
	}
}

// TestAuthEditor_EnterOnHeaderExpandsItsSection guards the reveal action -
// picking a collapsed section and pressing enter (or space) shows its
// fields, the same activation keys used elsewhere in this app (env box,
// send button).
func TestAuthEditor_EnterOnHeaderExpandsItsSection(t *testing.T) {
	a := newAuthEditor()
	a.SetFocus(true)
	a.SetSize(60, 20)

	a, _, _ = a.Update(tea.KeyMsg{Type: tea.KeyEnter})

	got := stripANSI(a.View())
	if !strings.Contains(got, authFieldLabels[authFieldTokenField]) {
		t.Errorf("expected Token Capture's fields visible after expanding, got:\n%s", got)
	}
	if strings.Contains(got, authFieldLabels[authFieldRefreshPath]) {
		t.Errorf("expected Refresh Flow to remain collapsed, got:\n%s", got)
	}
}

// TestAuthEditor_CollapsingKeepsValues guards against data loss: hiding a
// section must not clear whatever was typed into its fields.
func TestAuthEditor_CollapsingKeepsValues(t *testing.T) {
	a := newAuthEditor()
	a.SetFocus(true)

	a, _, _ = a.Update(tea.KeyMsg{Type: tea.KeyEnter}) // expand Token Capture
	a, _, _ = a.Update(tea.KeyMsg{Type: tea.KeyDown})  // onto Token field
	a, _, _ = a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("data.token")})
	a, _, _ = a.Update(tea.KeyMsg{Type: tea.KeyUp})    // back onto the header
	a, _, _ = a.Update(tea.KeyMsg{Type: tea.KeyEnter}) // collapse it again

	if got := a.AuthCapture().TokenField; got != "data.token" {
		t.Errorf("TokenField = %q, want it preserved after collapsing", got)
	}
}

// TestAuthEditor_UpDownMovesThroughVisibleRowsOnly guards cursor movement
// against a stale field count: collapsed, there are only 2 rows (the two
// headers) to cycle through, not all 6 fields.
func TestAuthEditor_UpDownMovesThroughVisibleRowsOnly(t *testing.T) {
	a := newAuthEditor()
	a.SetFocus(true)
	if a.cursor != 0 {
		t.Fatalf("cursor = %d, want 0 initially", a.cursor)
	}

	a, _, handled := a.Update(tea.KeyMsg{Type: tea.KeyDown})
	if !handled {
		t.Fatal("expected down to be handled")
	}
	if a.cursor != 1 {
		t.Errorf("cursor = %d, want 1 (the Refresh Flow header) while collapsed", a.cursor)
	}

	a, _, _ = a.Update(tea.KeyMsg{Type: tea.KeyDown})
	if a.cursor != 0 {
		t.Errorf("cursor = %d, want wrap back to 0 with only 2 collapsed rows", a.cursor)
	}
}

func TestAuthEditor_TypingGoesIntoTheFocusedFieldWhenExpanded(t *testing.T) {
	a := newAuthEditor()
	a.SetFocus(true)

	a, _, _ = a.Update(tea.KeyMsg{Type: tea.KeyEnter}) // expand Token Capture
	a, _, _ = a.Update(tea.KeyMsg{Type: tea.KeyDown})  // onto Token field row

	a, _, handled := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if !handled {
		t.Fatal("expected typing to be handled")
	}
	if a.fields[authFieldTokenField].Value() != "t" {
		t.Errorf("fields[TokenField].Value() = %q, want %q", a.fields[authFieldTokenField].Value(), "t")
	}
}
