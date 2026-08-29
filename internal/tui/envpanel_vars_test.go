package tui

import "testing"

func TestEnvPanel_SetRowsAndRowsRoundTrip(t *testing.T) {
	var p envPanelState
	want := []envVarRow{{Key: "a", Value: "1", Enabled: true}}
	p.SetRows(want)
	got := p.Rows()
	if len(got) != 1 || got[0] != want[0] {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

// TestEnvPanel_AddRowWithValuesAppendsWithoutEnteringEditMode guards the
// replacement for the old AddRow flow: the kvAdd modal (kvadd.go) already
// collected key/value from the user, so this just appends a finished row -
// no inline edit form, and so no "isNewRow" discard-on-cancel concept
// needed anymore either.
func TestEnvPanel_AddRowWithValuesAppendsWithoutEnteringEditMode(t *testing.T) {
	var p envPanelState
	p.SetRows([]envVarRow{{Key: "existing", Value: "v"}})
	p.AddRowWithValues("apiKey", "secret123")

	got := p.Rows()
	if len(got) != 2 {
		t.Fatalf("got %d rows, want 2", len(got))
	}
	if got[1].Key != "apiKey" || got[1].Value != "secret123" || !got[1].Enabled {
		t.Errorf("got %+v, want Key=apiKey Value=secret123 Enabled=true", got[1])
	}
	if p.editing {
		t.Error("AddRowWithValues should not enter the inline edit form")
	}
}

func TestEnvPanel_CancelEditOnExistingRowKeepsIt(t *testing.T) {
	var p envPanelState
	p.SetRows([]envVarRow{{Key: "k", Value: "v"}})
	p.StartEdit()
	p.CancelEdit()

	if len(p.Rows()) != 1 {
		t.Errorf("got %d rows, want the existing row kept", len(p.Rows()))
	}
}

func TestEnvPanel_DeleteRowRemovesSelected(t *testing.T) {
	var p envPanelState
	p.SetRows([]envVarRow{{Key: "a"}, {Key: "b"}})
	p.DeleteRow()

	got := p.Rows()
	if len(got) != 1 || got[0].Key != "b" {
		t.Errorf("got %+v, want only \"b\" left", got)
	}
}

func TestEnvPanel_ToggleEnabledFlipsSelectedRow(t *testing.T) {
	var p envPanelState
	p.SetRows([]envVarRow{{Key: "a", Enabled: true}})
	p.ToggleEnabled()
	if p.Rows()[0].Enabled {
		t.Error("expected Enabled to flip to false")
	}
}

func TestEnvPanel_ToggleSecretFlipsSelectedRow(t *testing.T) {
	var p envPanelState
	p.SetRows([]envVarRow{{Key: "a", Secret: false}})
	p.ToggleSecret()
	if !p.Rows()[0].Secret {
		t.Error("expected Secret to flip to true")
	}
}

func TestEnvPanel_StartEditPrefillsPlainValueButNotSecretValue(t *testing.T) {
	var p envPanelState
	p.SetRows([]envVarRow{
		{Key: "plain", Value: "visible", Secret: false},
	})
	p.StartEdit()
	if got := p.keyInput.Value(); got != "plain" {
		t.Errorf("key: got %q, want %q", got, "plain")
	}
	if got := p.valInput.Value(); got != "visible" {
		t.Errorf("value: got %q, want the real value prefilled for a non-secret row", got)
	}
}

func TestEnvPanel_StartEditNeverPrefillsAStoredSecretValue(t *testing.T) {
	var p envPanelState
	p.SetRows([]envVarRow{
		{Key: "token", Value: "super-secret-value", Secret: true},
	})
	p.StartEdit()
	if got := p.valInput.Value(); got != "" {
		t.Errorf("value: got %q, want blank - a stored secret must never be re-displayed", got)
	}
}

func TestEnvPanel_CommitEditOnNonSecretAlwaysOverwrites(t *testing.T) {
	var p envPanelState
	p.SetRows([]envVarRow{{Key: "k", Value: "old", Secret: false}})
	p.StartEdit()
	p.valInput.SetValue("")
	p.CommitEdit()

	if got := p.Rows()[0].Value; got != "" {
		t.Errorf("got %q, want the field cleared like any other row (not a secret)", got)
	}
}

func TestEnvPanel_CommitEditOnSecretWithBlankInputKeepsStoredValue(t *testing.T) {
	var p envPanelState
	p.SetRows([]envVarRow{{Key: "token", Value: "still-the-old-secret", Secret: true}})
	p.StartEdit() // valInput starts blank, per the blind-overwrite rule
	p.CommitEdit()

	if got := p.Rows()[0].Value; got != "still-the-old-secret" {
		t.Errorf("got %q, want the stored secret preserved when left blank", got)
	}
}

func TestEnvPanel_CommitEditOnSecretWithNewInputOverwrites(t *testing.T) {
	var p envPanelState
	p.SetRows([]envVarRow{{Key: "token", Value: "old-secret", Secret: true}})
	p.StartEdit()
	p.valInput.SetValue("new-secret")
	p.CommitEdit()

	if got := p.Rows()[0].Value; got != "new-secret" {
		t.Errorf("got %q, want the new value to overwrite the stored secret", got)
	}
}

func TestEnvPanel_CommitEditUpdatesKeyRegardlessOfSecret(t *testing.T) {
	var p envPanelState
	p.SetRows([]envVarRow{{Key: "old-name", Value: "v", Secret: true}})
	p.StartEdit()
	p.keyInput.SetValue("new-name")
	p.CommitEdit()

	if got := p.Rows()[0].Key; got != "new-name" {
		t.Errorf("got %q, want key renamed", got)
	}
}
