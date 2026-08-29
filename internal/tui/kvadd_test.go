package tui

import (
	"strings"
	"testing"
)

// TestKVAddState_ViewShowsTitleForEachKind guards the whole point of this
// type: one modal shape reused for params, headers, and variables, with
// only the title differing - not three separate inline forms rendered
// wherever their table happens to sit.
func TestKVAddState_ViewShowsTitleForEachKind(t *testing.T) {
	cases := []struct {
		kind kvAddKind
		want string
	}{
		{kvAddParam, "Add param"},
		{kvAddHeader, "Add header"},
		{kvAddVariable, "Add variable"},
	}
	for _, c := range cases {
		a := newKVAddState()
		a.Open(c.kind)
		got := stripANSI(a.View())
		if !strings.Contains(got, c.want) {
			t.Errorf("kind %v: View() = %q, want it to contain %q", c.kind, got, c.want)
		}
	}
}

func TestKVAddState_OpenFocusesKeyField(t *testing.T) {
	a := newKVAddState()
	a.Open(kvAddParam)
	if !a.keyInput.Focused() {
		t.Error("Open should focus the key field first")
	}
	if a.valInput.Focused() {
		t.Error("Open should leave the value field blurred")
	}
}

func TestKVAddState_ToggleFieldSwitchesFocus(t *testing.T) {
	a := newKVAddState()
	a.Open(kvAddParam)
	a.ToggleField()
	if a.keyInput.Focused() {
		t.Error("after ToggleField, key field should be blurred")
	}
	if !a.valInput.Focused() {
		t.Error("after ToggleField, value field should be focused")
	}
}

func TestKVAddState_CloseResetsFields(t *testing.T) {
	a := newKVAddState()
	a.Open(kvAddParam)
	a.keyInput.SetValue("leftover")
	a.Close()
	if a.active {
		t.Error("Close should deactivate the modal")
	}
	a.Open(kvAddHeader)
	if a.keyInput.Value() != "" {
		t.Errorf("key field = %q, want empty on a fresh Open after Close", a.keyInput.Value())
	}
}
