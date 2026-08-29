package importexport

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	"strings"
	"testing"
)

func TestGenerateCurl_IncludesMethodURLHeadersAndBody(t *testing.T) {
	req := collection.Request{
		Method: collection.POST,
		URL:    "https://api.example.com/users",
		Headers: []collection.Header{
			{Key: "Authorization", Value: "Bearer xyz", Enabled: true},
			{Key: "X-Skip", Value: "nope", Enabled: false},
		},
		Body: collection.Body{Type: collection.BodyRaw, RawContentType: "application/json", RawText: `{"name":"ada"}`},
	}
	got := GenerateCurl(req)
	if !strings.Contains(got, "-X POST") {
		t.Errorf("missing method, got:\n%s", got)
	}
	if !strings.Contains(got, "'https://api.example.com/users'") {
		t.Errorf("missing URL, got:\n%s", got)
	}
	if !strings.Contains(got, "-H 'Authorization: Bearer xyz'") {
		t.Errorf("missing enabled header, got:\n%s", got)
	}
	if strings.Contains(got, "X-Skip") {
		t.Errorf("disabled header should be omitted, got:\n%s", got)
	}
	if !strings.Contains(got, `-d '{"name":"ada"}'`) {
		t.Errorf("missing body, got:\n%s", got)
	}
}

func TestGenerateCurl_AutoInjectsContentTypeFromBodyWhenNotExplicit(t *testing.T) {
	req := collection.Request{
		Method: collection.POST,
		URL:    "https://api.example.com/users",
		Body:   collection.Body{Type: collection.BodyRaw, RawContentType: "application/json", RawText: `{"name":"ada"}`},
	}
	got := GenerateCurl(req)
	if !strings.Contains(got, "-H 'Content-Type: application/json'") {
		t.Errorf("expected auto-injected Content-Type header matching what httpclient actually sends, got:\n%s", got)
	}
}

func TestGenerateCurl_DoesNotDuplicateExplicitContentTypeHeader(t *testing.T) {
	req := collection.Request{
		Method: collection.POST,
		URL:    "https://api.example.com/users",
		Headers: []collection.Header{
			{Key: "Content-Type", Value: "application/xml", Enabled: true},
		},
		Body: collection.Body{Type: collection.BodyRaw, RawContentType: "application/json", RawText: `<a/>`},
	}
	got := GenerateCurl(req)
	if strings.Count(got, "Content-Type") != 1 {
		t.Errorf("expected exactly one Content-Type header, got:\n%s", got)
	}
	if !strings.Contains(got, "-H 'Content-Type: application/xml'") {
		t.Errorf("expected the explicit header to win, got:\n%s", got)
	}
}

func TestGenerateCurl_OmitsDataForBodylessRequest(t *testing.T) {
	req := collection.Request{Method: collection.GET, URL: "https://api.example.com/users"}
	got := GenerateCurl(req)
	if strings.Contains(got, "-d ") {
		t.Errorf("expected no -d flag for a bodyless request, got:\n%s", got)
	}
}

func TestGenerateCurl_EscapesEmbeddedSingleQuotes(t *testing.T) {
	req := collection.Request{
		Method: collection.GET,
		URL:    "https://api.example.com/users",
		Headers: []collection.Header{
			{Key: "X-Note", Value: "it's here", Enabled: true},
		},
	}
	got := GenerateCurl(req)
	if !strings.Contains(got, `it'\''s here`) {
		t.Errorf("expected escaped single quote, got:\n%s", got)
	}
}

func TestGenerateCurl_RoundTripsThroughParseCurl(t *testing.T) {
	original := collection.Request{
		Method: collection.POST,
		URL:    "https://api.example.com/users",
		Params: []collection.QueryParam{{Key: "page", Value: "2", Enabled: true}},
		Headers: []collection.Header{
			{Key: "Authorization", Value: "Bearer xyz", Enabled: true},
		},
		Body: collection.Body{Type: collection.BodyRaw, RawContentType: "application/json", RawText: `{"name":"ada"}`},
	}
	generated := GenerateCurl(original)
	reparsed, err := ParseCurl(generated)
	if err != nil {
		t.Fatalf("generated curl command failed to re-parse: %v\ncommand:\n%s", err, generated)
	}
	if reparsed.Method != original.Method {
		t.Errorf("method = %s, want %s", reparsed.Method, original.Method)
	}
	if reparsed.URL != original.URL {
		t.Errorf("url = %q, want %q", reparsed.URL, original.URL)
	}
	if len(reparsed.Params) != 1 || reparsed.Params[0].Value != "2" {
		t.Errorf("params = %+v", reparsed.Params)
	}
	if reparsed.Body.RawText != original.Body.RawText {
		t.Errorf("body = %q, want %q", reparsed.Body.RawText, original.Body.RawText)
	}
}

func TestGenerateGo_ProducesCompilableLookingSnippet(t *testing.T) {
	req := collection.Request{
		Method:  collection.POST,
		URL:     "https://api.example.com/users",
		Headers: []collection.Header{{Key: "Authorization", Value: "Bearer xyz", Enabled: true}},
		Body:    collection.Body{Type: collection.BodyRaw, RawContentType: "application/json", RawText: `{"name":"ada"}`},
	}
	got := GenerateGo(req)
	for _, want := range []string{
		"package main",
		`http.NewRequest("POST", "https://api.example.com/users"`,
		`req.Header.Set("Authorization", "Bearer xyz")`,
		`strings.NewReader("{\"name\":\"ada\"}")`,
		"http.DefaultClient.Do(req)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected snippet to contain %q, got:\n%s", want, got)
		}
	}
}

func TestGenerateGo_DoesNotDuplicateExplicitContentTypeHeader(t *testing.T) {
	req := collection.Request{
		Method: collection.POST,
		URL:    "https://api.example.com/users",
		Headers: []collection.Header{
			{Key: "Content-Type", Value: "application/xml", Enabled: true},
		},
		Body: collection.Body{Type: collection.BodyRaw, RawContentType: "application/json", RawText: `<a/>`},
	}
	got := GenerateGo(req)
	if strings.Count(got, "Content-Type") != 1 {
		t.Errorf("expected exactly one Content-Type header.Set call, got:\n%s", got)
	}
	if !strings.Contains(got, `req.Header.Set("Content-Type", "application/xml")`) {
		t.Errorf("expected the explicit header to win, got:\n%s", got)
	}
}

func TestGenerateGo_NilBodyForBodylessRequest(t *testing.T) {
	req := collection.Request{Method: collection.GET, URL: "https://api.example.com/users"}
	got := GenerateGo(req)
	if !strings.Contains(got, `http.NewRequest("GET", "https://api.example.com/users", nil)`) {
		t.Errorf("expected nil body for GET, got:\n%s", got)
	}
}
