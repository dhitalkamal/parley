package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Visual mode for the Response body: v anchors a line selection at the current
// line (the top of the viewport), j/k extend it, y copies the selected lines,
// esc cancels. The "current line" is vp.YOffset, so the anchor plus YOffset
// define the range.

// visualSelStyle highlights the selected line range.
var visualSelStyle = lipgloss.NewStyle().Reverse(true)

// visualContent returns the body content with the selected line range reverse-
// highlighted while Visual mode is active, or the content unchanged otherwise.
func (rv responseView) visualContent(content string) string {
	if !rv.visualActive || len(rv.renderedLines) == 0 {
		return content
	}
	lo, hi := rv.visualRange()
	lines := make([]string, len(rv.renderedLines))
	copy(lines, rv.renderedLines)
	for i := lo; i <= hi && i < len(lines); i++ {
		lines[i] = visualSelStyle.Render(ansi.Strip(lines[i]))
	}
	return strings.Join(lines, "\n")
}

// refreshVisualOverlay repaints only the Visual selection highlight from the
// already-rendered lines, skipping refresh()'s expensive body formatting
// (DetectAndRender + syntax highlight + secret masking + line split). While
// Visual mode is active the rendered body never changes across scrolls - only
// the selected range moves - and visualContent rebuilds purely from
// rv.renderedLines when active (its content argument is ignored), so passing an
// empty string here is safe. Callers must only use this while visualActive.
func (rv *responseView) refreshVisualOverlay() {
	rv.vp.SetContent(rv.visualContent(""))
}

// visualRange is the selected line range [lo, hi], clamped to the content.
func (rv responseView) visualRange() (int, int) {
	lo, hi := rv.visualAnchor, rv.vp.YOffset
	if lo > hi {
		lo, hi = hi, lo
	}
	if lo < 0 {
		lo = 0
	}
	if hi >= len(rv.renderedLines) {
		hi = len(rv.renderedLines) - 1
	}
	return lo, hi
}

// toggleVisual turns Visual mode on (anchoring at the current line) or off.
func (rv *responseView) toggleVisual() {
	rv.visualActive = !rv.visualActive
	if rv.visualActive {
		rv.visualAnchor = rv.vp.YOffset
	}
	rv.refresh()
}

// exitVisual leaves Visual mode without copying.
func (rv *responseView) exitVisual() {
	if !rv.visualActive {
		return
	}
	rv.visualActive = false
	rv.refresh()
}

// visualCopy copies the selected line range to the clipboard and exits Visual.
func (rv *responseView) visualCopy() {
	lo, hi := rv.visualRange()
	if lo <= hi && lo < len(rv.renderedLines) {
		_ = writeClipboard(ansi.Strip(strings.Join(rv.renderedLines[lo:hi+1], "\n")))
		rv.copiedAt = time.Now()
		rv.copiedLineOnly = false
	}
	rv.visualActive = false
	rv.refresh()
}
