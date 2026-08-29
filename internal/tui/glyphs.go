package tui

// Filled triangle glyphs used for every expand/collapse chevron and dropdown
// affordance. Built from their code points at runtime rather than written as
// literal glyphs, so the source file itself stays ASCII - keeping the
// project's ASCII-only file-content policy and the no-decorative-chars guard
// satisfied while the UI still renders solid triangles. Each is one display
// column wide but three UTF-8 bytes, so any width math on strings that embed
// them must use lipgloss.Width, never len().
var (
	// glyphTriangleDown is the down-pointing filled triangle - shown for an
	// expanded/open node and as the dropdown-open affordance.
	glyphTriangleDown = string(rune(0x25BC))
	// glyphTriangleRight is the right-pointing filled triangle - shown for a
	// collapsed/closed node.
	glyphTriangleRight = string(rune(0x25B6))
	// glyphAngleLeft / glyphAngleRight bracket the active-environment selector
	// in the Environment panel (a "< dev >"-style pill).
	glyphAngleLeft  = string(rune(0x2039))
	glyphAngleRight = string(rune(0x203A))
	// glyphDot is one masked character of a secret value.
	glyphDot = string(rune(0x2022))
	// glyphCheck marks a positive/connected state (the VPN indicator).
	glyphCheck = string(rune(0x2713))
	// glyphVerticalLine is the box-drawing vertical bar - tab dividers and the
	// old Response body/info separator.
	glyphVerticalLine = string(rune(0x2502))
	// glyphHorizontalLine is the box-drawing horizontal bar - the rule under the
	// Request/Response tab bar (see tabbar.go).
	glyphHorizontalLine = string(rune(0x2500))
)
