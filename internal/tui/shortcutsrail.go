package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// shortcutsRailOuterWidth is the rail's fixed total width (border included).
// shortcutsRailKeyColumnWidth is the padded key column inside it - together
// with borderStyle's own border(2)+padding(2) overhead, this fixes the
// description column at exactly shortcutsRailOuterWidth-4-
// shortcutsRailKeyColumnWidth characters, which every line below is chosen
// (and defensively clipped, see shortcutsSection) to fit inside.
const (
	shortcutsRailOuterWidth     = 40
	shortcutsRailKeyColumnWidth = 14
)

// minWidthForShortcutsRail is the total terminal width below which the rail
// is dropped entirely rather than squeezing the Request/Response zones -
// the same graceful-degradation floor pattern workspaceHorizontalFloor
// already uses for the two-zone split itself.
const minWidthForShortcutsRail = 120

// shortcutsRailVisible reports whether the Environment section is currently
// interactable: the left sidebar has to be on screen (terminal wide enough) and
// the section not collapsed to its header bar. Now that Environment lives in the
// left column rather than a right rail, there's no separate wide-terminal floor
// for it beyond the sidebar's own.
func (m Model) shortcutsRailVisible() bool {
	return m.leftSidebarShown() && m.environmentExpanded()
}

// shortcutKeyStyle colors just the key column - padded on the plain text
// first, then styled as a whole (see history.go's View for why: padding an
// already-ANSI-wrapped string with %-Ns counts escape bytes toward the
// width and misaligns every column after it).
var shortcutKeyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(activeTheme.Accent))

type shortcutLine struct {
	key  string
	desc string
}

// railView picks which of the rail's two views to render for the current
// mode: the compact editable environment editor (the default) or the
// shortcuts reference list. Both produce a box of identical width/height, so
// they swap in place - see view.go, which reserves the same railWidth either
// way.
func (m Model) railView(height int) string {
	if m.railEnvMode() {
		return railEnvView(m, height, m.effectiveFocus() == focusRail)
	}
	return shortcutsRailView(m, height)
}

// shortcutsRailView renders the persistent "KEY SHORTCUTS" reference panel
// shown beside the Request/Response workspace on a wide enough terminal -
// same bordered-zone look as Request/Response (zoneHeaderText/titledBox),
// so it reads as a third panel rather than a different kind of UI element.
func shortcutsRailView(m Model, height int) string {
	innerHeight := height - 2
	content := shortcutsRailContent(m)
	// Clip, don't let Height() silently grow past its own budget: this
	// panel's three sections are a fixed reference list, not something that
	// shrinks with the terminal, so a short enough terminal can hand this a
	// budget narrower than the content's own natural height - and
	// lipgloss.Style.Height() only pads shorter content, it never truncates
	// taller content (the same class of bug reqpanel.go's/kvtable.go's own
	// notes describe for their panels).
	if lines := strings.Split(content, "\n"); len(lines) > innerHeight {
		content = strings.Join(lines[:innerHeight], "\n")
	}
	body := borderStyle.Width(shortcutsRailOuterWidth - 2).Height(innerHeight).Render(content)
	return titledBox(body, zoneHeaderText("Shortcuts", true, false))
}

// shortcutsRailContent builds the three always-visible sections: Request
// (context-sensitive to the active reqTab), Response, and Global - the
// mockup's own grouping. Response/Global are static reference lists;
// Request's tab-specific entries are appended to a small common set so
// switching tabs changes what's relevant without hiding the keys that work
// on every tab.
func shortcutsRailContent(m Model) string {
	sections := []string{
		shortcutsSection("Request", requestShortcutLines(m.reqTab)),
		shortcutsSection("Response", responseShortcutLines()),
		shortcutsSection("Global", globalShortcutLines()),
	}
	return strings.Join(sections, "\n\n")
}

// shortcutsRailContentWidth is the plain-text width available inside the
// rail's border+padding - see shortcutsRailOuterWidth's doc comment.
const shortcutsRailContentWidth = shortcutsRailOuterWidth - 4

func shortcutsSection(title string, lines []shortcutLine) string {
	var b strings.Builder
	b.WriteString(activeTabStyle.Render(strings.ToUpper(title)))
	b.WriteString("\n")
	descWidth := shortcutsRailContentWidth - shortcutsRailKeyColumnWidth
	for _, l := range lines {
		// Clip, don't wrap: a description running even one column over
		// budget would otherwise word-wrap onto its own line, silently
		// growing this section's height and throwing off
		// shortcutsRailView's fixed-height box - the same class of bug
		// response.go's StatusLine already guards against.
		key := shortcutKeyStyle.Render(fmt.Sprintf("%-*s", shortcutsRailKeyColumnWidth, l.key))
		desc := labelStyle.Render(ansi.Cut(l.desc, 0, descWidth))
		fmt.Fprintf(&b, "%s%s\n", key, desc)
	}
	return strings.TrimRight(b.String(), "\n")
}

// requestShortcutLines is the keys that always apply to the Request panel,
// plus whichever tab-specific ones match the currently active reqTab - see
// reqtab.go for the tab set (Params/Headers/Body/Auth/Scripts/Settings).
func requestShortcutLines(tab reqTab) []shortcutLine {
	lines := []shortcutLine{
		{"shift+<-/->", "prev/next tab"},
		{"ctrl+r", "send request"},
		{"ctrl+s", "save"},
	}
	switch tab {
	case reqTabParams, reqTabHeaders:
		lines = append(lines,
			shortcutLine{"a", "add row"},
			shortcutLine{"d", "delete row"},
			shortcutLine{"space", "enable/disable"},
			shortcutLine{"enter", "edit row"},
		)
	case reqTabBody:
		lines = append(lines,
			shortcutLine{"ctrl+b", "body type"},
			shortcutLine{"ctrl+g", "content-type"},
			shortcutLine{"ctrl+p", "format (raw)"},
		)
	case reqTabAuth:
		lines = append(lines,
			shortcutLine{"enter/space", "expand section"},
		)
	case reqTabScripts:
		lines = append(lines,
			shortcutLine{"ctrl+b", "pre-req/test"},
		)
	case reqTabSettings:
		lines = append(lines,
			shortcutLine{"up/down", "change field"},
			shortcutLine{"space/enter", "toggle"},
		)
	}
	return lines
}

// responseShortcutLines is a static reference list - unlike Request, the
// Response panel's own keys don't vary by which mode (Body/Headers/
// Cookies/Tests/Timeline) is active.
func responseShortcutLines() []shortcutLine {
	return []shortcutLine{
		{"ctrl+t", "next tab"},
		{"ctrl+f", "search"},
		{"y", "copy"},
		{"Y", "copy line"},
		{"ctrl+u", "reveal secrets"},
		{"ctrl+z", "zoom"},
	}
}

// globalShortcutLines is every screen-level shortcut reachable regardless
// of which panel has focus.
func globalShortcutLines() []shortcutLine {
	return []shortcutLine{
		{"ctrl+k", "commands"},
		{"ctrl+\\", "collections"},
		{"f5", "dashboard"},
		{"f9", "settings"},
		{"?", "help"},
		{"esc", "back"},
	}
}
