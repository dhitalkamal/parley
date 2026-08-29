package tui

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestLabelDropdownState_OpenStartsOnTheGivenSelection(t *testing.T) {
	var d labelDropdownState
	d.Open(2)
	if !d.active || d.cursor != 2 {
		t.Errorf("got active=%v cursor=%d, want active=true cursor=2", d.active, d.cursor)
	}
}

func TestLabelDropdownState_MoveCursorClampsWithinRange(t *testing.T) {
	var d labelDropdownState
	d.Open(0)
	d.MoveCursor(-1, 3)
	if d.cursor != 0 {
		t.Errorf("cursor = %d, want 0 (clamped)", d.cursor)
	}
	d.MoveCursor(5, 3)
	if d.cursor != 2 {
		t.Errorf("cursor = %d, want 2 (clamped to count-1)", d.cursor)
	}
}

func TestLabelDropdownState_Close(t *testing.T) {
	var d labelDropdownState
	d.Open(0)
	d.Close()
	if d.active {
		t.Error("expected Close to deactivate the dropdown")
	}
}

// TestBodyHeaderRightText_EmptyUnlessBodyTabActive guards the dropdown
// labels only showing in the Request zone header while the Body tab is
// the one actually selected - showing them over Params/Headers would be a
// live control with nothing to control.
func TestBodyHeaderRightText_EmptyUnlessBodyTabActive(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.reqTab = reqTabParams
	if got := m.bodyHeaderRightText(); got != "" {
		t.Errorf("got %q, want empty while Params is active", got)
	}
}

// TestBodyHeaderRightText_ShowsBodyTypeLabel guards the basic case: any
// body type shows its own label.
func TestBodyHeaderRightText_ShowsBodyTypeLabel(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.reqTab = reqTabBody
	m.body.SetBody(collection.Body{Type: collection.BodyMultipart})

	got := m.bodyHeaderRightText()
	if !strings.Contains(got, "Multipart") {
		t.Errorf("got %q, want it to contain %q", got, "Multipart")
	}
	if strings.Contains(got, "JSON") || strings.Contains(got, "XML") || strings.Contains(got, "Text") {
		t.Errorf("got %q, want no content-type label for a non-Raw body", got)
	}
}

// TestBodyHeaderRightText_ShowsContentTypeLabelOnlyForRaw guards the
// content-type dropdown's own visibility rule from the spec: shown for
// Raw, hidden for None/Form/Multipart/GraphQL/Binary.
func TestBodyHeaderRightText_ShowsContentTypeLabelOnlyForRaw(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.reqTab = reqTabBody
	m.body.SetBody(collection.Body{Type: collection.BodyRaw, RawContentType: "application/xml"})

	got := m.bodyHeaderRightText()
	if !strings.Contains(got, "Raw") {
		t.Errorf("got %q, want the body-type label", got)
	}
	if !strings.Contains(got, "XML") {
		t.Errorf("got %q, want the content-type label for Raw", got)
	}
}

// TestBodyDropdownClickTarget_MapsClickToTheRightDropdown guards the
// mouse-click affordance the header dropdowns add - clicking the body-type
// label opens that dropdown, clicking the content-type label opens the
// other one, and everything else on that row is neither.
func TestBodyDropdownClickTarget_MapsClickToTheRightDropdown(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 30
	m.reqTab = reqTabBody
	m.body.SetBody(collection.Body{Type: collection.BodyRaw, RawContentType: "application/json"})

	panelLeft, panelTop, panelWidth := 0, 5, 60
	title := zoneChevron(true) + " Request"
	right := m.bodyHeaderRightText() // "Raw v  JSON v"
	rightX := rightTextX(panelWidth, title, right)
	if rightX < 0 {
		t.Fatal("expected room for the right-side labels at this width")
	}
	textStart := panelLeft + rightX + 1

	isType, isCT := m.bodyDropdownClickTarget(textStart, panelTop, panelLeft, panelTop, panelWidth)
	if !isType || isCT {
		t.Errorf("click on the body-type label: got (isType=%v, isCT=%v), want (true, false)", isType, isCT)
	}

	ctStart := textStart + len("Raw v") + 2
	isType, isCT = m.bodyDropdownClickTarget(ctStart, panelTop, panelLeft, panelTop, panelWidth)
	if isType || !isCT {
		t.Errorf("click on the content-type label: got (isType=%v, isCT=%v), want (false, true)", isType, isCT)
	}

	isType, isCT = m.bodyDropdownClickTarget(panelLeft+3, panelTop, panelLeft, panelTop, panelWidth)
	if isType || isCT {
		t.Errorf("click on the title (left side): got (isType=%v, isCT=%v), want (false, false)", isType, isCT)
	}

	// A row below the header entirely - never a dropdown target.
	isType, isCT = m.bodyDropdownClickTarget(textStart, panelTop+1, panelLeft, panelTop, panelWidth)
	if isType || isCT {
		t.Errorf("click below the header row: got (isType=%v, isCT=%v), want (false, false)", isType, isCT)
	}
}

