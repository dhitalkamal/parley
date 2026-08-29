package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// tabBarSep is the spacing between tab labels (no dividers, underline style) -
// shared by tabBarWithUnderline (rendering) and tabLabelAt (hit-testing) so the
// two can't drift.
const tabBarSep = "   "

// tabBarWithUnderline renders a two-line tab strip shared by the Request
// panel's Params/Headers/Body/Auth/Scripts/Settings tabs, the Response panel's
// Body/Headers/Cookies/Tests/Timeline tabs, and the Body tab's own type tabs.
// Underline style: the labels spaced out (active accent/bold, inactive dim), no
// dividers, and a short accent underline sitting under just the active label on
// the line below - no full-width rule. A user picked this over the busier
// DevTools bar (filled pill + vertical dividers + full-width rule).
//
// Both lines are clipped to width - a real regression this guards against:
// adding a 6th Request-panel tab made the un-clipped label line wider than the
// panel at common widths, and the caller's own style.Width() call word-wrapped
// it instead of clipping, silently adding a row the height budget never
// accounted for. That single extra row cascaded into the whole 3-panel grid
// rendering taller than the terminal, scrolling the top bar off-screen - the
// same "pre-clip before it ever reaches a width-based style" fix already
// applied to kvtable.go/response.go/body_binary.go.
func tabBarWithUnderline(labels []string, activeIdx, width int) string {
	var line strings.Builder
	col, underAt, underLen := 0, 0, 0
	for i, label := range labels {
		if i > 0 {
			line.WriteString(tabBarSep)
			col += len(tabBarSep)
		}
		if i == activeIdx {
			underAt = col
			underLen = len(label)
			line.WriteString(activeTabStyle.Render(label))
		} else {
			line.WriteString(tabBarStyle.Render(label))
		}
		col += len(label)
	}
	labelLine := line.String()
	// the underline: leading spaces up to the active label, then an accent rule
	// exactly as wide as that label (not the whole bar).
	under := strings.Repeat(" ", underAt) + accentStyle.Render(strings.Repeat(glyphHorizontalLine, underLen))

	if width > 0 {
		labelLine = ansi.Cut(labelLine, 0, width)
		under = ansi.Cut(under, 0, width)
	}
	return labelLine + "\n" + under
}

// clipToLines truncates s to at most n lines - the height counterpart to
// clipLine's per-line width clipping, for sections whose natural content
// height depends on data (row counts, etc.) and can exceed the fixed budget
// it's rendered into. lipgloss's own Style.Height() only pads shorter
// content, it never truncates taller content, so anything variable-height
// needs this before it reaches a Height()-constrained style.
func clipToLines(s string, n int) string {
	if n < 0 {
		n = 0
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[:n], "\n")
}

// clipLine hard-clips a rendered line to width (ANSI-aware), so long hint
// text clips at the panel edge instead of word-wrapping onto a second row
// the height budget never allowed for - the same "clip, don't wrap" rule the
// tab bars and response status line follow. width 0 means "unmeasured".
func clipLine(s string, width int) string {
	if width > 0 {
		return ansi.Cut(s, 0, width)
	}
	return s
}

// tabLabelAt maps a click column to which label it landed on, mirroring
// tabBarWithUnderline's own layout exactly: each label occupies its own text
// width, tabs separated by tabBarSep spaces (the gaps are dead zones).
func tabLabelAt(labels []string, x int) (int, bool) {
	col := 0
	for i, label := range labels {
		if i > 0 {
			col += len(tabBarSep)
		}
		if x >= col && x < col+len(label) {
			return i, true
		}
		col += len(label)
	}
	return 0, false
}
