package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// railEnvView renders the rail's environment view: the active environment and
// its variable names at the top, and the add/edit form (or key hints) pinned
// to the bottom of the box - so the form sits in the same place no matter how
// many variables there are, instead of floating just under the last row. It
// shares the shortcuts rail's fixed width, height clamp, and bordered-zone
// look (borderStyle vs focusedBorder when focused) so the two modes swap in
// place without shifting the surrounding frame.
func railEnvView(m Model, height int, focused bool) string {
	innerHeight := height - 2
	content := composeRailEnv(railEnvTop(m, focused), railEnvBottom(m, focused), innerHeight)
	// Clip every line to the content width so a long value (e.g. mid-edit, when
	// the value input holds a URL longer than the box) can never word-wrap and
	// push the box taller than innerHeight - which would break the whole
	// frame's alignment. ANSI-aware so styled edit fields keep their color.
	content = clipRailContent(content, shortcutsRailContentWidth)
	border := borderStyle
	if focused {
		border = focusedBorder
	}
	body := border.Width(shortcutsRailOuterWidth - 2).Height(innerHeight).Render(content)
	return titledBox(body, zoneHeaderText("Environment", true, focused))
}

// clipRailContent clips each line of s to at most width display columns.
func clipRailContent(s string, width int) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		if lipgloss.Width(l) > width {
			lines[i] = ansi.Cut(l, 0, width)
		}
	}
	return strings.Join(lines, "\n")
}

// composeRailEnv stacks top at the top and bottom pinned to the last rows of
// innerHeight, padding the gap with blank lines. If they can't both fit, the
// bottom (the active add/edit form, or the key hints) wins and the top is
// clipped from its end - the part you're interacting with should never be the
// part that scrolls off.
func composeRailEnv(top, bottom string, innerHeight int) string {
	if innerHeight < 1 {
		return ""
	}
	topLines := strings.Split(top, "\n")
	var bottomLines []string
	if bottom != "" {
		bottomLines = strings.Split(bottom, "\n")
	}
	if len(bottomLines) >= innerHeight {
		return strings.Join(bottomLines[len(bottomLines)-innerHeight:], "\n")
	}
	avail := innerHeight - len(bottomLines)
	if len(topLines) > avail {
		topLines = topLines[:avail]
	}
	lines := topLines
	for len(lines) < avail {
		lines = append(lines, "")
	}
	lines = append(lines, bottomLines...)
	return strings.Join(lines, "\n")
}

// railEnvTop is the always-visible upper section: the active-environment
// selector, then this environment's VARIABLES (editable), then the GLOBALS
// section - shown in EVERY environment, since globals are shared and applied on
// top of whichever env is active. Both variable sections are editable; the
// cursor's section (see railSection) drives which rows highlight.
func railEnvTop(m Model, focused bool) string {
	var b strings.Builder

	// VPN indicator (read-only detection - parley never connects/disconnects).
	// Shows the tunnel interface that's up, or "No VPN detected".
	if m.detectedVPN != "" {
		b.WriteString(labelStyle.Render("VPN  ") + statusStyle.Render(ansi.Cut(m.detectedVPN, 0, shortcutsRailContentWidth-8)+" "+glyphCheck))
	} else {
		b.WriteString(labelStyle.Render("VPN  ") + labelStyle.Render("No VPN detected"))
	}
	b.WriteString("\n\n")

	// Selector: Active:  < dev >  (left/right switch env; the marker shows when
	// the cursor is on it).
	scope := m.activeEnvName
	if scope == "" {
		scope = "none"
	}
	selMarker := "  "
	if focused && m.railSection == railSelector {
		selMarker = glyphTriangleRight + " " // "you are here" - the Environment selector
	}
	selector := glyphAngleLeft + " " + ansi.Cut(scope, 0, 16) + " " + glyphAngleRight
	b.WriteString(labelStyle.Render(selMarker+"Active:  ") + accentStyle.Bold(true).Render(selector))
	b.WriteString("\n")
	if line := expectedVPNLine(m, focused); line != "" {
		b.WriteString(line + "\n")
	}
	b.WriteString("\n")

	// VARIABLES: the active named environment's own variables.
	envRows := m.railEnv.Rows()
	b.WriteString(railSectionHeader("VARIABLES", len(envRows), focused && m.railSection == railEnvVars))
	b.WriteString("\n")
	switch {
	case m.activeEnvName == "":
		b.WriteString(labelStyle.Render(railVarIndent + "  no env - " + glyphAngleLeft + " " + glyphAngleRight + " to pick"))
	case len(envRows) == 0:
		b.WriteString(labelStyle.Render(railVarIndent + "  no variables"))
	default:
		sel := m.railEnv.SelectedIndex()
		for i, r := range envRows {
			b.WriteString(railVarLine(r, focused && m.railSection == railEnvVars && i == sel, false))
			if i < len(envRows)-1 {
				b.WriteString("\n")
			}
		}
	}

	// GLOBALS: shared variables, always shown and editable (navigate down into
	// this section to edit them - they apply to every environment).
	b.WriteString("\n\n")
	globRows := m.railGlobals.Rows()
	b.WriteString(railSectionHeader("GLOBALS", len(globRows), focused && m.railSection == railGlobalVars))
	b.WriteString("\n")
	if len(globRows) == 0 {
		b.WriteString(labelStyle.Render(railVarIndent + "  none"))
	} else {
		sel := m.railGlobals.SelectedIndex()
		for i, r := range globRows {
			b.WriteString(railVarLine(r, focused && m.railSection == railGlobalVars && i == sel, false))
			if i < len(globRows)-1 {
				b.WriteString("\n")
			}
		}
	}
	return b.String()
}

