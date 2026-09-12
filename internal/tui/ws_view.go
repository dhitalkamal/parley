package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const (
	wsConnectBoxH   = 3  // bordered address bar
	wsComposerBoxH  = 3  // bordered message box
	wsHintH         = 1  // shared bottom hint
	wsMinSideBySide = 90 // below this width the two panes stack vertically instead
)

// wsSideBySide reports whether the two panes render left/right (wide terminals)
// or stacked top/bottom (narrow ones).
func (m Model) wsSideBySide() bool { return m.width >= wsMinSideBySide }

// wsPaneGeom returns the outer width/height of each pane for the current
// terminal and orientation - shared by resizeWS (sizing widgets) and wsView
// (drawing boxes) so the two never disagree.
func (m Model) wsPaneGeom() (w0, h0, w1, h1 int) {
	contentH := m.height - wsHintH
	if contentH < 6 {
		contentH = 6
	}
	if m.wsSideBySide() {
		w0 = m.width / 2
		return w0, contentH, m.width - w0, contentH
	}
	h0 = contentH / 2
	return m.width, h0, m.width, contentH - h0
}

// resizeWS sizes both panes' widgets to the current terminal, then refreshes
// their transcripts so wrapping and scroll stay correct. Called on
// WindowSizeMsg and at startup.
func (m *Model) resizeWS() {
	w0, h0, w1, h1 := m.wsPaneGeom()
	m.sizePane(0, w0, h0)
	m.sizePane(1, w1, h1)
	m.refreshWSPane(0)
	m.refreshWSPane(1)
}

func (m *Model) sizePane(p, outerW, outerH int) {
	innerW := outerW - 4
	if innerW < 10 {
		innerW = 10
	}
	pane := &m.ws.panes[p]
	pane.urlInput.Width = clampMin(innerW-18, 6)
	pane.composer.Width = clampMin(innerW-10, 6)
	pane.headers.SetWidth(innerW)
	middleInnerH := outerH - wsConnectBoxH - wsComposerBoxH - 2
	if middleInnerH < 1 {
		middleInnerH = 1
	}
	pane.vp.Width = innerW
	pane.vp.Height = middleInnerH
	pane.headers.SetHeight(middleInnerH)
}

func clampMin(v, min int) int {
	if v < min {
		return min
	}
	return v
}

// refreshWSPane rebuilds pane p's transcript content and scrolls it: to keep
// the selected message in view while the transcript is focused, otherwise to
// the newest line.
func (m *Model) refreshWSPane(p int) {
	pane := &m.ws.panes[p]
	focusedTranscript := m.ws.focusPane == p && pane.focus == wsFocusTranscript
	content, selStart, selEnd := wsBuildTranscript(pane, focusedTranscript)
	pane.vp.SetContent(content)
	if focusedTranscript && len(pane.transcript) > 0 {
		wsScrollTo(&pane.vp, selStart, selEnd)
	} else {
		pane.vp.GotoBottom()
	}
}

// wsScrollTo nudges the viewport so the [start,end] line range is visible.
func wsScrollTo(vp *viewport.Model, start, end int) {
	top := vp.YOffset
	h := vp.Height
	switch {
	case start < top:
		vp.SetYOffset(start)
	case end >= top+h:
		vp.SetYOffset(end - h + 1)
	}
}

// wsBuildTranscript returns the rendered transcript plus the first/last
// display-line of the selected message (for scrolling). Each message is one
// block (possibly multi-line when expanded/wrapped). Blocks are cached on the
// pane; only entries whose inputs changed since the last build are re-rendered,
// so a refresh costs work proportional to what changed, not to the backlog.
func wsBuildTranscript(pane *wsSession, focused bool) (content string, selStart, selEnd int) {
	width := pane.vp.Width
	if width < 10 {
		width = 10
	}
	if len(pane.transcript) == 0 {
		return labelStyle.Render("Not connected. Enter a ws:// or wss:// URL above and press enter."), 0, 0
	}
	wsSyncBlockCache(pane, width, focused)
	cur := 0
	for i, blk := range pane.blockCache {
		if i == pane.selected {
			selStart = cur
			selEnd = cur + strings.Count(blk, "\n")
		}
		cur += strings.Count(blk, "\n") + 1
	}
	return strings.Join(pane.blockCache, "\n"), selStart, selEnd
}

