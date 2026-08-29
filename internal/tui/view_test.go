package tui

import (
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	workspace "github.com/dhitalkamal/parley/internal/workspace/domain"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestSendBoxView_IsPlainBracketTextNoFill guards the flat redesign's
// button style: a user first asked for the button "filled fully" (a
// seamless solid block), then reported the resulting block as "too big" -
// pointing at the reference screenshot's own action button, which is plain
// "[ stay ]" bracket text with no background fill at all. Both properties
// (no fill, exactly 1 line) must hold in every focus state now, not just
// unfocused - there's no bordered/filled focused variant anymore.
// SetColorProfile pins TrueColor so the fill-color assertion is meaningful
// under `go test`'s non-TTY output (see TestSendBoxView_BorderIsFilledNotHollow's
// git history for why asserting on ANSI color codes without this is a silent
// no-op).
func TestSendBoxView_IsPlainBracketTextNoFill(t *testing.T) {
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(prev)

	for _, focused := range []bool{false, true} {
		got := sendBoxView(focused)
		if lines := strings.Split(got, "\n"); len(lines) != 1 {
			t.Errorf("focused=%v: rendered %d lines, want exactly 1\ngot: %q", focused, len(lines), got)
		}
		if strings.Contains(got, "48;2;") {
			t.Errorf("focused=%v: got a background fill color, want plain bracket text with no fill\ngot: %q", focused, got)
		}
		plain := stripANSI(got)
		if !strings.Contains(plain, "[") || !strings.Contains(plain, "]") || !strings.Contains(plain, "Send") {
			t.Errorf("focused=%v: got %q, want bracket text containing \"Send\"", focused, plain)
		}
	}
}

// TestModelView_RendersExactlyTheTerminalHeight guards a real regression:
// the 3-panel grid used to render taller than panelContentHeight's own
// budget (a request-panel tab bar wrapping instead of clipping, and the
// sidebar's empty-state hint not reserving space for itself), and
// panelContentHeight compensated with an unexplained "-1" fudge factor -
// together these left the whole View() one row short of the terminal
// (bubbletea's alt-screen renderer then scrolls the first row, the top
// bar, off-screen since there's nothing above it to absorb the shortfall).
// Both root causes are now fixed at their source, so this checks the
// actual contract holds again: View() always renders exactly m.height
// lines, for the default (collapsed) help bar.
func TestModelView_RendersExactlyTheTerminalHeight(t *testing.T) {
	for _, size := range []struct{ w, h int }{{80, 24}, {100, 32}, {120, 30}, {140, 34}} {
		m := New(t.TempDir(), t.TempDir())
		m.width, m.height = size.w, size.h

		got := m.View()
		lines := strings.Split(got, "\n")
		if len(lines) != size.h {
			t.Errorf("w=%d h=%d: got %d lines, want %d", size.w, size.h, len(lines), size.h)
		}
	}
}

// TestModelView_RendersExactlyTheTerminalHeightAcrossLayoutStates extends
// the contract above across every orientation x collapse-state combination
// the new workspace layout introduces - each one changes which rendering
// branch runs (see workspace_view.go), and each is its own opportunity to
// get a height budget off by one.
func TestModelView_RendersExactlyTheTerminalHeightAcrossLayoutStates(t *testing.T) {
	sizes := []struct{ w, h int }{{80, 24}, {100, 32}, {140, 34}}
	orientations := []workspace.Orientation{workspace.OrientationVertical, workspace.OrientationHorizontal}
	collapseStates := []struct{ req, resp bool }{
		{false, false}, {true, false}, {false, true}, {true, true},
	}

	for _, size := range sizes {
		for _, hasResponse := range []bool{false, true} {
			for _, o := range orientations {
				for _, c := range collapseStates {
					m := New(t.TempDir(), t.TempDir())
					m.width, m.height = size.w, size.h
					m.orientation = o
					m.requestCollapsed = c.req
					m.responseCollapsed = c.resp
					if hasResponse {
						m.response.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK"}, 10)
					}

					got := m.View()
					lines := strings.Split(got, "\n")
					if len(lines) != size.h {
						t.Errorf("w=%d h=%d orientation=%v req=%v resp=%v hasResponse=%v: got %d lines, want %d",
							size.w, size.h, o, c.req, c.resp, hasResponse, len(lines), size.h)
					}
				}
			}
		}
	}
}

// TestLeftSidebarWidth_FixedWhenShown guards that the shared left column is a
// fixed width (Collections and Environment stack in it), not a fraction of the
// terminal - a wide terminal just gives the center more room.
func TestLeftSidebarWidth_FixedWhenShown(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	for _, w := range []int{120, 240, 1000} {
		m.width = w
		if got := m.leftSidebarWidth(); got != leftSidebarOuterWidth {
			t.Errorf("leftSidebarWidth at width %d = %d, want fixed %d", w, got, leftSidebarOuterWidth)
		}
	}
}

// TestLeftSidebarWidth_ZeroWhenTooNarrow guards graceful degradation: below the
// floor the sidebar drops out entirely so the center stays usable.
func TestLeftSidebarWidth_ZeroWhenTooNarrow(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width = minWidthForSidebar - 1
	if got := m.leftSidebarWidth(); got != 0 {
		t.Errorf("leftSidebarWidth below the floor = %d, want 0", got)
	}
	if m.leftSidebarShown() {
		t.Error("leftSidebarShown should be false below the floor")
	}
}

func TestUrlRowWidth_SpansFullTerminalWidth(t *testing.T) {
	// The url row sits above the 3-panel grid, not beside the sidebar - it
	// should track the full terminal width, not shrink for the sidebar.
	got := urlRowWidth(240)
	want := 240 - 3
	if got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func TestUrlRowWidth_ClampsOnNarrowTerminal(t *testing.T) {
	got := urlRowWidth(10)
	if got < 20 {
		t.Errorf("got %d, want at least 20 even on a narrow terminal", got)
	}
}

func TestUrlBoxOuterWidth_FitsAlongsideMethodAndSendBoxesExactly(t *testing.T) {
	for _, total := range []int{60, 90, 140, 200} {
		urlW := urlBoxOuterWidth(total)
		got := methodBoxOuterWidth() + 1 + urlW + 1 + sendBoxWidth
		if got != total {
			t.Errorf("total %d: method+url+send+gaps = %d, want exactly %d", total, got, total)
		}
	}
}

// TestEnvDropdownAnchor_OpensRightOfSidebar guards that the switcher opens just
// to the RIGHT of the left sidebar (where the Environment panel now lives), so
// it doesn't cover the panel's own variables while picking.
func TestEnvDropdownAnchor_OpensRightOfSidebar(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 44
	m.activeEnvName = "staging"
	m.openEnvDropdown()

	if !m.leftSidebarShown() || !m.environmentExpanded() {
		t.Fatal("precondition: sidebar + Environment expanded at width 140")
	}
	colH, _ := m.sidebarSectionHeights()
	wantX := m.leftSidebarWidth() + 1
	wantY := colH + 4

	gotX, gotY := m.envDropdownAnchor()
	if gotX != wantX || gotY != wantY {
		t.Errorf("envDropdownAnchor() = (%d, %d), want (%d, %d)", gotX, gotY, wantX, wantY)
	}
	if gotX < m.leftSidebarWidth() {
		t.Errorf("dropdown x=%d should sit right of the sidebar (width %d), not over it", gotX, m.leftSidebarWidth())
	}
}

// TestMainView_RequestFillsWorkspaceBeforeAnyResponse guards item 3 of the
// workspace redesign: no response zone at all - not even an empty one -
// occupies space until a response actually exists.
func TestMainView_RequestFillsWorkspaceBeforeAnyResponse(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 30

	got := stripANSI(m.mainView())

	if !strings.Contains(got, "Params 0") {
		t.Errorf("got %q, want the Request zone's own tab bar visible", got)
	}
	if !strings.Contains(got, "No response yet") {
		t.Errorf("got %q, want the hint about sending a request", got)
	}
	if strings.Contains(got, glyphTriangleDown+" Response") || strings.Contains(got, glyphTriangleRight+" Response") {
		t.Errorf("got %q, want no Response zone header at all before anything's been sent", got)
	}
}

// TestMainView_ResponseZoneAppearsAfterASend guards the reveal half of the
// same contract.
func TestMainView_ResponseZoneAppearsAfterASend(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 30
	m.response.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK"}, 10)

	got := stripANSI(m.mainView())

	if !strings.Contains(got, "Response") {
		t.Errorf("got %q, want the Response zone visible once a response exists", got)
	}
	if strings.Contains(got, "No response yet") {
		t.Errorf("got %q, want the hint gone once a response exists", got)
	}
}

// TestMainView_RequestCollapsedGivesResponseTheFreedSpace guards
// independent zone collapse (f7/f8): collapsing Request leaves just its
// header bar, and Response expands into the rest.
func TestMainView_RequestCollapsedGivesResponseTheFreedSpace(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 30
	m.response.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK"}, 10)
	m.requestCollapsed = true

	got := stripANSI(m.mainView())

	if !strings.Contains(got, glyphTriangleRight+" Request") {
		t.Errorf("got %q, want a collapsed (>) Request header", got)
	}
	if strings.Contains(got, "Params 0") {
		t.Errorf("got %q, want the Request zone's own body gone while collapsed", got)
	}
	if !strings.Contains(got, glyphTriangleDown+" Response") {
		t.Errorf("got %q, want an expanded (v) Response header", got)
	}
}

// TestMainView_ResponseCollapsedGivesRequestTheFreedSpace is the mirror of
// the test above: collapsing Response leaves just its header bar, and
// Request expands into the rest.
func TestMainView_ResponseCollapsedGivesRequestTheFreedSpace(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 30
	m.response.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK"}, 10)
	m.responseCollapsed = true

	got := stripANSI(m.mainView())

	if !strings.Contains(got, glyphTriangleRight+" Response") {
		t.Errorf("got %q, want a collapsed (>) Response header", got)
	}
	if !strings.Contains(got, glyphTriangleDown+" Request") {
		t.Errorf("got %q, want an expanded (v) Request header", got)
	}
	if !strings.Contains(got, "Params 0") {
		t.Errorf("got %q, want the Request zone's own body still visible", got)
	}
}

// TestMainView_HorizontalOrientationPlacesZonesSideBySide guards the f6
// toggle: request and response render side by side, each full height,
// instead of stacked.
func TestMainView_HorizontalOrientationPlacesZonesSideBySide(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = 140, 30
	m.response.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK"}, 10)
	m.orientation = workspace.OrientationHorizontal

	got := stripANSI(m.mainView())
	lines := strings.Split(got, "\n")

	reqLine, respLine := -1, -1
	for i, l := range lines {
		if reqLine < 0 && strings.Contains(l, glyphTriangleDown+" Request") {
			reqLine = i
		}
		if respLine < 0 && strings.Contains(l, glyphTriangleDown+" Response") {
			respLine = i
		}
	}
	if reqLine < 0 || respLine < 0 {
		t.Fatalf("expected to find both zone headers, got:\n%s", got)
	}
	if reqLine != respLine {
		t.Errorf("Request header on line %d, Response header on line %d - want the same line (side by side)", reqLine, respLine)
	}
}

// TestMainView_HorizontalOrientationFallsBackToVerticalWhenTooNarrow
// guards the narrow-terminal safety net computeWorkspaceGeom applies in
// place of the old 3-panel grid's narrowGridWidth fallback.
func TestMainView_HorizontalOrientationFallsBackToVerticalWhenTooNarrow(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.width, m.height = workspaceHorizontalFloor-1, 30
	m.response.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK"}, 10)
	m.orientation = workspace.OrientationHorizontal

	got := stripANSI(m.mainView())
	lines := strings.Split(got, "\n")

	reqLine, respLine := -1, -1
	for i, l := range lines {
		if reqLine < 0 && strings.Contains(l, glyphTriangleDown+" Request") {
			reqLine = i
		}
		if respLine < 0 && strings.Contains(l, glyphTriangleDown+" Response") {
			respLine = i
		}
	}
	if reqLine < 0 || respLine < 0 {
		t.Fatalf("expected to find both zone headers, got:\n%s", got)
	}
	if reqLine == respLine {
		t.Errorf("Request and Response headers both on line %d, want them stacked (vertical fallback) below the width floor", reqLine)
	}
}
