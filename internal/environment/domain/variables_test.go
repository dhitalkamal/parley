package environment

import "testing"

func TestApplyVariableUpdates_UpdatesExistingValuePreservingSecretAndEnabled(t *testing.T) {
	existing := []Variable{
		{Key: "token", Value: "old", Enabled: true, Secret: true},
	}
	got := ApplyVariableUpdates(existing, map[string]string{"token": "new"})
	if len(got) != 1 {
		t.Fatalf("got %+v, want 1 entry", got)
	}
	if got[0].Value != "new" {
		t.Errorf("value = %q, want new", got[0].Value)
	}
	if !got[0].Secret || !got[0].Enabled {
		t.Errorf("expected Secret/Enabled preserved, got %+v", got[0])
	}
}

func TestApplyVariableUpdates_AppendsNewKeyAsEnabledNonSecret(t *testing.T) {
	got := ApplyVariableUpdates(nil, map[string]string{"fresh": "value"})
	if len(got) != 1 {
		t.Fatalf("got %+v, want 1 entry", got)
	}
	if got[0].Key != "fresh" || got[0].Value != "value" || got[0].Secret || !got[0].Enabled {
		t.Errorf("got %+v", got[0])
	}
}

func TestApplyVariableUpdates_NewKeysAppendedInSortedOrder(t *testing.T) {
	got := ApplyVariableUpdates(nil, map[string]string{"zebra": "1", "alpha": "2", "mango": "3"})
	if len(got) != 3 {
		t.Fatalf("got %+v", got)
	}
	if got[0].Key != "alpha" || got[1].Key != "mango" || got[2].Key != "zebra" {
		t.Errorf("order = [%s, %s, %s], want [alpha, mango, zebra]", got[0].Key, got[1].Key, got[2].Key)
	}
}

func TestApplyVariableUpdates_LeavesUntouchedVariablesAlone(t *testing.T) {
	existing := []Variable{
		{Key: "keepme", Value: "unchanged", Enabled: true},
	}
	got := ApplyVariableUpdates(existing, map[string]string{"other": "x"})
	if len(got) != 2 {
		t.Fatalf("got %+v", got)
	}
	if got[0].Key != "keepme" || got[0].Value != "unchanged" {
		t.Errorf("existing variable mutated: %+v", got[0])
	}
}

func TestApplyVariableUpdates_EmptyUpdatesReturnsEquivalentList(t *testing.T) {
	existing := []Variable{{Key: "a", Value: "1", Enabled: true}}
	got := ApplyVariableUpdates(existing, nil)
	if len(got) != 1 || got[0] != existing[0] {
		t.Errorf("got %+v, want unchanged %+v", got, existing)
	}
}

func TestApplyVariableUpdates_DoesNotMutateOriginalSlice(t *testing.T) {
	existing := []Variable{{Key: "a", Value: "1", Enabled: true}}
	_ = ApplyVariableUpdates(existing, map[string]string{"a": "2"})
	if existing[0].Value != "1" {
		t.Errorf("original slice was mutated: %+v", existing)
	}
}
