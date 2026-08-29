package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// kvRow is one row of a key/value/enabled editor. Shared shape for both the
// query-param table and the header table.
type kvRow struct {
	Key         string
	Value       string
	Enabled     bool
	Description string
}

// kvTable is a reusable key/value/enabled editor used by both the params and
// headers panes: a table for browsing rows, plus an inline two-field form for
// editing the selected row.
type kvTable struct {
	title        string
	emptyMessage string
	rows         []kvRow
	tbl          table.Model
	editing      bool
	keyInput     textinput.Model
	valInput     textinput.Model
	descInput    textinput.Model
	editField    int // 0 = key, 1 = value, 2 = description
	focused      bool
	width        int
	height       int

	// maskSecrets opts this table into secret-looking-key masking (see
	// secretmask.go) - only the request body's Form/Multipart table sets
	// this; params/headers tables leave it false, since masking there was
	// never asked for and would be a surprising behavior change to a table
	// that already shows literal header/param values today.
	maskSecrets   bool
	revealSecrets bool

	// borderless renders the table flush (no inner border, no padding) so it
	// sits directly inside its parent panel's border instead of drawing a
	// second box-in-a-box. The Request panel's params/headers set this so they
	// match the Response body and the Auth/Settings/Body-None/Raw tabs, which
	// already render flush (see reqpanel.go).
	borderless bool
}

// SetBorderless opts this table into flush rendering (see borderless field).
func (kv *kvTable) SetBorderless(b bool) {
	kv.borderless = b
}

// SetMaskSecrets opts this table into masking secret-looking row values by
// default (see maskSecrets field doc).
func (kv *kvTable) SetMaskSecrets(mask bool) {
	kv.maskSecrets = mask
}

// SetRevealSecrets toggles whether masked rows show their literal value -
// a no-op on a table that never opted into masking via SetMaskSecrets.
func (kv *kvTable) SetRevealSecrets(reveal bool) {
	kv.revealSecrets = reveal
	kv.refreshRows()
}

func newKVTable(title, keyLabel, emptyMessage string) kvTable {
	cols := []table.Column{
		{Title: "#", Width: 2},
		{Title: "On", Width: 3},
		{Title: keyLabel, Width: 20},
		{Title: "Value", Width: 30},
		{Title: "Description", Width: 20},
	}
	t := table.New(table.WithColumns(cols), table.WithHeight(5))
	ki := textinput.New()
	ki.Placeholder = "key"
	vi := textinput.New()
	vi.Placeholder = "value"
	di := textinput.New()
	di.Placeholder = "description"
	kv := kvTable{title: title, emptyMessage: emptyMessage, tbl: t, keyInput: ki, valInput: vi, descInput: di}
	kv.refreshRows()
	return kv
}

func (kv *kvTable) SetRows(rows []kvRow) {
	kv.rows = rows
	kv.refreshRows()
}

func (kv kvTable) Rows() []kvRow {
	return kv.rows
}

func (kv *kvTable) refreshRows() {
	trows := make([]table.Row, len(kv.rows))
	for i, r := range kv.rows {
		on := " "
		if r.Enabled {
			on = "x"
		}
		value := r.Value
		if kv.maskSecrets {
			value = maskedValue(r.Key, r.Value, kv.revealSecrets)
		}
		trows[i] = table.Row{fmt.Sprintf("%d", i+1), on, r.Key, value, r.Description}
	}
	kv.tbl.SetRows(trows)
	// SetRows only clamps the cursor downward, so going from zero rows to
	// one leaves it at -1 (its zero-rows resting value) instead of 0.
	if len(trows) > 0 && kv.tbl.Cursor() < 0 {
		kv.tbl.SetCursor(0)
	}
}

func (kv *kvTable) SetFocus(focused bool) {
	kv.focused = focused
	if focused {
		kv.tbl.Focus()
	} else {
		kv.tbl.Blur()
		kv.editing = false
	}
}

func (kv *kvTable) SetHeight(h int) {
	kv.height = h
	kv.tbl.SetHeight(h)
}