// wsSyncBlockCache brings pane.blockCache up to date for the given width and
// focus, re-rendering only what changed. A width or focus flip changes every
// block (wrap width, or whether gutters/markers are drawn) so the whole cache
// is rebuilt; otherwise the only blocks that can change without a new append are
// the selected one (its expand state may have toggled) and the previously
// selected one (its marker must be dropped).
func wsSyncBlockCache(pane *wsSession, width int, focused bool) {
	if width != pane.cacheWidth || focused != pane.cacheFocused {
		pane.blockCache = pane.blockCache[:0]
		pane.cacheWidth = width
		pane.cacheFocused = focused
		pane.cacheSel = -1
	}
	// render entries appended since the last build.
	for i := len(pane.blockCache); i < len(pane.transcript); i++ {
		sel := focused && i == pane.selected
		pane.blockCache = append(pane.blockCache, wsEventBlock(pane.transcript[i], width, pane.expanded[i], sel))
	}
	// defensive: the transcript only grows today, but keep the cache in bounds.
	if len(pane.blockCache) > len(pane.transcript) {
		pane.blockCache = pane.blockCache[:len(pane.transcript)]
	}
	if !focused {
		return
	}
	if pane.cacheSel != pane.selected && pane.cacheSel >= 0 && pane.cacheSel < len(pane.blockCache) {
		pane.blockCache[pane.cacheSel] = wsEventBlock(pane.transcript[pane.cacheSel], width, pane.expanded[pane.cacheSel], false)
	}
	if pane.selected >= 0 && pane.selected < len(pane.blockCache) {
		pane.blockCache[pane.selected] = wsEventBlock(pane.transcript[pane.selected], width, pane.expanded[pane.selected], true)
	}
	pane.cacheSel = pane.selected
}

// wsEventBlock formats one transcript entry, soft-wrapped to width. A selected
// entry gets a "> " gutter; binary frames get a "(bin)" tag; text frames are
// JSON-pretty-printed when this message is expanded.
func wsEventBlock(e wsEvent, width int, expanded, selected bool) string {
	gutter := "  "
	if selected {
		gutter = "> "
	}
	ts := e.at.Format("15:04:05") + " "
	var marker, body string
	switch e.dir {
	case wsSent:
		marker = accentStyle.Render(">")
		body = e.text
	case wsRecv:
		marker = statusStyle.Render("<")
		body = e.text
	default: // wsStatus
		marker = labelStyle.Render("*")
		body = labelStyle.Render(e.text)
	}
	if e.binary {
		body = labelStyle.Render("(bin) ") + body
	} else if expanded && e.dir != wsStatus {
		body = prettyJSONMaybe(e.text)
	}
	line := labelStyle.Render(gutter+ts) + marker + " " + body
	return lipgloss.NewStyle().Width(width).Render(line)
}

// prettyJSONMaybe indents text if it's valid JSON, otherwise returns it as-is.
func prettyJSONMaybe(text string) string {
	trimmed := strings.TrimSpace(text)
	if !json.Valid([]byte(trimmed)) {
		return text
	}
	var out bytes.Buffer
	if err := json.Indent(&out, []byte(trimmed), "", "  "); err != nil {
		return text
	}
	return out.String()
}

// wsBinaryPreview describes a received binary frame: byte length plus a hex dump
// of the first bytes.
func wsBinaryPreview(data []byte) string {
	const max = 24
	show := data
	truncated := false
	if len(show) > max {
		show, truncated = show[:max], true
	}
	s := fmt.Sprintf("%d bytes: %s", len(data), encodeHexSpaced(show))
	if truncated {
		s += " ..."
	}
	return s
}

// wsView renders the whole WebSocket screen: two panes (side by side, or
// stacked on a narrow terminal) over a shared key-hint line.
func (m Model) wsView() string {
	w0, h0, w1, h1 := m.wsPaneGeom()
	var body string
	if m.wsSideBySide() {
		body = lipgloss.JoinHorizontal(lipgloss.Top, m.wsPaneView(0, w0, h0), m.wsPaneView(1, w1, h1))
	} else {
		body = lipgloss.JoinVertical(lipgloss.Left, m.wsPaneView(0, w0, h0), m.wsPaneView(1, w1, h1))
	}
	return padLinesTo(lipgloss.JoinVertical(lipgloss.Left, body, m.wsHint()), m.height)
}

// wsPaneView draws one pane: address/connect bar, a middle box (transcript, or
// the headers editor while it's focused), and a composer.
func (m Model) wsPaneView(p, outerW, outerH int) string {
	pane := &m.ws.panes[p]
	active := p == m.ws.focusPane
	label := wsPaneLabel(p)

	connectBox := wsBoxTitled(m.wsConnectBar(p, outerW-4), outerW, wsConnectBoxH,
		"WS "+label+"  "+m.wsConnState(p), active && pane.focus == wsFocusURL)

	middleH := outerH - wsConnectBoxH - wsComposerBoxH
	var middle string
	if active && pane.focus == wsFocusHeaders {
		middle = wsBoxTitled(pane.headers.View(), outerW, middleH, "Headers "+label+"  (tab away for transcript)", true)
	} else {
		middle = wsBoxTitled(pane.vp.View(), outerW, middleH, m.wsTranscriptTitle(p),
			active && pane.focus == wsFocusTranscript)
	}

	composerBox := wsBoxTitled(pane.composer.View(), outerW, wsComposerBoxH,
		m.wsComposerTitle(p), active && pane.focus == wsFocusComposer)

	return lipgloss.JoinVertical(lipgloss.Left, connectBox, middle, composerBox)
}