// expectedVPNLine renders the active environment's expected-VPN hint under the
// selector: "(any)" when none is set (with a "p pick" nudge while the selector
// is focused), a green name + check when that tunnel is currently up, or a red
// name + "!" when it's expected but not connected. Empty when no environment is
// active - there's nothing to attach an expectation to. Display only; picking
// never touches the VPN itself (see railvpn.go).
func expectedVPNLine(m Model, focused bool) string {
	if m.activeEnvName == "" {
		return ""
	}
	// Rendered as its own selector row, mirroring the Active env selector above:
	// a "you are here" marker when the cursor is on it, then a "< value >" pill
	// that left/right cycle through ((none) -> each open tunnel -> (none)).
	marker := "  "
	if focused && m.railSection == railVPNSelector {
		marker = glyphTriangleRight + " "
	}
	val := m.activeEnv.ExpectedVPN
	if val == "" {
		val = "(none)"
	}
	selector := glyphAngleLeft + " " + ansi.Cut(val, 0, 14) + " " + glyphAngleRight
	line := labelStyle.Render(marker+"Expects: ") + accentStyle.Bold(true).Render(selector)
	// Connected status when a specific tunnel is expected.
	if m.activeEnv.ExpectedVPN != "" {
		if vpnConnected(m.activeEnv.ExpectedVPN, m.detectedVPNs) {
			line += " " + statusStyle.Render(glyphCheck)
		} else {
			line += " " + errStyle.Render("!")
		}
	}
	return line
}

// railSectionHeader renders a section label with its count right-aligned to the
// panel's content width. When active (the cursor is in this section) it gets a
// filled "you are here" marker and a bright accent; otherwise it's dim - so the
// section you're on is always obvious, even when it has no variables to put a
// row marker on.
func railSectionHeader(label string, count int, active bool) string {
	marker := "  "
	styled := labelStyle.Render(label)
	if active {
		marker = glyphTriangleRight + " "
		styled = activeTabStyle.Bold(true).Render(label)
	}
	countStr := fmt.Sprintf("%d", count)
	pad := shortcutsRailContentWidth - lipgloss.Width(marker) - lipgloss.Width(label) - lipgloss.Width(countStr)
	if pad < 1 {
		pad = 1
	}
	return marker + styled + strings.Repeat(" ", pad) + labelStyle.Render(countStr)
}

// fitCol pads or truncates s to exactly w display columns.
func fitCol(s string, w int) string {
	if lipgloss.Width(s) > w {
		return ansi.Cut(s, 0, w)
	}
	return s + strings.Repeat(" ", w-lipgloss.Width(s))
}

// railEnvBottom is the pinned lower section: only the inline add/edit form
// while one is open. The persistent key-hint block was removed on request, so
// there's nothing here otherwise.
func railEnvBottom(m Model, focused bool) string {
	ed := m.railEditor()
	if !ed.IsEditing() {
		return ""
	}
	title := "EDIT"
	if ed.IsAdding() {
		title = "NEW VARIABLE"
	}
	if m.railSection == railGlobalVars {
		title += " (global)"
	}
	var b strings.Builder
	b.WriteString(activeTabStyle.Render(title) + "\n")
	b.WriteString(labelStyle.Render("key ") + ed.EditKeyView() + "\n")
	b.WriteString(labelStyle.Render("val ") + ed.EditValueView() + "\n")
	b.WriteString(labelStyle.Render("enter save  esc cancel"))
	return b.String()
}

// railVarLine renders one variable as "> key   value": a selection marker, the
// key in a fixed column, then the value (secrets masked to dots, long values
// clipped with an ellipsis). Disabled rows are dimmed/struck; read-only rows
// (the GLOBALS reference section) render their key dim rather than accented.
// Each segment is sized in plain text before styling, so ANSI escapes never
// count toward the column widths (see shortcutsSection's note).
const railVarKeyWidth = 10

// railVarIndent nests a variable row under its section header (VARIABLES /
// GLOBALS), so the header and its rows read as a clean group rather than all
// sitting at the same left edge.
const railVarIndent = "  "

func railVarLine(r envVarRow, selected, readonly bool) string {
	// Indented under the header, then the selected row's marker - the same
	// filled "you are here" triangle the headers and selectors use (not a plain
	// ">", which read as an odd stray arrow next to the solid triangles).
	marker := railVarIndent + "  "
	if selected {
		marker = railVarIndent + accentStyle.Render(glyphTriangleRight) + " "
	}
	key := fitCol(r.Key, railVarKeyWidth)
	value := r.Value
	if r.Secret {
		value = strings.Repeat(glyphDot, 6)
	}
	// Key column padded to a fixed width, then a plain space before the value -
	// the padding already gives clean column alignment without a divider.
	valWidth := shortcutsRailContentWidth - lipgloss.Width(marker) - railVarKeyWidth - 1
	if valWidth < 0 {
		valWidth = 0
	}
	value = ansi.Cut(value, 0, valWidth)

	if !r.Enabled {
		return marker + disabledRowStyle.Render(key+" "+value)
	}
	keyStyle := shortcutKeyStyle
	if readonly {
		keyStyle = labelStyle
	}
	return marker + keyStyle.Render(key) + " " + labelStyle.Render(value)
}
