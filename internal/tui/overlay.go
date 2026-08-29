package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// overlay splices fg centered on top of bg - see overlayAt for the general
// case this delegates to.
func overlay(bg, fg string, totalWidth, totalHeight int) string {
	fgLines := strings.Split(fg, "\n")
	fgWidth := 0
	for _, l := range fgLines {
		if w := lipgloss.Width(l); w > fgWidth {
			fgWidth = w
		}
	}
	x := (totalWidth - fgWidth) / 2
	y := (totalHeight - len(fgLines)) / 2
	return overlayAt(bg, fg, totalWidth, totalHeight, x, y)
}

// overlayAt splices fg on top of bg at column x, row y, producing a single
// totalWidth x totalHeight block. lipgloss has no built-in layer
// compositing, so this is what lets dialogs and dropdowns float over the
// still-visible screen instead of blanking it out with lipgloss.Place -
// splicing happens line by line, ANSI-aware so styled backgrounds and
// dialogs don't get their escape codes cut mid-sequence. x and y are clamped
// so fg always lands fully within bounds rather than spilling off the edge.
func overlayAt(bg, fg string, totalWidth, totalHeight, x, y int) string {
	bgLines := padLines(strings.Split(bg, "\n"), totalWidth, totalHeight)
	fgLines := strings.Split(fg, "\n")

	fgWidth := 0
	for _, l := range fgLines {
		if w := lipgloss.Width(l); w > fgWidth {
			fgWidth = w
		}
	}

	if x < 0 {
		x = 0
	}
	if max := totalWidth - fgWidth; x > max {
		x = max
	}
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if max := totalHeight - len(fgLines); y > max {
		y = max
	}
	if y < 0 {
		y = 0
	}

	out := make([]string, totalHeight)
	copy(out, bgLines)
	for i, fgLine := range fgLines {
		row := y + i
		if row < 0 || row >= totalHeight {
			continue
		}
		left := ansi.Cut(bgLines[row], 0, x)
		right := ansi.Cut(bgLines[row], x+lipgloss.Width(fgLine), totalWidth)
		out[row] = left + fgLine + right
	}
	return strings.Join(out, "\n")
}

// padLines pads/truncates lines to exactly width columns and height rows so
// overlay's row/column splice math stays valid regardless of what the
// background actually rendered at.
func padLines(lines []string, width, height int) []string {
	out := make([]string, height)
	for i := 0; i < height; i++ {
		var line string
		if i < len(lines) {
			line = lines[i]
		}
		w := lipgloss.Width(line)
		switch {
		case w < width:
			line += strings.Repeat(" ", width-w)
		case w > width:
			line = ansi.Cut(line, 0, width)
		}
		out[i] = line
	}
	return out
}
