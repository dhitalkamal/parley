package tui

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TestResponseView_StatusLineIsBlankBeforeAnyRequestSent guards a real
// complaint: the status line used to say "Ready" while ContentView's own
// emptyStateView, one row below, already says "No response yet" - two
// slightly different messages for the same "nothing sent" state read as
// inconsistent rather than deliberate.
func TestResponseView_StatusLineIsBlankBeforeAnyRequestSent(t *testing.T) {
	rv := newResponseView()

	if got := rv.StatusLine(); got != "" {
		t.Errorf("StatusLine() = %q, want blank before anything's been sent", got)
	}
}

// TestResponseView_HasContentIsFalseBeforeAnythingIsSent guards the shared
// condition responsePanelView relies on to decide whether the
// Body/Headers/Cookies tabs are worth showing at all.
func TestResponseView_HasContentIsFalseBeforeAnythingIsSent(t *testing.T) {
	rv := newResponseView()
	if rv.HasContent() {
		t.Error("HasContent() = true before anything's been sent, want false")
	}

	rv.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK"}, 10)
	if !rv.HasContent() {
		t.Error("HasContent() = false after a response arrived, want true")
	}
}

// TestResponsePanelView_HidesTabsWhenNoResponseYet guards a real
// complaint: Body/Headers/Cookies showed even before anything was sent,
// looking like a live control with nothing to actually switch between.
func TestResponsePanelView_HidesTabsWhenNoResponseYet(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())

	got := stripANSI(m.responsePanelView(60, 20))

	for _, label := range []string{"Body", "Headers", "Cookies"} {
		if strings.Contains(got, label) {
			t.Errorf("got %q, want no %q tab before anything's been sent", got, label)
		}
	}
}

// TestResponsePanelView_ShowsTabsOnceAResponseArrives is the regression
// guard: the tabs must still show up once there's actually something to
// switch between.
func TestResponsePanelView_ShowsTabsOnceAResponseArrives(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.response.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK", Body: []byte("hello")}, 10)

	got := stripANSI(m.responsePanelView(60, 20))

	for _, label := range []string{"Body", "Headers", "Cookies"} {
		if !strings.Contains(got, label) {
			t.Errorf("got %q, want a %q tab once a response has arrived", got, label)
		}
	}
}

func TestHumanSize(t *testing.T) {
	cases := []struct {
		bytes int
		want  string
	}{
		{0, "0 bytes"},
		{412, "412 bytes"},
		{1023, "1023 bytes"},
		{1024, "1.0 KB"},
		{1229, "1.2 KB"},
		{1024 * 1024, "1.0 MB"},
	}
	for _, c := range cases {
		if got := humanSize(c.bytes); got != c.want {
			t.Errorf("humanSize(%d) = %q, want %q", c.bytes, got, c.want)
		}
	}
}

// TestResponseView_TabBarUnderlinesTheActiveMode guards the move from a
// "[Body]" bracket marker to an underline (see tabbar.go), matching the
// Request panel's own reqTabBarText.
func TestResponseView_TabBarMarksActiveMode(t *testing.T) {
	rv := newResponseView()
	rv.SetSize(60, 20)

	got := stripANSI(rv.TabBarText())
	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2 (label row + rule row)", len(lines))
	}
	if lines[1] == "" || strings.Trim(lines[1], glyphHorizontalLine) != "" {
		t.Errorf("second row = %q, want a full horizontal rule", lines[1])
	}
	for _, want := range []string{"Body", "Headers", "Cookies"} {
		if !strings.Contains(lines[0], want) {
			t.Errorf("TabBarText() = %q, want %q listed", lines[0], want)
		}
	}

	// The active tab is a filled pill; its color isn't rendered in the test
	// environment (no TTY), so verify the active mode advances via the field.
	before := rv.mode
	rv.CycleMode()
	if rv.mode == before {
		t.Error("CycleMode should advance the active mode")
	}
}

// TestResponseView_TabBarHasNoFormatPill guards the fix for the response
// panel's old pretty/raw picker - a user asked for the response to always
// be fully formatted instead, with no format choice to show at all (that
// control now lives on the request body's own tab bar - see body.go).
func TestResponseView_TabBarHasNoFormatPill(t *testing.T) {
	rv := newResponseView()

	if got := rv.TabBarText(); strings.Contains(got, "Format:") {
		t.Errorf("TabBarText() = %q, want no \"Format:\" pill", got)
	}
}

