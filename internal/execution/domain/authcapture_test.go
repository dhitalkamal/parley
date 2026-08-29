package execution

import (
	"testing"
	"time"
)

func TestExtractJSONField_TopLevelString(t *testing.T) {
	got, ok := ExtractJSONField([]byte(`{"token":"abc123"}`), "token")
	if !ok || got != "abc123" {
		t.Errorf("got (%q, %v), want (abc123, true)", got, ok)
	}
}

func TestExtractJSONField_NestedDotPath(t *testing.T) {
	got, ok := ExtractJSONField([]byte(`{"data":{"tokens":{"access":"xyz"}}}`), "data.tokens.access")
	if !ok || got != "xyz" {
		t.Errorf("got (%q, %v), want (xyz, true)", got, ok)
	}
}

func TestExtractJSONField_NumberFormatsWithoutDecimalWhenWhole(t *testing.T) {
	got, ok := ExtractJSONField([]byte(`{"expiresIn":3600}`), "expiresIn")
	if !ok || got != "3600" {
		t.Errorf("got (%q, %v), want (3600, true)", got, ok)
	}
}

func TestExtractJSONField_MissingPathReturnsFalse(t *testing.T) {
	if _, ok := ExtractJSONField([]byte(`{"token":"abc"}`), "missing"); ok {
		t.Error("expected ok=false for a missing field")
	}
}

func TestExtractJSONField_PathThroughNonObjectReturnsFalse(t *testing.T) {
	if _, ok := ExtractJSONField([]byte(`{"token":"abc"}`), "token.nested"); ok {
		t.Error("expected ok=false when the path continues past a leaf value")
	}
}

func TestExtractJSONField_InvalidJSONReturnsFalse(t *testing.T) {
	if _, ok := ExtractJSONField([]byte(`not json`), "token"); ok {
		t.Error("expected ok=false for invalid JSON")
	}
}

func TestExtractJSONField_EmptyPathReturnsFalse(t *testing.T) {
	if _, ok := ExtractJSONField([]byte(`{"token":"abc"}`), ""); ok {
		t.Error("expected ok=false for an empty path")
	}
}

func TestComputeExpiresAt_AddsSecondsToNow(t *testing.T) {
	now := time.Unix(1000, 0)
	got, ok := ComputeExpiresAt(now, "3600")
	if !ok || got != "4600" {
		t.Errorf("got (%q, %v), want (4600, true)", got, ok)
	}
}

func TestComputeExpiresAt_InvalidSecondsReturnsFalse(t *testing.T) {
	if _, ok := ComputeExpiresAt(time.Unix(1000, 0), "not-a-number"); ok {
		t.Error("expected ok=false for a non-numeric seconds value")
	}
}

func TestIsExpired_TrueWhenNowAtOrPastExpiry(t *testing.T) {
	if !IsExpired("1000", time.Unix(1000, 0)) {
		t.Error("expected expired at exactly the expiry time")
	}
	if !IsExpired("1000", time.Unix(1001, 0)) {
		t.Error("expected expired after the expiry time")
	}
}

func TestIsExpired_FalseWhenNowBeforeExpiry(t *testing.T) {
	if IsExpired("1000", time.Unix(999, 0)) {
		t.Error("expected not expired before the expiry time")
	}
}

func TestIsExpired_FalseForUnparsableOrEmptyExpiry(t *testing.T) {
	if IsExpired("", time.Unix(1000, 0)) {
		t.Error("expected not expired when there's no expiry recorded at all")
	}
	if IsExpired("garbage", time.Unix(1000, 0)) {
		t.Error("expected not expired when the recorded expiry can't be parsed")
	}
}
