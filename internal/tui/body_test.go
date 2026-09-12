package tui

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TestBodyEditor_TypingPromotesFromNoneToRaw guards a real bug a user hit:
// the body type defaults to None, and Update used to swallow every key
// except ctrl+b/ctrl+g while None - so typing looked like it did nothing at
// all, with no way to tell why short of already knowing to press ctrl+b
// first.
func TestBodyEditor_TypingPromotesFromNoneToRaw(t *testing.T) {
	b := newBodyEditor()
	b.SetFocus(true)
	if bodyTypes[b.typeIdx] != collection.BodyNone {
		t.Fatalf("newBodyEditor should default to BodyNone")
	}

	b, _, handled := b.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if !handled {
		t.Fatalf("typing while body type is None should be handled")
	}
	if bodyTypes[b.typeIdx] != collection.BodyRaw {
		t.Errorf("typing should promote body type from None to Raw, got %v", bodyTypes[b.typeIdx])
	}
	if b.area.Value() != "h" {
		t.Errorf("area.Value() = %q, want the typed character to reach the textarea", b.area.Value())
	}
}

// TestBodyEditor_ViewNoLongerShowsItsOwnTypeTabs guards the move of the
// body-type/content-type tab row out of the editor's own header and into
// the Request zone header as a pair of dropdowns (see bodydropdown.go,
// workspace_view.go's requestZoneBox / bodyHeaderRightText) - the editor's
// own View() shows only the keyboard-shortcut hint line now, freeing the
// row(s) the old tab row used for the editor itself (see
// TestBodyEditor_HeaderLinesShrankByOneRow).
func TestBodyEditor_ViewNoLongerShowsItsOwnTypeTabs(t *testing.T) {
	b := newBodyEditor()
	b.SetSize(60, 5)

	view := stripANSI(b.View())

	if strings.Contains(view, "Multipart") {
		t.Errorf("got %q, want no body-type tab row in the editor's own view", view)
	}
	// the ctrl+b/ctrl+g/ctrl+p shortcut hints are gone (the controls live in the
	// Request title dropdowns now); None shows a plain no-body note instead.
	if strings.Contains(view, "ctrl+b") {
		t.Errorf("got %q, want no keyboard-shortcut hint in the body area", view)
	}
	if !strings.Contains(view, "No request body") {
		t.Errorf("got %q, want the plain no-body note", view)
	}
}

// TestBodyEditor_ViewNoLongerShowsContentTypeTabs is
// TestBodyEditor_ViewNoLongerShowsItsOwnTypeTabs's Raw-specific
// counterpart - content-type selection is a header dropdown now too.
func TestBodyEditor_ViewNoLongerShowsContentTypeTabs(t *testing.T) {
	b := newBodyEditor()
	b.SetSize(60, 5)
	b.typeIdx = rawTypeIdx()

	rawView := stripANSI(b.View())
	if strings.Contains(rawView, "application/json") {
		t.Errorf("got %q, want no content-type tab row in the editor's own view", rawView)
	}
}

// TestBodyEditor_HeaderLinesShrankByOneRow guards the freed-vertical-space
// acceptance criterion: removing the tab row (moved to the Request zone
// header) should shrink header() by exactly the one row that row used to
// take, handing that row to the active sub-editor instead.
func TestBodyEditor_HeaderLinesShrankByOneRow(t *testing.T) {
	b := newBodyEditor()
	b.SetSize(60, 5)
	if got := b.headerLines(); got != 1 {
		t.Errorf("headerLines() = %d, want 1 (just the hint line, no tab row)", got)
	}
}

// TestBodyEditor_ViewHasNoNestedBorder guards the fix for a user-reported
// stray vertical line: bodyEditor used to wrap its textarea in its own
// border on top of the Request panel's own, and at this panel's typical
// height that second border ran nearly floor to ceiling - which at a
// glance read as a stray line rather than a deliberate box. Corner runes
// are pulled from the actual border styles rather than typed as literals,
// so this can't accidentally start passing just because the theme's border
// glyphs changed.
func TestBodyEditor_ViewHasNoNestedBorder(t *testing.T) {
	b := newBodyEditor()
	b.typeIdx = rawTypeIdx()
	b.SetSize(60, 5)

	view := b.View()
	corners := []string{
		lipgloss.RoundedBorder().TopLeft, lipgloss.RoundedBorder().TopRight,
		lipgloss.RoundedBorder().BottomLeft, lipgloss.RoundedBorder().BottomRight,
		lipgloss.ThickBorder().TopLeft, lipgloss.ThickBorder().TopRight,
		lipgloss.ThickBorder().BottomLeft, lipgloss.ThickBorder().BottomRight,
	}
	for _, corner := range corners {
		if strings.Contains(view, corner) {
			t.Errorf("got a border corner %q in bodyEditor's View(), want no nested border", corner)
		}
	}
}

