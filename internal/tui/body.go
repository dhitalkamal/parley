package tui

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// bodyTypes are the body kinds selectable via ctrl+b, in cycle order.
// URLEncoded/Multipart reuse kvTable (see body_form.go); GraphQL is a
// query+variables pair (body_graphql.go); Binary is a single file-path
// field (body_binary.go).
var bodyTypes = []collection.BodyType{
	collection.BodyNone, collection.BodyRaw, collection.BodyURLEncoded,
	collection.BodyMultipart, collection.BodyGraphQL, collection.BodyBinary,
}

var bodyTypeLabels = []string{"None", "Raw", "Form", "Multipart", "GraphQL", "Binary"}

var rawContentTypes = []string{"application/json", "text/plain", "application/xml"}

// rawTypeIdx finds collection.BodyRaw's index in bodyTypes rather than assuming
// it's 1, so it stays correct if bodyTypes ever gains an entry before it.
func rawTypeIdx() int {
	for i, t := range bodyTypes {
		if t == collection.BodyRaw {
			return i
		}
	}
	return 0
}

type bodyEditor struct {
	typeIdx        int
	contentTypeIdx int
	area           textarea.Model // BodyRaw
	formTable      kvTable        // BodyURLEncoded, BodyMultipart
	graphQuery     textarea.Model // BodyGraphQL
	graphVars      textarea.Model // BodyGraphQL
	graphField     int            // 0 = query, 1 = variables focused/shown
	binaryPath     textinput.Model
	binaryBoxWidth int
	binaryHeight   int
	width          int // last width given to SetSize - header() needs it to clip the type tabs (see tabBarWithUnderline)
}

func newBodyEditor() bodyEditor {
	a := textarea.New()
	a.Placeholder = `{"key": "value"}`
	// drop the default "|" prompt gutter (bubbles sets Prompt to a vertical
	// bar) - it rendered as a stray bright line down the left of the editor,
	// inside the panel's own border. Same for every other textarea below.
	a.Prompt = ""

	ft := newKVTable("Form fields", "Key",
		"Fields are sent as the request body. For a Multipart field, prefix the value with @ to upload a file instead of literal text, e.g. @/path/to/file.png")
	ft.SetMaskSecrets(true)

	gq := textarea.New()
	gq.Placeholder = "query { ... }"
	gq.Prompt = ""
	gv := textarea.New()
	gv.Placeholder = `{"variable": "value"}`
	gv.Prompt = ""

	bp := textinput.New()
	bp.Placeholder = "/path/to/file"

	return bodyEditor{area: a, formTable: ft, graphQuery: gq, graphVars: gv, binaryPath: bp}
}

// activeType is a shorthand for bodyTypes[b.typeIdx], used throughout so the
// index math stays in one place.
func (b bodyEditor) activeType() collection.BodyType {
	return bodyTypes[b.typeIdx]
}

func (b bodyEditor) Body() collection.Body {
	switch b.activeType() {
	case collection.BodyNone:
		return collection.Body{Type: collection.BodyNone}
	case collection.BodyRaw:
		return collection.Body{Type: collection.BodyRaw, RawContentType: rawContentTypes[b.contentTypeIdx], RawText: b.area.Value()}
	case collection.BodyURLEncoded, collection.BodyMultipart:
		return collection.Body{Type: b.activeType(), FormFields: formFieldsFromRows(b.formTable.Rows(), b.activeType() == collection.BodyMultipart)}
	case collection.BodyGraphQL:
		return collection.Body{Type: collection.BodyGraphQL, GraphQLQuery: b.graphQuery.Value(), GraphQLVariables: b.graphVars.Value()}
	case collection.BodyBinary:
		return collection.Body{Type: collection.BodyBinary, BinaryFilePath: b.binaryPath.Value()}
	default:
		return collection.Body{Type: collection.BodyNone}
	}
}

func (b *bodyEditor) SetBody(body collection.Body) {
	for i, t := range bodyTypes {
		if t == body.Type {
			b.typeIdx = i
		}
	}
	for i, ct := range rawContentTypes {
		if ct == body.RawContentType {
			b.contentTypeIdx = i
		}
	}
	b.area.SetValue(body.RawText)
	b.formTable.SetRows(rowsFromFormFields(body.FormFields))
	b.graphQuery.SetValue(body.GraphQLQuery)
	b.graphVars.SetValue(body.GraphQLVariables)
	b.binaryPath.SetValue(body.BinaryFilePath)
}

func (b *bodyEditor) SetFocus(focused bool) {
	b.area.Blur()
	b.formTable.SetFocus(false)
	b.graphQuery.Blur()
	b.graphVars.Blur()
	b.binaryPath.Blur()
	if !focused {
		return
	}
	switch b.activeType() {
	case collection.BodyNone:
		// The very first keystroke while None promotes to Raw and is
		// forwarded to the textarea in the same Update call (see
		// updateActiveSubEditor) - it has to already be focused for that
		// keystroke to register, or it's silently dropped.
		b.area.Focus()
	case collection.BodyRaw:
		b.area.Focus()
	case collection.BodyURLEncoded, collection.BodyMultipart:
		b.formTable.SetFocus(true)
	case collection.BodyGraphQL:
		b.focusActiveGraphField()
	case collection.BodyBinary:
		b.binaryPath.Focus()
	}
}

