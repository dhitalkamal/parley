package tui

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// dropdownArrow is the affordance shown after a dropdown selector's current
// value - a filled down-triangle (see glyphs.go). The leading space stands in
// for the old " v" spacing. The glyph is one display column but three UTF-8
// bytes, so every width calculation below uses lipgloss.Width, never len().
var dropdownArrow = " " + glyphTriangleDown

// labelDropdownState is the small, static-list dropdown shared by the
// Request zone header's body-type and content-type controls (see
// workspace_view.go/reqpanel.go) - simpler than envDropdownState, since
// neither list here has a trailing "Manage..." entry to special-case.
type labelDropdownState struct {
	active bool
	cursor int
}

// Open shows the dropdown with the cursor starting on the current
// selection, so opening it doesn't itself change anything until a
// different entry is actually picked.
func (d *labelDropdownState) Open(selected int) {
	d.active = true
	d.cursor = selected
}

func (d *labelDropdownState) Close() {
	d.active = false
}

// MoveCursor shifts the selection by delta, clamped to [0, count-1] - no
// wraparound, matching envDropdownState's own MoveCursor.
func (d *labelDropdownState) MoveCursor(delta, count int) {
	d.cursor += delta
	if d.cursor < 0 {
		d.cursor = 0
	}
	if d.cursor > count-1 {
		d.cursor = count - 1
	}
}

// View renders the dropdown as a small bordered list, the selected entry
// highlighted.
func (d labelDropdownState) View(labels []string) string {
	var b strings.Builder
	for i, label := range labels {
		line := "  " + label
		if i == d.cursor {
			line = selStyle.Bold(true).Render("> " + label)
		}
		b.WriteString(line)
		if i < len(labels)-1 {
			b.WriteString("\n")
		}
	}
	return borderStyle.Render(b.String())
}

// shortContentTypeLabels maps a raw content-type string to a compact label
// for the header's right-side dropdown text - "application/json" would
// otherwise eat most of the header row's width.
var shortContentTypeLabels = map[string]string{
	"application/json": "JSON",
	"text/plain":       "Text",
	"application/xml":  "XML",
}

func shortContentTypeLabel(contentType string) string {
	if s, ok := shortContentTypeLabels[contentType]; ok {
		return s
	}
	return contentType
}

// requestHeaderTitleText is the plain (unstyled) text of the Request zone
// header's left-side title - used purely for its rendered width, which is
// identical whether or not it's actually focused/styled, so click/anchor
// math below never needs to know that.
const requestHeaderTitleText = "v Request"

// bodyDropdownClickTarget reports whether an absolute screen click (x, y)
// landed on the Request zone header's body-type or content-type dropdown
// label, given that zone's own absolute origin and rendered width - this
// mirrors titledBoxWithRight/rightTextX's exact splice position, the same
// compute-once-share-with-render-and-click-testing principle
// computeWorkspaceGeom already follows, so a click can never disagree with
// what's actually on screen.
func (m Model) bodyDropdownClickTarget(x, y, panelLeft, panelTop, panelWidth int) (isBodyType, isContentType bool) {
	if y != panelTop || m.reqTab != reqTabBody {
		return false, false
	}
	right := m.bodyHeaderRightText()
	if right == "" {
		return false, false
	}
	rightX := rightTextX(panelWidth, requestHeaderTitleText, right)
	if rightX < 0 {
		return false, false
	}
	textStart := panelLeft + rightX + 1 // +1 for titledBoxWithRight's own leading space
	relX := x - textStart
	if relX < 0 || relX >= lipgloss.Width(right) {
		return false, false
	}
	typeLabel := bodyTypeLabels[m.body.typeIdx] + dropdownArrow
	if relX < lipgloss.Width(typeLabel) {
		return true, false
	}
	if m.body.activeType() != collection.BodyRaw {
		return false, false
	}
	ctStart := lipgloss.Width(typeLabel) + 2 // "  " separator
	return false, relX >= ctStart
}

// bodyTypeDropdownAnchor positions the body-type dropdown just under its
// own label in the Request zone header.
func (m Model) bodyTypeDropdownAnchor() (x, y int) {
	g, xOffset := m.currentWorkspaceGeom()
	panelTop := gridTop() + g.request.y
	rightX := rightTextX(g.request.width, requestHeaderTitleText, m.bodyHeaderRightText())
	if rightX < 0 {
		rightX = 0
	}
	return xOffset + g.request.x + rightX + 1, panelTop + 1
}

// contentTypeDropdownAnchor is bodyTypeDropdownAnchor's counterpart for the
// content-type dropdown, offset past the body-type label and its
// separator.
func (m Model) contentTypeDropdownAnchor() (x, y int) {
	bx, by := m.bodyTypeDropdownAnchor()
	typeLabel := bodyTypeLabels[m.body.typeIdx] + dropdownArrow
	return bx + lipgloss.Width(typeLabel) + 2, by
}

// bodyHeaderRightText is the Request zone header's right-side dropdown
// text (see workspace_view.go) - the body-type label, plus the content-
// type label when the active type carries a text body (Raw only: GraphQL
// bodies are always sent as JSON with no stored content-type to select
// from, so there's nothing for a content-type dropdown to control there).
// Empty whenever the Body tab isn't the one selected - showing a dropdown
// for a tab that isn't visible would be a live control with nothing to
// control.
func (m Model) bodyHeaderRightText() string {
	if m.reqTab != reqTabBody {
		return ""
	}
	typeLabel := bodyTypeLabels[m.body.typeIdx] + dropdownArrow
	if m.body.activeType() != collection.BodyRaw {
		return typeLabel
	}
	ctLabel := shortContentTypeLabel(rawContentTypes[m.body.contentTypeIdx]) + dropdownArrow
	return typeLabel + "  " + ctLabel
}