// TestBodyEditor_CtrlPFormatsValidJSONInPlace guards the new one-click
// format action that replaced the response panel's old pretty/raw toggle -
// a user asked for the response to always be fully formatted, and for a
// pretty/raw-style control to live on the request body instead, where it's
// actually being typed rather than just viewed.
func TestBodyEditor_CtrlPFormatsValidJSONInPlace(t *testing.T) {
	b := newBodyEditor()
	b.SetFocus(true)
	b.typeIdx = rawTypeIdx()
	b.area.SetValue(`{"a":1,"b":[2,3]}`)

	b, _, handled := b.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	if !handled {
		t.Fatal("expected ctrl+p to be handled by bodyEditor")
	}
	got := b.area.Value()
	want := "{\n  \"a\": 1,\n  \"b\": [\n    2,\n    3\n  ]\n}"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestBodyEditor_CtrlBCyclesThroughAllSixTypes guards the type cycle
// growing from None/Raw to include the four newer body types - a user
// asked for form-urlencoded, multipart, GraphQL, and binary/file-upload
// bodies alongside the existing None/Raw pair.
func TestBodyEditor_CtrlBCyclesThroughAllSixTypes(t *testing.T) {
	b := newBodyEditor()
	want := []collection.BodyType{
		collection.BodyRaw, collection.BodyURLEncoded, collection.BodyMultipart,
		collection.BodyGraphQL, collection.BodyBinary, collection.BodyNone,
	}
	for i, w := range want {
		var handled bool
		b, _, handled = b.Update(tea.KeyMsg{Type: tea.KeyCtrlB})
		if !handled {
			t.Fatalf("step %d: ctrl+b should be handled", i)
		}
		if got := bodyTypes[b.typeIdx]; got != w {
			t.Fatalf("step %d: type = %v, want %v", i, got, w)
		}
	}
}

// TestBodyEditor_SetBodyResetsStaleContentTypeIdx guards a real bug: SetBody
// used to only assign contentTypeIdx on a match, so loading a raw request
// whose RawContentType is outside the three known values (e.g. an imported
// "application/json; charset=utf-8") left the previously loaded request's
// index in place - Body() then rebuilt the wrong Content-Type from that stale
// index and sent/saved it.
func TestBodyEditor_SetBodyResetsStaleContentTypeIdx(t *testing.T) {
	b := newBodyEditor()
	// request A: raw xml -> contentTypeIdx lands on application/xml.
	b.SetBody(collection.Body{Type: collection.BodyRaw, RawContentType: "application/xml", RawText: "<a/>"})
	if got := b.Body().RawContentType; got != "application/xml" {
		t.Fatalf("setup: RawContentType = %q, want application/xml", got)
	}

	// request B: raw body with an unknown content type must not inherit A's.
	b.SetBody(collection.Body{Type: collection.BodyRaw, RawContentType: "application/json; charset=utf-8", RawText: "{}"})
	if got := b.Body().RawContentType; got == "application/xml" {
		t.Errorf("RawContentType = %q, want the stale application/xml index reset", got)
	}
	if b.contentTypeIdx != 0 {
		t.Errorf("contentTypeIdx = %d, want 0 (reset) for an unknown content type", b.contentTypeIdx)
	}
}

// TestBodyEditor_URLEncodedBodyRoundTrips guards Body()/SetBody() for the
// form-urlencoded type - it reuses kvTable (same editor as Params/Headers)
// rather than a bespoke widget.
func TestBodyEditor_URLEncodedBodyRoundTrips(t *testing.T) {
	b := newBodyEditor()
	b.SetBody(collection.Body{
		Type: collection.BodyURLEncoded,
		FormFields: []collection.BodyFormField{
			{Key: "user", Value: "ada", Enabled: true},
		},
	})

	got := b.Body()
	if got.Type != collection.BodyURLEncoded {
		t.Fatalf("Type = %v, want BodyURLEncoded", got.Type)
	}
	if len(got.FormFields) != 1 || got.FormFields[0].Key != "user" || got.FormFields[0].Value != "ada" {
		t.Errorf("FormFields = %+v", got.FormFields)
	}
}

// TestBodyEditor_URLEncodedIgnoresAtPrefixConvention guards URLEncoded
// never treating "@" specially - that convention only means "upload this
// file" for Multipart (see fileFieldPrefix in body_form.go); a literal "@"
// typed into a URLEncoded value must round-trip as plain text.
func TestBodyEditor_URLEncodedIgnoresAtPrefixConvention(t *testing.T) {
	b := newBodyEditor()
	b.SetBody(collection.Body{
		Type: collection.BodyURLEncoded,
		FormFields: []collection.BodyFormField{
			{Key: "handle", Value: "@ada", Enabled: true},
		},
	})

	got := b.Body()
	if len(got.FormFields) != 1 || got.FormFields[0].IsFile {
		t.Fatalf("got %+v, want IsFile=false for URLEncoded", got.FormFields)
	}
	if got.FormFields[0].Value != "@ada" {
		t.Errorf("Value = %q, want the literal @ada preserved", got.FormFields[0].Value)
	}
}

// TestBodyEditor_MultipartFileFieldUsesAtPrefixConvention guards how a
// multipart field is marked as a file within the shared kvTable editor: an
// "@" prefix on the value, the same convention curl's -F flag uses - this
// keeps multipart reusing kvTable as-is instead of a new field-type-aware
// widget.
func TestBodyEditor_MultipartFileFieldUsesAtPrefixConvention(t *testing.T) {
	b := newBodyEditor()
	b.SetBody(collection.Body{
		Type: collection.BodyMultipart,
		FormFields: []collection.BodyFormField{
			{Key: "title", Value: "photo", Enabled: true},
			{Key: "file", FilePath: "/tmp/photo.png", IsFile: true, Enabled: true},
		},
	})

	rows := b.formTable.Rows()
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	if rows[1].Value != "@/tmp/photo.png" {
		t.Errorf("file row value = %q, want @/tmp/photo.png", rows[1].Value)
	}

	got := b.Body()
	if len(got.FormFields) != 2 || !got.FormFields[1].IsFile || got.FormFields[1].FilePath != "/tmp/photo.png" {
		t.Errorf("FormFields = %+v", got.FormFields)
	}
	if got.FormFields[0].IsFile {
		t.Errorf("title field should not be marked as a file: %+v", got.FormFields[0])
	}
}

// TestBodyEditor_GraphQLBodyRoundTrips guards Body()/SetBody() for the
// query+variables pair.
func TestBodyEditor_GraphQLBodyRoundTrips(t *testing.T) {
	b := newBodyEditor()
	b.SetBody(collection.Body{
		Type:             collection.BodyGraphQL,
		GraphQLQuery:     "query { me { id } }",
		GraphQLVariables: `{"id": 1}`,
	})

	got := b.Body()
	if got.Type != collection.BodyGraphQL {
		t.Fatalf("Type = %v, want BodyGraphQL", got.Type)
	}
	if got.GraphQLQuery != "query { me { id } }" || got.GraphQLVariables != `{"id": 1}` {
		t.Errorf("got %+v", got)
	}
}

// TestBodyEditor_BinaryBodyRoundTrips guards Body()/SetBody() for the
// binary file-path field.
func TestBodyEditor_BinaryBodyRoundTrips(t *testing.T) {
	b := newBodyEditor()
	b.SetBody(collection.Body{Type: collection.BodyBinary, BinaryFilePath: "/tmp/upload.bin"})

	got := b.Body()
	if got.Type != collection.BodyBinary || got.BinaryFilePath != "/tmp/upload.bin" {
		t.Errorf("got %+v", got)
	}
}

// TestBodyEditor_HasBodyTrueForAllNonNoneTypes guards the tab-bar's "Body *"
// badge continuing to light up for every real body type, not just Raw.
func TestBodyEditor_HasBodyTrueForAllNonNoneTypes(t *testing.T) {
	b := newBodyEditor()
	if b.HasBody() {
		t.Error("HasBody() should be false for the default None type")
	}
	for _, bt := range []collection.BodyType{collection.BodyRaw, collection.BodyURLEncoded, collection.BodyMultipart, collection.BodyGraphQL, collection.BodyBinary} {
		b.SetBody(collection.Body{Type: bt})
		if !b.HasBody() {
			t.Errorf("HasBody() should be true for %v", bt)
		}
	}
}

// TestBodyEditor_BinaryViewDoesNotWrapOnANarrowPanel guards against the
// word-wrap-instead-of-clip bug this session already hit more than once
// (lipgloss.Style.Width()/Place word-wraps over-budget content rather than
// clipping it) - the binary hint text is long enough to overflow a narrow
// Body tab if it isn't pre-wrapped to the actual available width, which
// silently grows the rendered box taller than requestPanelView budgeted
// and corrupts the whole panel's layout.
func TestBodyEditor_BinaryViewDoesNotWrapOnANarrowPanel(t *testing.T) {
	b := newBodyEditor()
	b.SetBody(collection.Body{Type: collection.BodyBinary})
	b.SetSize(24, 10)

	got := b.activeSubEditorView()
	if w := lipgloss.Width(got); w != 24 {
		t.Errorf("got width %d, want 24", w)
	}
	// binaryView draws its own border (unlike responseView's borderless
	// ContentView) - SetSize's height is the inner content budget, same
	// convention kvTable.SetHeight uses, so the rendered total is +2.
	if h := lipgloss.Height(got); h != 12 {
		t.Errorf("got height %d, want 12", h)
	}
}

// TestBodyEditor_CtrlPOnInvalidJSONIsANoOp guards against ctrl+p corrupting
// whatever's currently typed just because it doesn't parse yet (e.g.
// mid-edit, or a non-JSON content-type) - it should leave the text exactly
// as it was rather than erroring or clearing it.
func TestBodyEditor_CtrlPOnInvalidJSONIsANoOp(t *testing.T) {
	b := newBodyEditor()
	b.SetFocus(true)
	b.typeIdx = rawTypeIdx()
	b.area.SetValue(`{not valid json`)

	b, _, handled := b.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	if !handled {
		t.Fatal("expected ctrl+p to be handled by bodyEditor")
	}
	if got := b.area.Value(); got != `{not valid json` {
		t.Errorf("got %q, want the invalid text left untouched", got)
	}
}
