package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// kvAddKind is what a completed kvAddState should create - a param row, a
// header row, or an environment variable. One small modal serves all three
// instead of each having its own inline add-form rendered wherever its
// table happens to sit, which a user flagged as confusing (different key,
// no feedback, no consistent shape across the app).
type kvAddKind int

const (
	kvAddParam kvAddKind = iota
	kvAddHeader
	kvAddVariable
)

func (k kvAddKind) title() string {
	switch k {
	case kvAddHeader:
		return "Add header"
	case kvAddVariable:
		return "Add variable"
	default:
		return "Add param"
	}
}

// kvAddState is the one modal every "add a key/value" action in the app
// opens - see kvAddKind for which table it writes the result into.
type kvAddState struct {
	active   bool
	kind     kvAddKind
	field    int // 0 = key field focused, 1 = value field focused
	keyInput textinput.Model
	valInput textinput.Model
}

func newKVAddState() kvAddState {
	ki := textinput.New()
	ki.Prompt = ""
	ki.Placeholder = "key"
	vi := textinput.New()
	vi.Prompt = ""
	vi.Placeholder = "value"
	return kvAddState{keyInput: ki, valInput: vi}
}

func (a *kvAddState) Open(kind kvAddKind) {
	a.active = true
	a.kind = kind
	a.field = 0
	a.keyInput.SetValue("")
	a.valInput.SetValue("")
	a.keyInput.Focus()
	a.valInput.Blur()
}

func (a *kvAddState) Close() {
	a.active = false
	a.keyInput.Blur()
	a.valInput.Blur()
}

func (a *kvAddState) ToggleField() {
	a.field = (a.field + 1) % 2
	if a.field == 0 {
		a.keyInput.Focus()
		a.valInput.Blur()
	} else {
		a.valInput.Focus()
		a.keyInput.Blur()
	}
}

// Update forwards a key to whichever field currently has focus.
func (a kvAddState) Update(msg tea.Msg) (kvAddState, tea.Cmd) {
	var cmd tea.Cmd
	if a.field == 0 {
		a.keyInput, cmd = a.keyInput.Update(msg)
	} else {
		a.valInput, cmd = a.valInput.Update(msg)
	}
	return a, cmd
}

// View renders the same centered modal shape regardless of kind - only the
// title changes.
func (a kvAddState) View() string {
	content := labelStyle.Render(a.kind.title()) + "\n\n" +
		labelStyle.Render("Key: ") + a.keyInput.View() + "\n" +
		labelStyle.Render("Value: ") + a.valInput.View() +
		"\n\n" + labelStyle.Render("tab switch field - enter add - esc cancel")
	return modalStyle.Width(modalWidth).Render(content)
}

// handleKVAddKey handles input while the kvAdd modal is open.
func (m Model) handleKVAddKey(k tea.KeyMsg) (Model, tea.Cmd) {
	switch k.String() {
	case "esc":
		m.kvAdd.Close()
		return m, nil
	case "tab":
		m.kvAdd.ToggleField()
		return m, nil
	case "enter":
		return m.submitKVAdd()
	}
	var cmd tea.Cmd
	m.kvAdd, cmd = m.kvAdd.Update(k)
	return m, cmd
}

// submitKVAdd writes the completed key/value into whichever table kvAdd.kind
// points at - params, headers, or the currently-selected environment scope's
// variables (persisted immediately, matching how every other edit in the
// environments panel saves as it goes).
func (m Model) submitKVAdd() (Model, tea.Cmd) {
	key := m.kvAdd.keyInput.Value()
	value := m.kvAdd.valInput.Value()
	kind := m.kvAdd.kind
	m.kvAdd.Close()

	switch kind {
	case kvAddParam:
		m.params.AddRow(key, value)
	case kvAddHeader:
		m.headers.AddRow(key, value)
	case kvAddVariable:
		m.envPanel.AddRowWithValues(key, value)
		m.saveEnvPanelScope()
	}
	return m, nil
}