// AddRow appends a fully-formed row and moves the cursor onto it - the
// kvAdd modal (kvadd.go) already collected key/value from the user, so
// there's no inline edit form to open afterward the way the old "a"
// keybinding used to.
func (kv *kvTable) AddRow(key, value string) {
	kv.rows = append(kv.rows, kvRow{Key: key, Value: value, Enabled: true})
	kv.refreshRows()
	kv.tbl.SetCursor(len(kv.rows) - 1)
}

// ExtraLines reports how many lines View adds on top of the bordered table
// itself - the caller (requestPanelView) needs this up front to shrink the
// table's own height budget by the same amount, or the extra lines (the
// key/value edit form) get silently clipped off the bottom of the panel by
// lipgloss's Height() instead of ever being seen. The empty-state message
// lives inside the table's own box (see emptyStateView), not appended below
// it, so it needs no extra lines of its own.
func (kv kvTable) ExtraLines() int {
	if kv.editing {
		return 3 // "Key: ... Value: ..." + "Description: ..." + "Tab switch field..." hint line
	}
	return 0
}

// Column order is #, On, Key, Value, Description (see newKVTable) - # and On
// stay fixed-width, and the remaining space splits 3:4:3 across Key:Value:
// Description, roughly matching how much each tends to actually need. The
// "-10" is each of the 5 columns' own 2-column cell padding (bubbles/table's
// default Cell style is Padding(0, 1)) - the same per-column fudge the old
// 3-column version subtracted as "-6" (2 x 3).
func (kv *kvTable) SetWidth(w int) {
	kv.width = w
	cols := kv.tbl.Columns()
	if len(cols) == 5 {
		remaining := w - cols[0].Width - cols[1].Width - 10
		if remaining < 15 {
			remaining = 15
		}
		cols[2].Width = remaining * 3 / 10
		cols[3].Width = remaining * 4 / 10
		cols[4].Width = remaining - cols[2].Width - cols[3].Width
		kv.tbl.SetColumns(cols)
	}
}

func (kv kvTable) startEdit() kvTable {
	idx := kv.tbl.Cursor()
	if idx < 0 || idx >= len(kv.rows) {
		return kv
	}
	kv.editing = true
	kv.editField = 0
	kv.keyInput.SetValue(kv.rows[idx].Key)
	kv.valInput.SetValue(kv.rows[idx].Value)
	kv.descInput.SetValue(kv.rows[idx].Description)
	kv.keyInput.Focus()
	kv.valInput.Blur()
	kv.descInput.Blur()
	return kv
}

func (kv kvTable) commitEdit() kvTable {
	idx := kv.tbl.Cursor()
	if idx >= 0 && idx < len(kv.rows) {
		kv.rows[idx].Key = kv.keyInput.Value()
		kv.rows[idx].Value = kv.valInput.Value()
		kv.rows[idx].Description = kv.descInput.Value()
		kv.refreshRows()
	}
	kv.editing = false
	return kv
}

// Update handles input when the table has focus. It returns handled=false
// when the key wasn't consumed here, so the caller can fall through to
// global keybindings (e.g. Tab to change focus zone).
func (kv kvTable) Update(msg tea.Msg) (kvTable, tea.Cmd, bool) {
	if kv.editing {
		return kv.updateEditing(msg)
	}

	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return kv, nil, false
	}
	switch k.String() {
	case "d":
		idx := kv.tbl.Cursor()
		if idx >= 0 && idx < len(kv.rows) {
			kv.rows = append(kv.rows[:idx], kv.rows[idx+1:]...)
			kv.refreshRows()
		}
		return kv, nil, true
	case " ":
		idx := kv.tbl.Cursor()
		if idx >= 0 && idx < len(kv.rows) {
			kv.rows[idx].Enabled = !kv.rows[idx].Enabled
			kv.refreshRows()
		}
		return kv, nil, true
	case "enter":
		if len(kv.rows) > 0 {
			return kv.startEdit(), nil, true
		}
		return kv, nil, true
	}

	var cmd tea.Cmd
	kv.tbl, cmd = kv.tbl.Update(msg)
	return kv, cmd, true
}

