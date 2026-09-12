package tui

import (
	"fmt"
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	scripting "github.com/dhitalkamal/parley/internal/scripting/domain"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// writeClipboard is a seam over clipboard.WriteAll so tests can substitute
// a fake instead of actually touching the real system clipboard.
var writeClipboard = clipboard.WriteAll

type responseViewMode int

const (
	viewBody responseViewMode = iota
	viewHeaders
	viewCookies
	viewTests
	viewTimeline
	viewModeCount
)

func (m responseViewMode) String() string {
	switch m {
	case viewHeaders:
		return "headers"
	case viewCookies:
		return "cookies"
	case viewTests:
		return "tests"
	case viewTimeline:
		return "timeline"
	default:
		return "body"
	}
}

// responseModeLabels is the Response panel's tab bar text, in mode order -
// shared by TabBarText (rendering) and mouse.go's responseModeTabAt
// (hit-testing), so the two can never drift out of sync with each other.
func responseModeLabels() []string {
	return []string{"Body", "Headers", "Cookies", "Tests", "Timeline"}
}

var searchHighlightStyle = lipgloss.NewStyle().Background(lipgloss.Color("226")).Foreground(lipgloss.Color("0"))

type responseView struct {
	vp            viewport.Model
	status        string
	statusCode    int
	elapsedMS     int64
	sizeBytes     int
	rawBody       []byte
	contentType   string
	headers       []collection.Header
	timing        execution.Timing
	mode          responseViewMode
	searchTerm    string
	renderedLines []string
	err           error

	// testResults/testScriptError back the Tests mode - set separately from
	// SetResponse because the test script (if any) runs, and finishes, after
	// the response itself has already arrived and been shown.
	testResults     []scripting.TestResult
	testScriptError string
	// copiedAt drives StatusLine's transient "Copied..." confirmation -
	// cleared implicitly by just going stale (see StatusLine), not by a
	// timer, since the app already repaints at least once a second
	// regardless (the top bar's own clock tick). copiedLineOnly picks
	// between the two confirmation wordings - "y" (whole view) vs "Y"
	// (just the current line - see CurrentLineText).
	copiedAt       time.Time
	copiedLineOnly bool

	// revealSecrets controls whether secret-looking body fields (see
	// secretmask.go) render masked or literal - toggled globally by ctrl+u.
	revealSecrets bool

	// visualActive/visualAnchor back vim Visual line-select: v anchors at the
	// current line (the top of the viewport), j/k extend the range, y copies the
	// selected lines. The current line is vp.YOffset, so the anchor plus YOffset
	// define the selection.
	visualActive bool
	visualAnchor int
}

// SetRevealSecrets toggles masking of secret-looking body fields and
// re-renders the current mode immediately so the change is visible without
// needing to switch tabs.
func (rv *responseView) SetRevealSecrets(reveal bool) {
	rv.revealSecrets = reveal
	rv.refresh()
}

func newResponseView() responseView {
	vp := viewport.New(0, 0)
	// Disabled by default in bubbles/viewport v1 - a long unbroken value
	// (an auth token, say) has nowhere to linebreak and used to just run
	// off the edge of the panel with no way to see the rest of it.
	vp.SetHorizontalStep(10)
	return responseView{vp: vp}
}

func (rv *responseView) SetSize(w, h int) {
	rv.vp.Width = w
	rv.vp.Height = h
}

// SetError shows a failed request's error, optionally followed by a plain-
// language hint (see networkErrorHint) that classifies a network failure and
// notes the current VPN state.
func (rv *responseView) SetError(err error, hint string) {
	rv.err = err
	rv.status = ""
	content := errStyle.Render(err.Error())
	if hint != "" {
		content += "\n\n" + labelStyle.Render(hint)
	}
	rv.vp.SetContent(content)
	rv.vp.GotoTop()
}

func (rv *responseView) SetResponse(resp execution.Response, elapsedMS int64) {
	rv.err = nil
	rv.statusCode = resp.StatusCode
	rv.status = resp.Status
	rv.elapsedMS = elapsedMS
	rv.sizeBytes = len(resp.Body)
	rv.rawBody = resp.Body
	rv.headers = resp.Headers
	rv.timing = resp.Timing
	rv.contentType = headerValue(resp.Headers, "Content-Type")
	rv.mode = viewBody
	rv.searchTerm = ""
	rv.refresh()
	rv.vp.GotoTop()
}

// SetTestResults records the outcome of the request's test script (if any)
// for the Tests mode - called once the script finishes running, separately
// from SetResponse, since that happens after the response itself already
// arrived. A fresh send clears both to nil/"" before its own result comes
// back, so a new request never shows the previous one's leftover results.
func (rv *responseView) SetTestResults(results []scripting.TestResult, scriptErr string) {
	rv.testResults = results
	rv.testScriptError = scriptErr
	rv.refresh()
}

func headerValue(headers []collection.Header, key string) string {
	for _, h := range headers {
		if strings.EqualFold(h.Key, key) {
			return h.Value
		}
	}
	return ""
}

func (rv *responseView) CycleMode() {
	rv.CycleModeBy(1)
}

// CycleModeBy advances the response tab by delta, wrapping in both directions -
// backs the focus-aware shift+left/shift+right nav (see handlekey.go), the
// Response mirror of the Request panel's own left/right tab movement. The
// double-modulo keeps a negative delta (shift+left from Body) in range instead
// of producing a negative mode.
func (rv *responseView) CycleModeBy(delta int) {
	n := int(viewModeCount)
	rv.mode = responseViewMode(((int(rv.mode)+delta)%n + n) % n)
	rv.refresh()
	rv.vp.GotoTop()
}

// SetMode jumps straight to mode - used by a mouse click on a specific
// response tab, where CycleMode's "advance by one" doesn't apply.
func (rv *responseView) SetMode(mode responseViewMode) {
	rv.mode = mode
	rv.refresh()
	rv.vp.GotoTop()
}

// SetSearch highlights every case-insensitive occurrence of term in the
// current body view and scrolls to the first one.
func (rv *responseView) SetSearch(term string) {
	rv.searchTerm = term
	rv.refresh()
	rv.scrollToFirstMatch(term)
}

func (rv *responseView) refresh() {
	var content string
	switch rv.mode {
	case viewBody:
		// Always fully formatted - a user asked for the response to no
		// longer have a pretty/raw choice at all (that control now lives on
		// the request body instead, where it's actually being typed rather
		// than just viewed - see body.go's ctrl+p).
		content = DetectAndRender(rv.rawBody, rv.contentType, true, rv.revealSecrets)
		content = highlightSearch(content, rv.searchTerm)
	case viewHeaders:
		content = formatHeaders(rv.headers, rv.revealSecrets)
	case viewCookies:
		content = rv.cookiesText()
	case viewTests:
		content = rv.testsText()
	case viewTimeline:
		content = rv.timelineText()
	}
	rv.renderedLines = strings.Split(content, "\n")
	rv.vp.SetContent(rv.visualContent(content))
}

func formatHeaders(headers []collection.Header, reveal bool) string {
	if len(headers) == 0 {
		return labelStyle.Render("(no headers)")
	}
	var b strings.Builder
	for _, h := range headers {
		// mask secret-looking header values (Authorization, Set-Cookie,
		// x-api-key, ...) unless the user toggled reveal - the headers tab
		// used to print every value verbatim, exposing them on screen-share.
		fmt.Fprintf(&b, "%s: %s\n", h.Key, maskedValue(h.Key, h.Value, reveal))
	}
	return strings.TrimRight(b.String(), "\n")
}

func (rv responseView) cookiesText() string {
	var cookies []execution.Cookie
	for _, h := range rv.headers {
		if strings.EqualFold(h.Key, "Set-Cookie") {
			cookies = append(cookies, execution.ParseSetCookie(h.Value))
		}
	}
	if len(cookies) == 0 {
		return labelStyle.Render("(no cookies)")
	}
	var b strings.Builder
	for _, c := range cookies {
		fmt.Fprintf(&b, "%s = %s\n", c.Name, c.Value)
		if c.Domain != "" {
			fmt.Fprintf(&b, "  Domain: %s\n", c.Domain)
		}
		if c.Path != "" {
			fmt.Fprintf(&b, "  Path: %s\n", c.Path)
		}
		if c.Expires != "" {
			fmt.Fprintf(&b, "  Expires: %s\n", c.Expires)
		}
		if c.Secure {
			b.WriteString("  Secure\n")
		}
		if c.HTTPOnly {
			b.WriteString("  HttpOnly\n")
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func (rv *responseView) scrollToFirstMatch(term string) {
	if term == "" {
		return
	}
	lowerTerm := strings.ToLower(term)
	for i, line := range rv.renderedLines {
		if strings.Contains(strings.ToLower(line), lowerTerm) {
			rv.vp.SetYOffset(i)
			return
		}
	}
}

// copiedConfirmationWindow is how long StatusLine keeps showing its
// "Copied..." confirmation after a copy - it fades back to the normal
// status line on its own (the app already repaints at least once a second
// regardless, via the top bar's own clock tick), not on any dedicated
// timer.
const copiedConfirmationWindow = 2 * time.Second

func (rv responseView) StatusLine() string {
	if rv.err != nil {
		return errStyle.Render("Error: " + rv.err.Error())
	}
	if !rv.copiedAt.IsZero() && time.Since(rv.copiedAt) < copiedConfirmationWindow {
		msg := "Copied to clipboard"
		if rv.copiedLineOnly {
			msg = "Copied line to clipboard"
		}
		return activeTabStyle.Render(msg)
	}
	if rv.status == "" {
		// Blank, not "Ready" - ContentView's own emptyStateView already
		// says "No response yet" front and center; repeating that one row
		// up as "Ready" read as two different, slightly conflicting
		// messages for the same "nothing sent" state.
		return ""
	}
	// "  |  " is the app's own established segment separator - a user found
	// the status/timing/size/hint run together with only double-spacing
	// between them too dense to scan at a glance, and this is the app's own
	// established way of visually separating several unrelated pieces of
	// information on one line.
	statusPart := statusClassStyle(rv.statusCode).Render(rv.status) +
		labelStyle.Render(fmt.Sprintf("  |  %d ms  |  %s", rv.elapsedMS, humanSize(rv.sizeBytes)))
	// "y copy"/"Y line" are always worth advertising once there's
	// something to copy; "h/l scroll" only when there's actually more of
	// the current view off to the right - a user hit a long unbroken
	// value (an auth token) that used to just run off the edge of the
	// panel with no way to see or grab the rest of it, and asked
	// specifically for a way to copy just that one line rather than the
	// whole body.
	hint := "y copy, Y line"
	if rv.vp.HorizontalScrollPercent() < 1.0 {
		hint = "h/l scroll, " + hint
	}
	full := statusPart + labelStyle.Render("  |  ("+hint+")")

	// clip, don't wrap: a long status (e.g. "500 Internal Server Error")
	// plus the hint used to overflow the panel width and word-wrap onto a
	// second row, stealing a line the height budget never allowed for. drop
	// the optional hint first, then hard-clip so even a long status alone
	// can't wrap. width 0 means "unmeasured" (unit tests) - leave it be.
	w := rv.vp.Width
	if w <= 0 {
		return full
	}
	if lipgloss.Width(full) > w {
		full = statusPart
	}
	return ansi.Cut(full, 0, w)
}

// CopyText is what y copies to the clipboard - whatever's currently
// displayed (Body/Headers/Cookies, following rv.mode), with syntax
// highlighting's ANSI codes stripped so the clipboard gets plain text
// rather than raw escape sequences mixed into the paste.
func (rv responseView) CopyText() string {
	return ansi.Strip(strings.Join(rv.renderedLines, "\n"))
}

// CurrentLineText is what Y copies - just the single line currently
// scrolled to the top of the viewport, ANSI-stripped, full width
// regardless of how far it's scrolled horizontally (see h/l scroll). A
// user with one long field (an auth token) buried in an otherwise short
// response asked for a way to grab just that value instead of the entire
// body every time.
func (rv responseView) CurrentLineText() string {
	i := rv.vp.YOffset
	if i < 0 || i >= len(rv.renderedLines) {
		return ""
	}
	return ansi.Strip(rv.renderedLines[i])
}

// humanSize formats a byte count the way the design reference does ("1.2
// KB") instead of an always-raw byte count that gets unreadable past a few
// KB.
func humanSize(n int) string {
	if n < 1024 {
		return fmt.Sprintf("%d bytes", n)
	}
	units := [...]string{"KB", "MB", "GB"}
	div, exp := 1024.0, 0
	// stop climbing units once we hit the largest one we know about, so exp
	// can never index past the end of the array for a >=1 TiB byte count.
	for m := n / 1024; m >= 1024 && exp < len(units)-1; m /= 1024 {
		div *= 1024
		exp++
	}
	return fmt.Sprintf("%.1f %s", float64(n)/div, units[exp])
}

// TabBarText renders the Response panel's mode indicator as a real tab bar
// (underlined active tab, matching reqTabBarText - see tabbar.go) instead of
// a plain text label. Used to also carry a Pretty/Raw format pill for Body,
// but the response is always fully formatted now (see refresh) - there's no
// choice left to show.
func (rv responseView) TabBarText() string {
	return tabBarWithUnderline(responseModeLabels(), int(rv.mode), rv.vp.Width)
}

// HasContent reports whether a response (or error) has actually come back
// yet - shared by ContentView (whether to show the empty state) and
// resppanel.go (whether the Body/Headers/Cookies tabs are worth showing at
// all): before anything's been sent, switching modes has nothing to switch
// between, so the tabs are a live control that would silently do nothing.
func (rv responseView) HasContent() bool {
	return rv.status != "" || rv.err != nil
}

// ContentView is the raw viewport content, borderless - see TabBarText. Before
// anything's been sent (or errored), it shows a centered explanation instead
// of the viewport's own content - a single line of hint text pinned to the
// top-left of a mostly-blank panel read as broken, not "empty and ready."
func (rv responseView) ContentView() string {
	if !rv.HasContent() {
		return rv.emptyStateView()
	}
	return rv.vp.View()
}

// emptyStateView centers a short explanation within the viewport's own
// dimensions, so it looks deliberately designed rather than a stray line of
// text floating over empty space. Every line is pre-wrapped to the
// viewport's actual width before centering - lipgloss.Place words-wraps
// (rather than clips) content wider than the box it's given, which would
// otherwise corrupt the layout on a narrow Response panel (the same class
// of bug this session already hit and fixed in kvtable.go/sidebar.go).
func (rv responseView) emptyStateView() string {
	width := rv.vp.Width
	if width < 10 {
		width = 10
	}
	title := labelStyle.Bold(true).Render(ansi.Wordwrap("No response yet", width, ""))
	body := labelStyle.Render(ansi.Wordwrap("Send a request to see its body, headers, and cookies", width, ""))
	hint := activeTabStyle.Render(ansi.Wordwrap("ctrl+r send", width, ""))
	msg := lipgloss.JoinVertical(lipgloss.Center, title, "", body, "", hint)
	return lipgloss.Place(rv.vp.Width, rv.vp.Height, lipgloss.Center, lipgloss.Center, msg)
}

// Update forwards scroll/navigation keys to the viewport - the Response
// panel's own focus target for reading a long body. "y" (vim-style yank)
// copies everything currently displayed; "Y" copies just the single line
// scrolled to the top (see CurrentLineText) - both intercepted first
// rather than falling through to the viewport, which doesn't bind either
// to anything.
func (rv responseView) Update(msg tea.Msg) (responseView, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "v": // toggle Visual line-select
			rv.toggleVisual()
			return rv, nil
		case "esc":
			if rv.visualActive {
				rv.exitVisual()
				return rv, nil
			}
		case "y":
			if rv.visualActive {
				rv.visualCopy()
				return rv, nil
			}
			_ = writeClipboard(rv.CopyText())
			rv.copiedAt = time.Now()
			rv.copiedLineOnly = false
			return rv, nil
		case "Y":
			_ = writeClipboard(rv.CurrentLineText())
			rv.copiedAt = time.Now()
			rv.copiedLineOnly = true
			return rv, nil
		}
	}
	var cmd tea.Cmd
	rv.vp, cmd = rv.vp.Update(msg)
	// While selecting, moving the cursor (scroll) extends the highlight.
	// Only the selection overlay moves - the rendered body is identical
	// across scrolls - so repaint just the overlay instead of re-running
	// refresh()'s full DetectAndRender/highlight/split over the whole body
	// on every j/k keystroke (that was per-keystroke lag on large bodies).
	if rv.visualActive {
		rv.refreshVisualOverlay()
	}
	return rv, cmd
}

// ScrollTop / ScrollBottom jump the response body to its first/last line - the
// NORMAL-mode g / G bindings.
func (rv *responseView) ScrollTop()    { rv.vp.GotoTop() }
func (rv *responseView) ScrollBottom() { rv.vp.GotoBottom() }
