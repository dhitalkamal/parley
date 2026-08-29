package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// binaryFieldLabel prefixes the file-path input in binaryView - its width
// has to be subtracted from the box's inner width before sizing the input
// itself, or "label + input" renders wider than the box lipgloss.Place
// centers it in, overflowing past the border the same way an un-pre-
// wrapped hint would (see binaryView's doc comment).
const binaryFieldLabel = "File: "

// setBinarySize sizes the file-path input to fit inside binaryView's own
// bordered box (border+padding, plus the label prefixed to the input on
// the same line) - called from bodyEditor.SetSize.
func (b *bodyEditor) setBinarySize(w, h int) {
	boxWidth := w - 4
	if boxWidth < 10 {
		boxWidth = 10
	}
	inputWidth := boxWidth - lipgloss.Width(binaryFieldLabel)
	if inputWidth < 1 {
		inputWidth = 1
	}
	b.binaryPath.Width = inputWidth
	b.binaryBoxWidth = boxWidth
	b.binaryHeight = h
}

// binaryView shows a single file-path field, its own hint, and its own
// border (see rendersBorderless) - centered the same way kvTable's
// empty-state message is (see kvtable.go's emptyStateView), so a single
// input field in a tall panel reads as deliberately designed rather than
// pinned to the top-left corner with a large gap of blank space below it.
// The hint is pre-wrapped to the box's own known width before it ever
// reaches lipgloss.Place - Place word-wraps (rather than clips) content
// wider than the box it's given, which silently grows the rendered height
// past what requestPanelView budgeted and corrupts the whole panel's
// layout (the same class of bug kvtable.go's emptyStateView already
// works around).
func (b bodyEditor) binaryView() string {
	style := borderStyle
	if b.binaryPath.Focused() {
		style = focusedBorder
	}
	width := b.binaryBoxWidth
	height := b.binaryHeight
	if height < 3 {
		height = 3
	}
	wrapped := ansi.Wordwrap("path to the file to send as the raw request body", width, "")
	hint := labelStyle.Render(wrapped)
	field := labelStyle.Render(binaryFieldLabel) + b.binaryPath.View()
	msg := lipgloss.JoinVertical(lipgloss.Center, hint, "", field)
	placed := lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, msg)
	return style.Render(placed)
}