// TestResponseView_EmptyStateExplainsPurposeAndShortcut guards a real
// complaint: before any request is sent, the response body area was a
// single line of text ("Response appears here...") floating at the top of
// an otherwise huge, unused blank panel - it read as broken rather than
// "empty and ready."
func TestResponseView_EmptyStateExplainsPurposeAndShortcut(t *testing.T) {
	rv := newResponseView()
	rv.SetSize(60, 20)

	got := stripANSI(rv.ContentView())
	if !strings.Contains(got, "ctrl+r") {
		t.Errorf("ContentView() = %q, want the send shortcut mentioned", got)
	}
	if !strings.Contains(strings.ToLower(got), "response") {
		t.Errorf("ContentView() = %q, want it to explain what belongs here", got)
	}
}

// TestResponseView_EmptyStateIsVerticallyCentered checks the message sits
// roughly in the middle of the viewport rather than pinned to the top row
// with acres of blank space below it - a bordered box that's technically
// the right height isn't the same as looking deliberately designed.
func TestResponseView_EmptyStateIsVerticallyCentered(t *testing.T) {
	rv := newResponseView()
	rv.SetSize(60, 20)

	lines := strings.Split(stripANSI(rv.ContentView()), "\n")
	if len(lines) != 20 {
		t.Fatalf("got %d lines, want 20", len(lines))
	}
	firstNonBlank := -1
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			firstNonBlank = i
			break
		}
	}
	if firstNonBlank < 6 {
		t.Errorf("first non-blank line is row %d of 20, want it pushed down toward vertical center, not pinned near the top", firstNonBlank)
	}
}

// TestResponseView_EmptyStateIsHorizontallyCentered checks the message text
// itself sits away from the left edge, not flush against column 0.
func TestResponseView_EmptyStateIsHorizontallyCentered(t *testing.T) {
	rv := newResponseView()
	rv.SetSize(60, 20)

	lines := strings.Split(stripANSI(rv.ContentView()), "\n")
	for _, line := range lines {
		if trimmed := strings.TrimLeft(line, " "); trimmed != "" {
			leadingSpaces := len(line) - len(trimmed)
			if leadingSpaces < 4 {
				t.Errorf("line %q starts at column %d, want it indented toward horizontal center", line, leadingSpaces)
			}
			return
		}
	}
	t.Fatal("no non-blank line found")
}

// TestResponseView_EmptyStateDoesNotWrapOnANarrowPanel guards against the
// word-wrap-instead-of-clip bug this session already hit more than once
// (lipgloss.Style.Width()/Place word-wraps over-budget content rather than
// clipping it) - the explanatory text is long enough to overflow a narrow
// Response panel if it isn't pre-wrapped to the actual available width.
func TestResponseView_EmptyStateDoesNotWrapOnANarrowPanel(t *testing.T) {
	rv := newResponseView()
	rv.SetSize(24, 10)

	got := rv.ContentView()
	if w := lipgloss.Width(got); w != 24 {
		t.Errorf("got width %d, want 24", w)
	}
	if h := lipgloss.Height(got); h != 10 {
		t.Errorf("got height %d, want 10", h)
	}
}

// TestResponseView_ContentViewShowsRealResponseAfterSend checks the empty
// state only shows up before anything's been sent - once a real response
// (or error) arrives, ContentView must show that instead.
func TestResponseView_ContentViewShowsRealResponseAfterSend(t *testing.T) {
	rv := newResponseView()
	rv.SetSize(60, 20)
	rv.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK", Body: []byte("hello")}, 12)

	got := rv.ContentView()
	if !strings.Contains(got, "hello") {
		t.Errorf("ContentView() = %q, want the real response body, not the empty state", got)
	}
}

// stubClipboard swaps writeClipboard for a fake that records what would
// have been copied, instead of ever touching the real system clipboard
// during a test run.
func stubClipboard(t *testing.T) *string {
	t.Helper()
	var got string
	orig := writeClipboard
	writeClipboard = func(s string) error {
		got = s
		return nil
	}
	t.Cleanup(func() { writeClipboard = orig })
	return &got
}

// TestResponseView_YCopiesCurrentContentToClipboard guards the actual ask:
// a user with a long value (an auth token) that ran off the edge of the
// panel needs an easy way to grab the whole thing, not just whatever
// happens to be visible.
func TestResponseView_YCopiesCurrentContentToClipboard(t *testing.T) {
	copied := stubClipboard(t)
	rv := newResponseView()
	rv.SetSize(60, 20)
	rv.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK", Body: []byte(`{"a":1}`)}, 10)

	rv, _ = rv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	if *copied == "" {
		t.Fatal("expected y to copy the current content to the clipboard")
	}
	if strings.Contains(*copied, "\x1b") {
		t.Errorf("copied text contains a raw ANSI escape byte, want syntax highlighting stripped: %q", *copied)
	}
	if rv.copiedAt.IsZero() {
		t.Error("expected y to record when the copy happened, for StatusLine's confirmation")
	}
}

