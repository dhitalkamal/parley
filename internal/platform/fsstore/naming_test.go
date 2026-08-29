package fsstore

import "testing"

func TestValidateName_RejectsEmpty(t *testing.T) {
	if err := ValidateName(""); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestValidateName_RejectsPathSeparators(t *testing.T) {
	for _, name := range []string{"a/b", "a\\b", "..", "../escape"} {
		if err := ValidateName(name); err == nil {
			t.Errorf("expected error for name %q", name)
		}
	}
}

func TestValidateName_AcceptsOrdinaryName(t *testing.T) {
	if err := ValidateName("get-user"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
