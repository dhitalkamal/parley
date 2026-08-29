package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// titledBox splices title into row 0 (the top border) of an already-rendered
// bordered box, 2 columns in from the left corner - the floating-title-on-
// the-border look from the design reference, which lipgloss's own border API
// has no concept of. Reuses the overlay splice technique (see overlay.go)
// instead of a bespoke implementation.
func titledBox(box, title string) string {
	width := lipgloss.Width(box)
	height := lipgloss.Height(box)
	return overlayAt(box, " "+title+" ", width, height, 2, 0)
}

// titledBoxWithRight is titledBox, plus a second string spliced right-
// aligned on the same border row - the Request zone's body-type/content-
// type dropdown labels (see bodydropdown.go), which sit on the opposite
// side of the header from the title/chevron. Dropped entirely (falls back
// to plain titledBox) if there isn't room for both without overlapping -
// silently truncating the labels into something unreadable would be worse
// than not showing them at a width this narrow.
func titledBoxWithRight(box, title, right string) string {
	width := lipgloss.Width(box)
	height := lipgloss.Height(box)
	spliced := overlayAt(box, " "+title+" ", width, height, 2, 0)
	if right == "" {
		return spliced
	}
	rightX := rightTextX(width, title, right)
	if rightX < 0 {
		return spliced
	}
	return overlayAt(spliced, " "+right+" ", width, height, rightX, 0)
}

// rightTextX is titledBoxWithRight's own column math for where right
// starts, shared with mouse.go's click hit-testing so a click can never
// disagree with what's actually rendered. Returns -1 when there isn't
// enough room for both title and right without overlapping.
func rightTextX(width int, title, right string) int {
	rightX := width - lipgloss.Width(right) - 3
	if rightX < lipgloss.Width(title)+4 {
		return -1
	}
	return rightX
}

// panelTitleText builds a "[N] Name" panel title, bold and accent-colored
// when the panel is focused, dim otherwise - same visual language as the
// numbered [1]/[2]/[3] panel jump keys.
func panelTitleText(num int, name string, focused bool) string {
	text := fmt.Sprintf("[%d] %s", num, name)
	if focused {
		return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(activeTheme.FocusedBorder)).Render(text)
	}
	return labelStyle.Render(text)
}
