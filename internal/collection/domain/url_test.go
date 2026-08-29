package collection

import "testing"

func TestBuildURL_MergesEnabledParams(t *testing.T) {
	params := []QueryParam{
		{Key: "page", Value: "2", Enabled: true},
		{Key: "limit", Value: "10", Enabled: true},
	}
	got, err := BuildURL("https://api.example.com/users", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "https://api.example.com/users?page=2&limit=10"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildURL_SkipsDisabledParams(t *testing.T) {
	params := []QueryParam{
		{Key: "page", Value: "2", Enabled: true},
		{Key: "debug", Value: "1", Enabled: false},
	}
	got, err := BuildURL("https://api.example.com/users", params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "https://api.example.com/users?page=2"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildURL_NoParamsStripsExistingQuery(t *testing.T) {
	got, err := BuildURL("https://api.example.com/users?stale=1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "https://api.example.com/users"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildURL_PreservesUnresolvedTemplatePlaceholders(t *testing.T) {
	got, err := BuildURL("https://api.example.com/users/{{id}}", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "https://api.example.com/users/{{id}}"
	if got != want {
		t.Errorf("got %q, want %q - {{...}} placeholders must not be percent-encoded", got, want)
	}
}

func TestBuildURL_InvalidURLReturnsError(t *testing.T) {
	_, err := BuildURL(":not a url:", nil)
	if err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
}

func TestParseQueryParams_ExtractsExistingQuery(t *testing.T) {
	got := ParseQueryParams("https://api.example.com/users?page=2&limit=10")
	want := []QueryParam{
		{Key: "page", Value: "2", Enabled: true},
		{Key: "limit", Value: "10", Enabled: true},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d params, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("param %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestParseQueryParams_NoQueryReturnsEmpty(t *testing.T) {
	got := ParseQueryParams("https://api.example.com/users")
	if len(got) != 0 {
		t.Errorf("got %d params, want 0: %+v", len(got), got)
	}
}

func TestParseQueryParams_InvalidURLReturnsEmpty(t *testing.T) {
	got := ParseQueryParams(":not a url:")
	if len(got) != 0 {
		t.Errorf("got %d params, want 0: %+v", len(got), got)
	}
}