// TestResponseView_StatusLineShowsCopiedConfirmation guards the visible
// half of the fix - copying has to actually show feedback, not just
// silently write to the clipboard with nothing to confirm it worked.
func TestResponseView_StatusLineShowsCopiedConfirmation(t *testing.T) {
	rv := newResponseView()
	rv.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK"}, 10)
	rv.copiedAt = time.Now()

	if got := rv.StatusLine(); !strings.Contains(got, "Copied") {
		t.Errorf("StatusLine() = %q, want a \"Copied\" confirmation", got)
	}
}

// TestResponseView_StatusLineConfirmationFadesAfterTheWindow guards against
// a stale "Copied to clipboard" sticking around forever and masking the
// actual response status.
func TestResponseView_StatusLineConfirmationFadesAfterTheWindow(t *testing.T) {
	rv := newResponseView()
	rv.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK"}, 10)
	rv.copiedAt = time.Now().Add(-copiedConfirmationWindow - time.Second)

	if got := rv.StatusLine(); strings.Contains(got, "Copied") {
		t.Errorf("StatusLine() = %q, want the confirmation gone after copiedConfirmationWindow has passed", got)
	}
}

// TestResponseView_StatusLineAlwaysHintsAtCopy guards discoverability - a
// user found "y" to copy purely by asking, not because the app told them.
func TestResponseView_StatusLineAlwaysHintsAtCopy(t *testing.T) {
	rv := newResponseView()
	rv.SetSize(60, 20)
	rv.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK", Body: []byte(`{"a":1}`)}, 10)

	if got := rv.StatusLine(); !strings.Contains(got, "y copy") {
		t.Errorf("StatusLine() = %q, want a \"y copy\" hint", got)
	}
}

// TestResponseView_StatusLineHintsAtScrollOnlyWhenContentOverflows guards
// against advertising "h/l scroll" when there's nothing further right to
// see - that would read as a live control that silently does nothing.
func TestResponseView_StatusLineHintsAtScrollOnlyWhenContentOverflows(t *testing.T) {
	rv := newResponseView()
	// 70, not 60: wide enough for statusPart + the scroll hint to both fit
	// with the "  |  " segment separators (see StatusLine) - narrower and
	// the hint-dropping fallback for an overflowing line kicks in before
	// this test ever gets to check the hint itself, which is a width
	// budget quirk of the separator style, not what this test guards.
	rv.SetSize(70, 20)
	rv.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK", Body: []byte(`{"a":1}`)}, 10)
	if got := rv.StatusLine(); strings.Contains(got, "scroll") {
		t.Errorf("StatusLine() = %q, want no scroll hint when the content already fits", got)
	}

	longValue := strings.Repeat("x", 200)
	rv.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK", Body: []byte(`{"value":"` + longValue + `"}`)}, 10)
	if got := rv.StatusLine(); !strings.Contains(got, "scroll") {
		t.Errorf("StatusLine() = %q, want a scroll hint once a line is wider than the panel", got)
	}
}

// TestNewResponseView_EnablesHorizontalScrolling guards the actual
// mechanism: bubbles/viewport ships with horizontal scrolling disabled by
// default, so pressing the left/right (or h/l) keys used to silently do
// nothing at all.
func TestNewResponseView_EnablesHorizontalScrolling(t *testing.T) {
	rv := newResponseView()
	rv.SetSize(10, 20)
	longValue := strings.Repeat("x", 200)
	rv.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK", Body: []byte(`{"value":"` + longValue + `"}`)}, 10)
	before := rv.vp.HorizontalScrollPercent()

	rv, _ = rv.Update(tea.KeyMsg{Type: tea.KeyRight})

	if after := rv.vp.HorizontalScrollPercent(); after <= before {
		t.Errorf("HorizontalScrollPercent() = %v after pressing right, want it to have advanced past %v", after, before)
	}
}

// TestModel_YKeyCopiesResponseWhileResponseFocused is the end-to-end guard
// that "y" actually reaches responseView.Update through the full key
// dispatch chain (root.go's handleKey -> dispatchKeyToFocusedWidget), not
// just when calling responseView.Update directly.
func TestModel_YKeyCopiesResponseWhileResponseFocused(t *testing.T) {
	copied := stubClipboard(t)
	m := New(t.TempDir(), t.TempDir())
	m.screen = ScreenRequest
	m.focus = focusResponse
	m.response.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK", Body: []byte(`{"a":1}`)}, 10)

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	after := next.(Model)

	if *copied == "" {
		t.Fatal("expected pressing y while the Response panel is focused to copy its content")
	}
	if after.response.copiedAt.IsZero() {
		t.Error("expected the copy confirmation state to be set on the model's response")
	}
}

