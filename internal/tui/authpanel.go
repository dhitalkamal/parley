package tui

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Field indices into authEditor.fields - see authFieldLabels for what each
// one means to the user.
const (
	authFieldTokenField = iota
	authFieldTokenVar
	authFieldExpiresInField
	authFieldExpiresAtVar
	authFieldRefreshPath
	authFieldRefreshExpiresAtVar
	authFieldCount
)

// authFieldLabels are shown one per row in View() - see collection.AuthCapture/
// collection.RefreshConfig for what each field does. Written as plain text
// fields (not a modal, not a picker) rather than building a new UI
// paradigm just for this: RefreshPath is the exact path already visible in
// the sidebar, and the JSON paths are dot-notation the user types once
// after a login response, the same shape as any other saved field.
var authFieldLabels = [authFieldCount]string{
	"Token field (response JSON path)",
	"Save token as variable",
	"Expires-in field (seconds, optional)",
	"Save expiry as variable (optional)",
	"Refresh request path (optional)",
	"Expiry variable to check before sending (optional)",
}

// authSection groups authFieldLabels' indices under one collapsible header -
// Token Capture (extracting a token from this request's own response) and
// Refresh Flow (proactively renewing it via another saved request) are each
// opt-in details a request may not use at all, so they start collapsed
// rather than always showing all 6 fields at once (see View()).
type authSection struct {
	title  string
	fields []int
}

var authSections = []authSection{
	{title: "Token Capture", fields: []int{authFieldTokenField, authFieldTokenVar, authFieldExpiresInField, authFieldExpiresAtVar}},
	{title: "Refresh Flow", fields: []int{authFieldRefreshPath, authFieldRefreshExpiresAtVar}},
}

// authRow is one entry in the editor's current visible-row list (see
// visibleRows) - either a section header (field == -1) or one of that
// section's fields, only present while the section is open.
type authRow struct {
	section int
	field   int // -1 for a header row
}

// authEditor configures a request's AuthCapture and RefreshConfig via two
// collapsible sections instead of 6 always-visible fields - see authSection.
type authEditor struct {
	fields  [authFieldCount]textinput.Model
	open    []bool // one per authSections entry
	cursor  int    // index into visibleRows()
	focused bool
}

func newAuthEditor() authEditor {
	var a authEditor
	for i := range a.fields {
		ti := textinput.New()
		ti.Prompt = ""
		a.fields[i] = ti
	}
	a.open = make([]bool, len(authSections))
	return a
}

func (a authEditor) AuthCapture() collection.AuthCapture {
	return collection.AuthCapture{
		TokenField:     a.fields[authFieldTokenField].Value(),
		TokenVar:       a.fields[authFieldTokenVar].Value(),
		ExpiresInField: a.fields[authFieldExpiresInField].Value(),
		ExpiresAtVar:   a.fields[authFieldExpiresAtVar].Value(),
	}
}

func (a authEditor) RefreshConfig() collection.RefreshConfig {
	return collection.RefreshConfig{
		RequestPath:  a.fields[authFieldRefreshPath].Value(),
		ExpiresAtVar: a.fields[authFieldRefreshExpiresAtVar].Value(),
	}
}

func (a *authEditor) SetAuth(capture collection.AuthCapture, refresh collection.RefreshConfig) {
	a.fields[authFieldTokenField].SetValue(capture.TokenField)
	a.fields[authFieldTokenVar].SetValue(capture.TokenVar)
	a.fields[authFieldExpiresInField].SetValue(capture.ExpiresInField)
	a.fields[authFieldExpiresAtVar].SetValue(capture.ExpiresAtVar)
	a.fields[authFieldRefreshPath].SetValue(refresh.RequestPath)
	a.fields[authFieldRefreshExpiresAtVar].SetValue(refresh.ExpiresAtVar)
}

// visibleRows is the editor's current row list: each section's header,
// followed by that section's fields only while it's open - what View()
// renders and what the cursor/up/down move through.
func (a authEditor) visibleRows() []authRow {
	var rows []authRow
	for si, sec := range authSections {
		rows = append(rows, authRow{section: si, field: -1})
		if a.open[si] {
			for _, f := range sec.fields {
				rows = append(rows, authRow{section: si, field: f})
			}
		}
	}
	return rows
}

func (a *authEditor) SetFocus(focused bool) {
	a.focused = focused
	for i := range a.fields {
		a.fields[i].Blur()
	}
	if focused {
		a.focusCursorField()
	}
}

// focusCursorField focuses whichever field the cursor currently points at -
// a no-op if it's resting on a header row (headers have nothing to type
// into).
func (a *authEditor) focusCursorField() {
	rows := a.visibleRows()
	if a.cursor < 0 || a.cursor >= len(rows) {
		return
	}
	if f := rows[a.cursor].field; f >= 0 {
		a.fields[f].Focus()
	}
}

func (a *authEditor) SetSize(w, h int) {
	for i := range a.fields {
		a.fields[i].Width = w - lipgloss.Width(longestAuthLabel()) - 2
	}
}

func longestAuthLabel() string {
	longest := ""
	for _, l := range authFieldLabels {
		if len(l) > len(longest) {
			longest = l
		}
	}
	return longest
}

// Update moves the cursor with up/down (wrapping over only the currently
// visible rows), toggles a section open/closed with enter/space on its
// header, and otherwise forwards to whichever field the cursor is on. Tab
// is deliberately NOT used here - it's bound globally to NextFocus (see
// root.go's handleKey), the same reason scriptEditor's pre-request/test
// toggle uses ctrl+b rather than Tab.
func (a authEditor) Update(msg tea.Msg) (authEditor, tea.Cmd, bool) {
	if k, ok := msg.(tea.KeyMsg); ok {
		rows := a.visibleRows()
		switch k.String() {
		case "down":
			for i := range a.fields {
				a.fields[i].Blur()
			}
			a.cursor = (a.cursor + 1) % len(rows)
			a.focusCursorField()
			return a, nil, true
		case "up":
			for i := range a.fields {
				a.fields[i].Blur()
			}
			a.cursor = (a.cursor - 1 + len(rows)) % len(rows)
			a.focusCursorField()
			return a, nil, true
		case "enter", " ":
			if a.cursor >= 0 && a.cursor < len(rows) && rows[a.cursor].field == -1 {
				si := rows[a.cursor].section
				a.open[si] = !a.open[si]
				return a, nil, true
			}
		}
	}
	rows := a.visibleRows()
	if a.cursor < 0 || a.cursor >= len(rows) || rows[a.cursor].field < 0 {
		return a, nil, true
	}
	var cmd tea.Cmd
	a.fields[rows[a.cursor].field], cmd = a.fields[rows[a.cursor].field].Update(msg)
	return a, cmd, true
}

func (a authEditor) View() string {
	rows := a.visibleRows()
	rendered := make([]string, len(rows))
	for i, row := range rows {
		marker := focusMarker(a.focused && a.cursor == i)
		if row.field == -1 {
			title := authSections[row.section].title
			// zoneChevron's own v/> convention (see workspace_view.go) -
			// this app's one existing expand/collapse indicator, reused
			// here instead of introducing a second one.
			rendered[i] = marker + zoneChevron(a.open[row.section]) + " " + activeTabStyle.Render(title)
			continue
		}
		label := authFieldLabels[row.field]
		rendered[i] = marker + "  " + labelStyle.Render(label+": ") + a.fields[row.field].View()
	}
	rendered = append(rendered, "", labelStyle.Render("up/down: navigate - enter/space: expand section"))
	return lipgloss.JoinVertical(lipgloss.Left, rendered...)
}