func wsPaneLabel(p int) string {
	if p == 0 {
		return "A"
	}
	return "B"
}

// wsTranscriptTitle names the middle box with a select/expand hint while the
// pane's transcript is focused.
func (m Model) wsTranscriptTitle(p int) string {
	base := "Transcript " + wsPaneLabel(p)
	if m.ws.focusPane == p && m.ws.panes[p].focus == wsFocusTranscript {
		return base + "  " + labelStyle.Render("up/down select, enter expand")
	}
	return base
}

// wsConnState is the connection indicator shown on the address box title.
func (m Model) wsConnState(p int) string {
	pane := m.ws.panes[p]
	switch {
	case pane.connected:
		return statusStyle.Render("connected " + wsOpenFor(pane.openedAt))
	case pane.connecting:
		return labelStyle.Render("connecting...")
	default:
		return labelStyle.Render("idle")
	}
}

// wsConnectBar is the inner line of the address box: the URL editor plus a
// Connect/Disconnect state token.
func (m Model) wsConnectBar(p, innerW int) string {
	pane := m.ws.panes[p]
	var token string
	switch {
	case pane.connected:
		token = statusStyle.Render("[esc disconnect]")
	case pane.connecting:
		token = labelStyle.Render("[connecting...]")
	default:
		token = accentStyle.Render("[enter Connect]")
	}
	left := pane.urlInput.View()
	pad := innerW - lipgloss.Width(left) - lipgloss.Width(token)
	if pad < 1 {
		pad = 1
	}
	return left + strings.Repeat(" ", pad) + token
}

// wsComposerTitle names the message box, calling out the send mode.
func (m Model) wsComposerTitle(p int) string {
	label := "Message " + wsPaneLabel(p)
	mode := wsSendModeLabel(m.ws.panes[p].sendMode)
	if m.ws.panes[p].sendMode == wsSendText {
		return label + "  (" + mode + ")"
	}
	return label + "  " + accentStyle.Render("["+mode+"]")
}

func wsSendModeLabel(mode wsSendMode) string {
	switch mode {
	case wsSendHex:
		return "binary hex"
	case wsSendBytes:
		return "binary bytes"
	default:
		return "text"
	}
}

// wsHint is the shared bottom line: keys plus the active pane's send mode and
// what esc does right now (close the headers editor, disconnect, or - with
// nothing to back out of - arm quit on a second press).
func (m Model) wsHint() string {
	pane := m.ws.active()
	esc := "esc esc quit"
	switch {
	case pane.focus == wsFocusHeaders:
		esc = "esc transcript"
	case pane.isLive():
		esc = "esc disconnect"
	}
	return labelStyle.Render("tab move/switch pane  enter connect/send  ctrl+b " + wsSendModeLabel(pane.sendMode) +
		"  ctrl+g ping  ctrl+s save  " + esc)
}

// wsOpenFor is the elapsed connection time as m:ss.
func wsOpenFor(since time.Time) string {
	d := time.Since(since)
	if d < 0 {
		d = 0
	}
	total := int(d.Seconds())
	return fmt.Sprintf("%d:%02d", total/60, total%60)
}

// wsBoxTitled renders content in a bordered box of the given outer size, with a
// floating title and focused-vs-not border color. Content is clipped to the
// exact inner width/height first: lipgloss's Height only sets a minimum, so a
// widget that renders taller than its box (the headers editor did) would
// otherwise overflow and shove the panes out of alignment.
func wsBoxTitled(content string, outerW, outerH int, title string, focused bool) string {
	border := borderStyle
	if focused {
		border = focusedBorder
	}
	innerH := outerH - 2
	if innerH < 1 {
		innerH = 1
	}
	innerW := outerW - 4 // border (2) + horizontal padding (2)
	if innerW < 1 {
		innerW = 1
	}
	body := border.Width(outerW - 2).Height(innerH).Render(clipToBox(content, innerW, innerH))
	return titledBox(body, title)
}

// clipToBox trims content to at most height lines, each at most width display
// columns (ANSI-aware), so a bordered box renders at exactly its declared size.
func clipToBox(content string, width, height int) string {
	lines := strings.Split(content, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for i, l := range lines {
		if lipgloss.Width(l) > width {
			lines[i] = ansi.Cut(l, 0, width)
		}
	}
	return strings.Join(lines, "\n")
}