func (kv kvTable) updateEditing(msg tea.Msg) (kvTable, tea.Cmd, bool) {
	k, ok := msg.(tea.KeyMsg)
	if ok {
		switch k.String() {
		case "esc":
			kv.editing = false
			return kv, nil, true
		case "enter":
			return kv.commitEdit(), nil, true
		case "tab":
			kv.editField = (kv.editField + 1) % 3
			kv.keyInput.Blur()
			kv.valInput.Blur()
			kv.descInput.Blur()
			switch kv.editField {
			case 0:
				kv.keyInput.Focus()
			case 1:
				kv.valInput.Focus()
			case 2:
				kv.descInput.Focus()
			}
			return kv, nil, true
		}
	}
	var cmd tea.Cmd
	switch kv.editField {
	case 0:
		kv.keyInput, cmd = kv.keyInput.Update(msg)
	case 1:
		kv.valInput, cmd = kv.valInput.Update(msg)
	case 2:
		kv.descInput, cmd = kv.descInput.Update(msg)
	}
	return kv, cmd, true
}

func (kv kvTable) View() string {
	style := borderStyle
	if kv.focused {
		style = focusedBorder
	}
	// borderless: render flush against the parent panel's own border (its
	// focus is already shown there), no second box, no padding.
	if kv.borderless {
		style = lipgloss.NewStyle()
	}
	if kv.editing {
		return style.Render(kv.tbl.View()) + "\n" +
			labelStyle.Render("Key: ") + kv.keyInput.View() + "  " +
			labelStyle.Render("Value: ") + kv.valInput.View() + "\n" +
			labelStyle.Render("Description: ") + kv.descInput.View() + "\n" +
			labelStyle.Render("Tab switch field - Enter save - Esc cancel")
	}
	if len(kv.rows) == 0 {
		return kv.emptyStateView(style)
	}
	return style.Render(kv.tbl.View())
}

// emptyStateView replaces the table with a short explanation of what these
// rows are for, plus the add hint, instead of a bare column-headers table
// stretched to fill the whole panel with dozens of blank rows and a hint
// tucked away at the bottom - that read as broken, not as "empty and ready
// to use." The message is centered within the table's own given dimensions
// (matching response.go's emptyStateView) rather than pinned to the top-left
// corner - a user flagged the top-left version by screenshot as looking
// unfinished next to the Response panel's already-centered treatment. The
// explanation text is pre-wrapped to the table's own known width here,
// rather than left to style.Render()/lipgloss.Place: both word-wrap (rather
// than clip) content wider than the box they're given, which would corrupt
// the layout on a narrow panel (see reqpanel.go's notes on that) - so the
// wrapping has to happen on the raw text before it ever reaches either.
func (kv kvTable) emptyStateView(style lipgloss.Style) string {
	// chrome is what the wrapping style adds on each side: 4 for a bordered
	// table (border 2 + horizontal padding 2), 0 when rendering flush.
	chrome := 4
	if kv.borderless {
		chrome = 0
	}
	textWidth := kv.width - chrome
	if textWidth < 10 {
		textWidth = 10
	}
	innerHeight := kv.height
	if innerHeight < 3 {
		innerHeight = 3
	}
	wrapped := ansi.Wordwrap(kv.emptyMessage, textWidth, "")
	msg := labelStyle.Render(wrapped) + "\n\n" + labelStyle.Render("a: add row")
	// Clip, don't let Place silently grow past its own budget: on a narrow
	// enough panel, wordwrap above can spread kv.emptyMessage across more
	// lines than innerHeight budgets for, and lipgloss.Place only pads
	// short content - it never truncates tall content, the same class of
	// bug reqpanel.go's own notes on this panel describe.
	if lines := strings.Split(msg, "\n"); len(lines) > innerHeight {
		msg = strings.Join(lines[:innerHeight], "\n")
	}
	placed := lipgloss.Place(textWidth, innerHeight, lipgloss.Center, lipgloss.Center, msg)
	return style.Render(placed)
}
