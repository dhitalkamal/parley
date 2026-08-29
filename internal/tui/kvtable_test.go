package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TestKVTable_EmptyStateExplainsPurposeInsteadOfABlankTable guards a real
// confusion a user hit: an empty Params/Headers table used to render as a
// mostly-blank box (just column headers, then dozens of empty rows) with a
// tiny "a: add row" hint easy to miss below it - nothing on screen said
// what these rows are even for.
func TestKVTable_EmptyStateExplainsPurposeInsteadOfABlankTable(t *testing.T) {
	kv := newKVTable("Query params", "Key", "Query params are appended to the URL after ?")
	kv.SetWidth(60)
	kv.SetHeight(10)

	got := stripANSI(kv.View())
	if !strings.Contains(got, "appended to the URL") {
		t.Errorf("View() = %q, want it to explain what this table is for", got)
	}
	if !strings.Contains(got, "a: add row") {
		t.Errorf("View() = %q, want the add hint still shown", got)
	}
}

// TestKVTable_EmptyStateFillsTheGivenHeight checks the empty-state
// replacement fills the same height a populated table would (matching
// SetHeight's request), rather than shrinking to a short box pinned near
// the top with a large gap of otherwise-blank panel below it - a user
// flagged that top-left-anchored look via screenshot as inconsistent with
// the Response panel's already-centered empty state (see response.go's
// emptyStateView).
func TestKVTable_EmptyStateFillsTheGivenHeight(t *testing.T) {
	kv := newKVTable("Query params", "Key", "explanation")
	kv.SetWidth(60)
	kv.SetHeight(10)

	got := kv.View()
	// SetHeight(10) plus the border (2, no vertical padding) is 12.
	if h := lipgloss.Height(got); h != 12 {
		t.Errorf("got height %d, want 12 (SetHeight's 10 plus the 2-row border)", h)
	}
	if kv.ExtraLines() != 0 {
		t.Errorf("ExtraLines() = %d, want 0 - the empty-state message now lives inside the table's own box, not appended below it", kv.ExtraLines())
	}
}

// TestKVTable_EmptyStateIsVerticallyCentered mirrors
// TestResponseView_EmptyStateIsVerticallyCentered in response_test.go - the
// same "centered, not pinned to the top" treatment applies here now.
// Unlike responseView.ContentView(), kvTable.View() includes its own
// border, so row 0 and the last row are always non-blank border lines -
// those are excluded before looking for the message's first line.
func TestKVTable_EmptyStateIsVerticallyCentered(t *testing.T) {
	kv := newKVTable("Query params", "Key", "explanation")
	kv.SetWidth(60)
	kv.SetHeight(20)

	lines := strings.Split(stripANSI(kv.View()), "\n")
	if len(lines) != 22 {
		t.Fatalf("got %d lines, want 22 (SetHeight's 20 plus the 2-row border)", len(lines))
	}
	interior := lines[1 : len(lines)-1]
	firstNonBlank := -1
	for i, line := range interior {
		runes := []rune(line)
		if len(runes) < 2 {
			continue
		}
		inner := string(runes[1 : len(runes)-1]) // drop the left/right border rune
		if strings.TrimSpace(inner) != "" {
			firstNonBlank = i
			break
		}
	}
	if firstNonBlank < 6 {
		t.Errorf("first non-blank interior line is row %d of %d, want it pushed down toward vertical center, not pinned near the top", firstNonBlank, len(interior))
	}
}

// TestKVTable_EmptyStateIsHorizontallyCentered mirrors
// TestResponseView_EmptyStateIsHorizontallyCentered. As above, the leading
// and trailing border/padding column is stripped first.
func TestKVTable_EmptyStateIsHorizontallyCentered(t *testing.T) {
	kv := newKVTable("Query params", "Key", "explanation")
	kv.SetWidth(60)
	kv.SetHeight(20)

	lines := strings.Split(stripANSI(kv.View()), "\n")
	for _, line := range lines[1 : len(lines)-1] {
		runes := []rune(line)
		if len(runes) < 2 {
			continue
		}
		inner := string(runes[1 : len(runes)-1]) // drop the left/right border rune
		if trimmed := strings.TrimLeft(inner, " "); trimmed != "" {
			leadingSpaces := len(inner) - len(trimmed)
			if leadingSpaces < 4 {
				t.Errorf("line %q starts at column %d, want it indented toward horizontal center", inner, leadingSpaces)
			}
			return
		}
	}
	t.Fatal("no non-blank line found")
}

// TestKVTable_AddRowAppendsAnEnabledRow guards the non-interactive append
// the kvAdd modal calls on submit - no inline edit form, just a plain row.
func TestKVTable_AddRowAppendsAnEnabledRow(t *testing.T) {
	kv := newKVTable("Headers", "Key", "explanation")
	kv.AddRow("Content-Type", "application/json")

	rows := kv.Rows()
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	if rows[0].Key != "Content-Type" || rows[0].Value != "application/json" || !rows[0].Enabled {
		t.Errorf("got %+v, want Key=Content-Type Value=application/json Enabled=true", rows[0])
	}
	if kv.editing {
		t.Error("AddRow should not enter the inline edit form - the kvAdd modal already collected the values")
	}
}

