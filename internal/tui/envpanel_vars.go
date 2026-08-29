package tui

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// envVarRow is one variable row in the env panel's editor. Value always
// holds the real value in memory (needed to actually persist it), but for a
// Secret row it is never re-rendered into an edit field - editing a secret
// always starts blank (blind-overwrite), so a saved secret never hits the
// screen or terminal scrollback a second time. See StartEdit/CommitEdit.
type envVarRow struct {
	Key     string
	Value   string
	Enabled bool
	Secret  bool
}

// SetRows loads rows into the editor's table, clamping the cursor onto the
// first row instead of leaving it at the table's zero-rows resting value.
func (p *envPanelState) SetRows(rows []envVarRow) {
	p.rows = rows
	p.refreshTable()
}

func (p envPanelState) Rows() []envVarRow {
	return p.rows
}

// refreshTable keeps the bubbles/table in sync with p.rows; it's built
// lazily the first time it's needed so a zero-value envPanelState (as used
// throughout the tests above) doesn't require a constructor.
func (p *envPanelState) refreshTable() {
	if p.tbl.Columns() == nil {
		p.tbl = table.New(
			table.WithColumns([]table.Column{
				{Title: "On", Width: 3},
				{Title: "Key", Width: 20},
				{Title: "Value", Width: 30},
				{Title: "Secret", Width: 6},
			}),
			table.WithHeight(6),
		)
	}
	trows := make([]table.Row, len(p.rows))
	for i, r := range p.rows {
		on, secret := " ", " "
		if r.Enabled {
			on = "x"
		}
		value := r.Value
		if r.Secret {
			secret = "x"
			value = "****"
		}
		trows[i] = table.Row{on, r.Key, value, secret}
	}
	p.tbl.SetRows(trows)
	if len(trows) > 0 && p.tbl.Cursor() < 0 {
		p.tbl.SetCursor(0)
	}
}

// AddRowWithValues appends a fully-formed row and moves the cursor onto it -
// the kvAdd modal (kvadd.go) already collected key/value from the user, so
// there's no inline edit form to open afterward.
func (p *envPanelState) AddRowWithValues(key, value string) {
	p.rows = append(p.rows, envVarRow{Key: key, Value: value, Enabled: true})
	p.refreshTable()
	p.tbl.SetCursor(len(p.rows) - 1)
}

// DeleteRow removes the currently selected row.
func (p *envPanelState) DeleteRow() {
	idx := p.tbl.Cursor()
	if idx < 0 || idx >= len(p.rows) {
		return
	}
	p.rows = append(p.rows[:idx], p.rows[idx+1:]...)
	p.refreshTable()
}

// ToggleEnabled flips the Enabled flag of the currently selected row.
func (p *envPanelState) ToggleEnabled() {
	idx := p.tbl.Cursor()
	if idx < 0 || idx >= len(p.rows) {
		return
	}
	p.rows[idx].Enabled = !p.rows[idx].Enabled
	p.refreshTable()
}

// ToggleSecret flips the Secret flag of the currently selected row. Toggling
// a plain variable to Secret doesn't blank its value - only editing it
// afterwards does, per the blind-overwrite rule in StartEdit/CommitEdit.
func (p *envPanelState) ToggleSecret() {
	idx := p.tbl.Cursor()
	if idx < 0 || idx >= len(p.rows) {
		return
	}
	p.rows[idx].Secret = !p.rows[idx].Secret
	p.refreshTable()
}

// StartAdd opens a blank edit form for a brand-new variable - the side rail's
// inline "add at the bottom" flow (see railenv_view.go). Unlike StartEdit it
// doesn't need a selected row; CommitEdit appends rather than overwrites while
// adding is set.
func (p *envPanelState) StartAdd() {
	p.editing = true
	p.adding = true
	p.editField = 0

	p.keyInput = textinput.New()
	p.keyInput.Prompt = ""
	p.valInput = textinput.New()
	p.valInput.Prompt = ""

	p.keyInput.Focus()
	p.valInput.Blur()
}

// SetEditWidth caps the two edit fields' visible width so they fit a narrow
// container (the rail); the ctrl+e modal leaves them at their default since it
// has room to spare.
func (p *envPanelState) SetEditWidth(w int) {
	if w < 1 {
		w = 1
	}
	p.keyInput.Width = w
	p.valInput.Width = w
}

// IsAdding reports whether the in-progress edit is a new-variable add rather
// than an edit of an existing row - lets the view label the form accordingly.
func (p envPanelState) IsAdding() bool { return p.adding }

