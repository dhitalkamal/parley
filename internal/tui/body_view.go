package tui

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"

	"github.com/charmbracelet/lipgloss"
)

// header renders everything View() shows above the active sub-editor (or,
// for BodyNone, everything there is to show) - split out so headerLines()
// can measure it directly instead of duplicating this layout as a
// hardcoded row count, which is exactly the class of bug that used to
// silently clip content off the bottom of this panel. The body-type/
// content-type tab row that used to live here moved into the Request
// zone's own header as a pair of dropdowns (see bodydropdown.go,
// workspace_view.go's requestZoneBox) - this is just the remaining
// keyboard-shortcut hint line, one row shorter than before, which is the
// vertical space the body editor itself now gets back.
// header is what View shows above the active sub-editor. The body-type and
// content-type controls moved to the Request title (the "Raw / JSON" dropdowns
// - see bodydropdown.go), so the old keyboard-shortcut hint lines here
// (ctrl+b/ctrl+g/ctrl+p) were just leftover clutter and are gone. Only the
// None type still shows a line: a plain "no body" note (no shortcut) so the
// empty tab does not read as broken. Every other type returns "" and gives
// that row back to the editor.
func (b bodyEditor) header() string {
	if b.activeType() == collection.BodyNone {
		return clipLine(labelStyle.Render("No request body. Pick a type from the menu above."), b.width)
	}
	return ""
}

// headerLines is how many rows header() renders - requestPanelView needs this
// to size the active sub-editor's own height budget correctly. An empty header
// is zero rows (lipgloss.Height would report 1 for "", so special-case it).
func (b bodyEditor) headerLines() int {
	if b.header() == "" {
		return 0
	}
	return lipgloss.Height(b.header())
}

// View shows the body-type (and, when Raw, content-type) choices as real
// tab bars, followed by whichever sub-editor is active. None/Raw render
// flush against the Request panel's own border (no nested border of their
// own - at this panel's typical height a second border ran nearly floor to
// ceiling, which read as a stray vertical line rather than a deliberate
// box); every other type reuses a widget that already draws its own
// border, the same as Params/Headers/Pre-request/Tests (see
// rendersBorderless).
func (b bodyEditor) View() string {
	switch b.activeType() {
	case collection.BodyNone:
		return b.header()
	case collection.BodyRaw:
		return b.area.View()
	default:
		// header() is "" for these now, so there is no leading line to join -
		// the sub-editor renders flush against the panel's own border.
		return b.activeSubEditorView()
	}
}

func (b bodyEditor) activeSubEditorView() string {
	switch b.activeType() {
	case collection.BodyURLEncoded, collection.BodyMultipart:
		return b.formTable.View()
	case collection.BodyGraphQL:
		return b.graphQLView()
	case collection.BodyBinary:
		return b.binaryView()
	default:
		return ""
	}
}