func (b *bodyEditor) SetSize(w, h int) {
	b.width = w
	b.area.SetWidth(w)
	b.area.SetHeight(h)
	b.formTable.SetWidth(w)
	b.formTable.SetHeight(h)
	b.setGraphQLSize(w, h)
	b.setBinarySize(w, h)
}

// HasBody reports whether a body is set (anything other than BodyNone) -
// drives the tab-bar's "Body *" badge.
func (b bodyEditor) HasBody() bool {
	return b.activeType() != collection.BodyNone
}

// rendersBorderless reports whether View() renders without its own nested
// border - true only for None/Raw, which sit flush against the Request
// panel's own border (see View's doc comment). Every other type reuses a
// widget (kvTable, or a scriptEditor-style dual textarea) that already
// draws its own border, the same way Params/Headers/Pre-request/Tests do -
// requestPanelView (reqpanel.go) needs this to size their height budget
// like those tabs instead of like the borderless Raw case.
func (b bodyEditor) rendersBorderless() bool {
	t := b.activeType()
	return t == collection.BodyNone || t == collection.BodyRaw
}

// ExtraLines reports how many lines View adds on top of a bordered
// sub-editor - only kvTable's inline edit form (see kvTable.ExtraLines)
// needs this; requestPanelView shrinks the height budget by the same
// amount before calling SetSize, or the extra lines get silently clipped.
func (b bodyEditor) ExtraLines() int {
	if b.activeType() == collection.BodyURLEncoded || b.activeType() == collection.BodyMultipart {
		return b.formTable.ExtraLines()
	}
	return 0
}

// Update returns handled=false for keys it doesn't own, so the caller can
// still cycle body type / content type with dedicated keys before falling
// through to whichever sub-editor is active.
func (b bodyEditor) Update(msg tea.Msg) (bodyEditor, tea.Cmd, bool) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "ctrl+b":
			b.typeIdx = (b.typeIdx + 1) % len(bodyTypes)
			return b, nil, true
		case "ctrl+g":
			return b.handleCtrlG()
		case "ctrl+p":
			b.formatJSON()
			return b, nil, true
		}
		// Typing directly into an empty body IS the user declaring one -
		// requiring ctrl+b first (to leave None) before any character ever
		// reached the textarea made typing look like it did nothing.
		if b.activeType() == collection.BodyNone && (k.Type == tea.KeyRunes || k.Type == tea.KeyEnter) {
			b.typeIdx = rawTypeIdx()
		}
	}
	return b.updateActiveSubEditor(msg)
}

// handleCtrlG's meaning depends on the active type: cycles Raw's
// content-type tabs (its original meaning), toggles GraphQL's query/
// variables field, or is a no-op for every other type - mirrors how
// scriptEditor's ctrl+b already means something different depending on
// which of its two fields is showing.
func (b bodyEditor) handleCtrlG() (bodyEditor, tea.Cmd, bool) {
	switch b.activeType() {
	case collection.BodyRaw:
		b.contentTypeIdx = (b.contentTypeIdx + 1) % len(rawContentTypes)
	case collection.BodyGraphQL:
		b.graphField = (b.graphField + 1) % 2
		b.focusActiveGraphField()
	}
	return b, nil, true
}

func (b bodyEditor) updateActiveSubEditor(msg tea.Msg) (bodyEditor, tea.Cmd, bool) {
	var cmd tea.Cmd
	switch b.activeType() {
	case collection.BodyNone:
		return b, nil, false
	case collection.BodyRaw:
		b.area, cmd = b.area.Update(msg)
	case collection.BodyURLEncoded, collection.BodyMultipart:
		var handled bool
		b.formTable, cmd, handled = b.formTable.Update(msg)
		if !handled {
			return b, nil, false
		}
	case collection.BodyGraphQL:
		b, cmd = b.updateActiveGraphField(msg)
	case collection.BodyBinary:
		var tiCmd tea.Cmd
		b.binaryPath, tiCmd = b.binaryPath.Update(msg)
		cmd = tiCmd
	}
	return b, cmd, true
}

// formatJSON reindents the body text in place if it's valid JSON - bound to
// ctrl+p, a no-op (rather than an error to dismiss) when the text isn't
// valid JSON, matching how a "Beautify" button behaves in other HTTP
// clients: nothing to beautify, nothing happens. Uses the same PrettyJSON
// (json.Indent) the response viewer already relies on, rather than
// unmarshal-then-remarshal into a map - the latter alphabetizes object
// keys, silently reordering whatever the user actually typed.
func (b *bodyEditor) formatJSON() {
	if b.activeType() != collection.BodyRaw {
		return
	}
	formatted, ok := PrettyJSON([]byte(b.area.Value()))
	if !ok {
		return
	}
	b.area.SetValue(formatted)
}