// StartEdit opens the currently selected row for editing.
func (p *envPanelState) StartEdit() {
	idx := p.tbl.Cursor()
	if idx < 0 || idx >= len(p.rows) {
		return
	}
	row := p.rows[idx]
	p.editing = true
	p.adding = false
	p.editField = 0

	p.keyInput = textinput.New()
	p.keyInput.Prompt = ""
	p.keyInput.SetValue(row.Key)

	p.valInput = textinput.New()
	p.valInput.Prompt = ""
	// Blind-overwrite: a secret's stored value never gets loaded back into
	// an editable field, so it never re-appears on screen or in scrollback.
	if !row.Secret {
		p.valInput.SetValue(row.Value)
	}

	p.keyInput.Focus()
	p.valInput.Blur()
}

// CommitEdit writes the edit fields back to the selected row. A secret row
// left blank keeps its previously stored value (blind overwrite means "type
// a new value or leave it," never "clear it by leaving blank") - a plain
// row's value is written verbatim, blank or not.
func (p *envPanelState) CommitEdit() {
	if p.adding {
		// A blank key is a cancelled add, not an empty-named variable - drop
		// it rather than persist a nameless row.
		if p.keyInput.Value() != "" {
			p.rows = append(p.rows, envVarRow{
				Key:     p.keyInput.Value(),
				Value:   p.valInput.Value(),
				Enabled: true,
			})
			p.refreshTable()
			p.tbl.SetCursor(len(p.rows) - 1)
		}
		p.editing = false
		p.adding = false
		return
	}
	idx := p.tbl.Cursor()
	if idx >= 0 && idx < len(p.rows) {
		p.rows[idx].Key = p.keyInput.Value()
		if p.rows[idx].Secret && p.valInput.Value() == "" {
			// leave the stored value untouched
		} else {
			p.rows[idx].Value = p.valInput.Value()
		}
		p.refreshTable()
	}
	p.editing = false
}

// ToggleEditField switches which of the two edit fields (key/value) has
// focus, mirroring kvTable's identical tab behavior for params/headers.
func (p *envPanelState) ToggleEditField() {
	p.editField = (p.editField + 1) % 2
	if p.editField == 0 {
		p.keyInput.Focus()
		p.valInput.Blur()
	} else {
		p.valInput.Focus()
		p.keyInput.Blur()
	}
}

// UpdateEditField routes a key to whichever edit field currently has focus.
func (p envPanelState) UpdateEditField(msg tea.Msg) (envPanelState, tea.Cmd) {
	var cmd tea.Cmd
	if p.editField == 0 {
		p.keyInput, cmd = p.keyInput.Update(msg)
	} else {
		p.valInput, cmd = p.valInput.Update(msg)
	}
	return p, cmd
}

// Update forwards a key the panel's own handler doesn't own (cursor
// movement, etc) to the underlying table widget.
func (p envPanelState) Update(msg tea.Msg) (envPanelState, tea.Cmd) {
	var cmd tea.Cmd
	p.tbl, cmd = p.tbl.Update(msg)
	return p, cmd
}

// CancelEdit closes the edit form without committing - discarding an
// in-progress add entirely (nothing was appended yet).
func (p *envPanelState) CancelEdit() {
	p.editing = false
	p.adding = false
}

// SelectedIndex is the row the cursor is on - the side rail renders its own
// compact list rather than the bubbles table, so it reads the cursor directly
// to know which row to mark selected (see railenv_view.go).
func (p envPanelState) SelectedIndex() int {
	return p.tbl.Cursor()
}

// MoveSelection shifts the cursor by delta, clamped to the row list - the
// rail drives movement explicitly instead of forwarding to the table's own
// Update, which only moves the cursor while the table widget itself is
// focused (the rail never focuses it).
func (p *envPanelState) MoveSelection(delta int) {
	if len(p.rows) == 0 {
		return
	}
	idx := p.tbl.Cursor() + delta
	if idx < 0 {
		idx = 0
	}
	if idx > len(p.rows)-1 {
		idx = len(p.rows) - 1
	}
	p.tbl.SetCursor(idx)
}

// SetSelection moves the cursor to a specific row index, clamped to the list.
func (p *envPanelState) SetSelection(idx int) {
	p.MoveSelection(idx - p.tbl.Cursor())
}

// EditKeyView / EditValueView expose the two edit fields' rendered strings so
// the rail's compact editor can lay them out itself; IsEditing / EditField
// report the edit state the same view needs.
func (p envPanelState) EditKeyView() string   { return p.keyInput.View() }
func (p envPanelState) EditValueView() string { return p.valInput.View() }
func (p envPanelState) IsEditing() bool       { return p.editing }
func (p envPanelState) EditField() int        { return p.editField }