// TestResponseView_CapitalYCopiesOnlyTheCurrentLine guards the follow-up
// ask: a user with one long field (an auth token) buried in an otherwise
// normal-sized response wanted a way to copy just that value, not the
// entire body every time.
func TestResponseView_CapitalYCopiesOnlyTheCurrentLine(t *testing.T) {
	copied := stubClipboard(t)
	rv := newResponseView()
	rv.SetSize(60, 2) // shorter than the content, so YOffset can move off 0
	rv.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK", Body: []byte("{\n  \"a\": 1,\n  \"b\": 2\n}")}, 10)
	rv.vp.SetYOffset(1) // scroll so line 1 ("  \"a\": 1,") is at the top

	rv, _ = rv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}})

	if !strings.Contains(*copied, `"a": 1`) {
		t.Errorf("got copied text %q, want just the line scrolled to the top", *copied)
	}
	if strings.Contains(*copied, `"b": 2`) {
		t.Errorf("got copied text %q, want only the current line, not the whole body", *copied)
	}
	if !rv.copiedLineOnly {
		t.Error("expected Y to mark this as a line-only copy, for StatusLine's confirmation wording")
	}
}

// TestResponseView_CurrentLineTextStripsANSI guards the same "plain text,
// not raw escape sequences" requirement the whole-body copy already has.
func TestResponseView_CurrentLineTextStripsANSI(t *testing.T) {
	rv := newResponseView()
	rv.SetSize(60, 20)
	rv.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK", Body: []byte(`{"a":1}`)}, 10)

	got := rv.CurrentLineText()

	if strings.Contains(got, "\x1b") {
		t.Errorf("CurrentLineText() = %q, contains a raw ANSI escape byte", got)
	}
}

// TestResponseView_StatusLineDistinguishesLineCopyFromFullCopy guards the
// wording difference between the two confirmations - "Copied to
// clipboard" after y (everything) vs "Copied line to clipboard" after Y
// (just the current line), so a user can tell which one just happened.
func TestResponseView_StatusLineDistinguishesLineCopyFromFullCopy(t *testing.T) {
	rv := newResponseView()
	rv.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK"}, 10)

	rv.copiedAt = time.Now()
	rv.copiedLineOnly = true
	if got := rv.StatusLine(); !strings.Contains(got, "line") {
		t.Errorf("StatusLine() = %q, want it to mention \"line\" after a Y (line-only) copy", got)
	}

	rv.copiedLineOnly = false
	if got := rv.StatusLine(); strings.Contains(got, "line") {
		t.Errorf("StatusLine() = %q, want no \"line\" mention after a y (whole-body) copy", got)
	}
}

// TestResponseView_StatusLineHintsAtLineCopy guards discoverability for the
// new shortcut, same as the existing "y copy" hint.
func TestResponseView_StatusLineHintsAtLineCopy(t *testing.T) {
	rv := newResponseView()
	rv.SetSize(60, 20)
	rv.SetResponse(execution.Response{StatusCode: 200, Status: "200 OK", Body: []byte(`{"a":1}`)}, 10)

	if got := rv.StatusLine(); !strings.Contains(got, "Y line") {
		t.Errorf("StatusLine() = %q, want a \"Y line\" hint", got)
	}
}

// TestResponseView_CookiesTabMasksValuesUnlessRevealed guards that the Cookies
// tab hides cookie values (session identifiers/tokens) by default, revealing
// them only when the global reveal toggle is on. Non-secret attributes
// (Path/Domain) stay visible either way.
func TestResponseView_CookiesTabMasksValuesUnlessRevealed(t *testing.T) {
	rv := newResponseView()
	rv.headers = []collection.Header{
		{Key: "Set-Cookie", Value: "session=secretvalue123; Path=/; HttpOnly"},
	}

	masked := stripANSI(rv.cookiesText())
	if strings.Contains(masked, "secretvalue123") {
		t.Errorf("cookie value must be masked by default, got:\n%s", masked)
	}
	if !strings.Contains(masked, "session") {
		t.Errorf("cookie name should still show, got:\n%s", masked)
	}
	if !strings.Contains(masked, "Path: /") {
		t.Errorf("non-secret cookie attributes should still show, got:\n%s", masked)
	}

	rv.revealSecrets = true
	shown := stripANSI(rv.cookiesText())
	if !strings.Contains(shown, "secretvalue123") {
		t.Errorf("reveal should show the cookie value, got:\n%s", shown)
	}
}