// TestKVTable_UpdateNoLongerHandlesA guards the removal of the old inline
// add-row flow - "a" is now intercepted at the root level to open the kvAdd
// modal instead, so kvTable itself must not still react to it.
// TestKVTable_MasksSecretLookingRowsOnlyWhenOptedIn guards two things at
// once: a table that opts into masking (the body's Form/Multipart table)
// hides a secret-looking value by default and reveals it once told to, and
// a table that never opted in (params/headers) keeps showing literal values
// even for a row named "password" - masking those was never asked for and
// would silently change behavior nobody requested.
func TestKVTable_MasksSecretLookingRowsOnlyWhenOptedIn(t *testing.T) {
	masked := newKVTable("Form fields", "Key", "")
	masked.SetMaskSecrets(true)
	masked.SetWidth(60)
	masked.SetRows([]kvRow{{Key: "password", Value: "hunter2", Enabled: true}})

	got := stripANSI(masked.View())
	if strings.Contains(got, "hunter2") {
		t.Errorf("View() = %q, want the password value masked by default", got)
	}
	if !strings.Contains(got, "****") {
		t.Errorf("View() = %q, want a **** mask", got)
	}

	masked.SetRevealSecrets(true)
	got = stripANSI(masked.View())
	if !strings.Contains(got, "hunter2") {
		t.Errorf("View() = %q, want the password value revealed", got)
	}

	unmasked := newKVTable("Headers", "Key", "")
	unmasked.SetWidth(60)
	unmasked.SetRows([]kvRow{{Key: "password", Value: "hunter2", Enabled: true}})
	if got := stripANSI(unmasked.View()); !strings.Contains(got, "hunter2") {
		t.Errorf("View() = %q, want a table that never opted into masking to still show the literal value", got)
	}
}

// TestKVTable_ShowsRowNumberAndDescriptionColumns guards the design spec's
// compact terminal table: each row shows its position (#) and an optional
// description alongside the existing enabled/key/value columns.
func TestKVTable_ShowsRowNumberAndDescriptionColumns(t *testing.T) {
	kv := newKVTable("Query params", "Key", "explanation")
	kv.SetWidth(80)
	kv.SetRows([]kvRow{
		{Key: "page", Value: "1", Enabled: true, Description: "Page number"},
		{Key: "limit", Value: "20", Enabled: true, Description: "Page size"},
	})

	got := stripANSI(kv.View())
	if !strings.Contains(got, "Page number") {
		t.Errorf("View() = %q, want the first row's description", got)
	}
	if !strings.Contains(got, "Page size") {
		t.Errorf("View() = %q, want the second row's description", got)
	}
	if !strings.Contains(got, "1") || !strings.Contains(got, "2") {
		t.Errorf("View() = %q, want row numbers 1 and 2", got)
	}
}

// TestKVTable_EditFormIncludesDescriptionField guards the third field the
// inline edit form gained alongside key/value - tab now cycles through all
// three instead of wrapping straight from value back to key.
func TestKVTable_EditFormIncludesDescriptionField(t *testing.T) {
	kv := newKVTable("Query params", "Key", "explanation")
	kv.SetRows([]kvRow{{Key: "page", Value: "1", Enabled: true, Description: "Page number"}})
	kv.tbl.SetCursor(0)

	kv = kv.startEdit()
	if kv.descInput.Value() != "Page number" {
		t.Errorf("descInput = %q, want the row's existing description loaded", kv.descInput.Value())
	}

	// tab, tab: key -> value -> description
	kv, _, _ = kv.Update(tea.KeyMsg{Type: tea.KeyTab})
	kv, _, _ = kv.Update(tea.KeyMsg{Type: tea.KeyTab})
	if kv.editField != 2 {
		t.Fatalf("editField = %d, want 2 (description) after two tabs", kv.editField)
	}

	kv.descInput.SetValue("Updated description")
	kv = kv.commitEdit()
	if kv.Rows()[0].Description != "Updated description" {
		t.Errorf("got description %q, want it committed from the edit form", kv.Rows()[0].Description)
	}
}

func TestKVTable_UpdateNoLongerHandlesA(t *testing.T) {
	kv := newKVTable("Headers", "Key", "explanation")
	before := len(kv.Rows())

	got, _, _ := kv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if len(got.Rows()) != before {
		t.Errorf("got %d rows after \"a\", want %d (kvTable must not add a row itself anymore)", len(got.Rows()), before)
	}
	if got.editing {
		t.Error("\"a\" must not start the inline edit form anymore")
	}
}
