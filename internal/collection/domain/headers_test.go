package collection

import "testing"

func TestEffectiveHeaders_KeepsOnlyEnabled(t *testing.T) {
	headers := []Header{
		{Key: "Authorization", Value: "Bearer xyz", Enabled: true},
		{Key: "X-Debug", Value: "1", Enabled: false},
		{Key: "Content-Type", Value: "application/json", Enabled: true},
	}
	got := EffectiveHeaders(headers)
	want := []Header{
		{Key: "Authorization", Value: "Bearer xyz", Enabled: true},
		{Key: "Content-Type", Value: "application/json", Enabled: true},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d headers, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("header %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestEffectiveHeaders_EmptyInputReturnsEmpty(t *testing.T) {
	got := EffectiveHeaders(nil)
	if len(got) != 0 {
		t.Errorf("got %d headers, want 0", len(got))
	}
}
