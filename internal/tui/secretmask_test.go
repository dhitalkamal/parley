package tui

import "testing"

func TestLooksLikeSecretKey_MatchesCaseInsensitiveSubstrings(t *testing.T) {
	cases := []struct {
		key  string
		want bool
	}{
		{"password", true},
		{"Password", true},
		{"api_token", true},
		{"authToken", true},
		{"clientSecret", true},
		{"Authorization", true},
		{"authorization", true},
		{"apiKey", true},
		{"api_key", true},
		{"x-api-key", true},
		{"access_key", true},
		{"client_id", true},
		{"Set-Cookie", true},
		{"Cookie", true},
		{"credential", true},
		{"username", false},
		{"Content-Length", false},
		{"Content-Type", false},
		{"Accept", false},
	}
	for _, c := range cases {
		if got := looksLikeSecretKey(c.key); got != c.want {
			t.Errorf("looksLikeSecretKey(%q) = %v, want %v", c.key, got, c.want)
		}
	}
}

func TestMaskedValue_MasksOnlySecretLookingKeysUnlessRevealed(t *testing.T) {
	if got := maskedValue("password", "hunter2", false); got != "****" {
		t.Errorf("got %q, want masked", got)
	}
	if got := maskedValue("password", "hunter2", true); got != "hunter2" {
		t.Errorf("got %q, want revealed", got)
	}
	if got := maskedValue("username", "ada", false); got != "ada" {
		t.Errorf("got %q, want non-secret key left untouched", got)
	}
	if got := maskedValue("password", "", false); got != "" {
		t.Errorf("got %q, want empty value left as empty rather than masked", got)
	}
}