// TestMainView_ShowsBodyDropdownLabelsInRequestHeaderWhenBodyTabActive is
// the integration-level guard: the labels bodyHeaderRightText computes
// must actually reach the rendered frame, spliced into the Request zone's
// own header row rather than the old inline tab row (which body_view.go no
// longer renders at all).
func TestMainView_ShowsBodyDropdownLabelsInRequestHeaderWhenBodyTabActive(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 30
	m.reqTab = reqTabBody
	m.body.SetBody(collection.Body{Type: collection.BodyRaw, RawContentType: "application/json"})

	got := stripANSI(m.mainView())
	if !strings.Contains(got, "Raw"+dropdownArrow) {
		t.Errorf("got %q, want the body-type dropdown label (with the filled triangle) in the header", got)
	}
	if !strings.Contains(got, "JSON"+dropdownArrow) {
		t.Errorf("got %q, want the content-type dropdown label (with the filled triangle) in the header", got)
	}
}

// TestHandleMouse_ClickBodyTypeLabelOpensDropdownAndSelectingChangesType is
// the full end-to-end flow: click the body-type label in the header, pick
// a different entry via the keyboard, and confirm the body actually
// switched.
func TestHandleMouse_ClickBodyTypeLabelOpensDropdownAndSelectingChangesType(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 30
	m.screen = ScreenRequest
	m.reqTab = reqTabBody
	m.body.SetBody(collection.Body{Type: collection.BodyRaw})

	x, y := m.bodyTypeDropdownAnchor()
	got, _ := m.Update(press(x+1, y-1)) // the anchor is one row below the header itself
	after := got.(Model)
	if !after.bodyTypeDropdown.active {
		t.Fatalf("expected clicking the body-type label to open the dropdown")
	}

	// Multipart is bodyTypes[3] - move the cursor from Raw (1) down twice.
	after = sendKey(t, after, tea.KeyMsg{Type: tea.KeyDown})
	after = sendKey(t, after, tea.KeyMsg{Type: tea.KeyDown})
	after = sendKey(t, after, tea.KeyMsg{Type: tea.KeyEnter})

	if after.bodyTypeDropdown.active {
		t.Error("expected enter to close the dropdown")
	}
	if after.body.activeType() != collection.BodyMultipart {
		t.Errorf("body type = %v, want Multipart", after.body.activeType())
	}
}

// TestHandleMouse_ClickContentTypeLabelOpensDropdown guards the second
// dropdown's own click target, distinct from the body-type one right next
// to it.
func TestHandleMouse_ClickContentTypeLabelOpensDropdown(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 30
	m.screen = ScreenRequest
	m.reqTab = reqTabBody
	m.body.SetBody(collection.Body{Type: collection.BodyRaw, RawContentType: "application/json"})

	x, y := m.contentTypeDropdownAnchor()
	got, _ := m.Update(press(x+1, y-1))
	after := got.(Model)

	if !after.contentTypeDropdown.active {
		t.Error("expected clicking the content-type label to open that dropdown")
	}
	if after.bodyTypeDropdown.active {
		t.Error("expected the body-type dropdown to stay closed")
	}
}
