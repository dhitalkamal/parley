package importexport

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	"testing"
)

func TestParseCurl_SimpleGet(t *testing.T) {
	req, err := ParseCurl(`curl https://api.example.com/users`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Method != collection.GET {
		t.Errorf("method = %s, want GET", req.Method)
	}
	if req.URL != "https://api.example.com/users" {
		t.Errorf("url = %q", req.URL)
	}
	if req.Body.Type != collection.BodyNone {
		t.Errorf("body type = %q, want none", req.Body.Type)
	}
}

func TestParseCurl_HeadersParsed(t *testing.T) {
	req, err := ParseCurl(`curl https://api.example.com/users -H 'Authorization: Bearer xyz' -H "Accept: application/json"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(req.Headers) != 2 {
		t.Fatalf("got %d headers, want 2: %+v", len(req.Headers), req.Headers)
	}
	if req.Headers[0].Key != "Authorization" || req.Headers[0].Value != "Bearer xyz" {
		t.Errorf("header[0] = %+v", req.Headers[0])
	}
	if req.Headers[1].Key != "Accept" || req.Headers[1].Value != "application/json" {
		t.Errorf("header[1] = %+v", req.Headers[1])
	}
}

func TestParseCurl_ExplicitMethodAndJSONBody(t *testing.T) {
	req, err := ParseCurl(`curl -X POST https://api.example.com/users -H 'Content-Type: application/json' -d '{"name":"ada"}'`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Method != collection.POST {
		t.Errorf("method = %s, want POST", req.Method)
	}
	if req.Body.Type != collection.BodyRaw || req.Body.RawText != `{"name":"ada"}` {
		t.Errorf("body = %+v", req.Body)
	}
}

func TestParseCurl_DataImpliesPOSTWithoutExplicitX(t *testing.T) {
	req, err := ParseCurl(`curl https://api.example.com/users -d 'name=ada'`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Method != collection.POST {
		t.Errorf("method = %s, want POST (implied by -d)", req.Method)
	}
	if req.Body.RawContentType != "application/x-www-form-urlencoded" {
		t.Errorf("content type = %q, want form-urlencoded default", req.Body.RawContentType)
	}
}

func TestParseCurl_MultipleDataFlagsJoinedWithAmpersand(t *testing.T) {
	req, err := ParseCurl(`curl https://api.example.com/users -d 'a=1' -d 'b=2'`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Body.RawText != "a=1&b=2" {
		t.Errorf("body = %q, want a=1&b=2", req.Body.RawText)
	}
}

func TestParseCurl_BasicAuthFromUserFlag(t *testing.T) {
	req, err := ParseCurl(`curl https://api.example.com/users -u alice:secret`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var auth string
	for _, h := range req.Headers {
		if h.Key == "Authorization" {
			auth = h.Value
		}
	}
	if auth != "Basic YWxpY2U6c2VjcmV0" {
		t.Errorf("Authorization = %q, want Basic YWxpY2U6c2VjcmV0", auth)
	}
}

func TestParseCurl_InsecureFlagSetsSkipVerify(t *testing.T) {
	req, err := ParseCurl(`curl -k https://api.example.com/users`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !req.InsecureSkipVerify {
		t.Error("expected InsecureSkipVerify = true")
	}
}

func TestParseCurl_QueryStringSplitIntoParams(t *testing.T) {
	req, err := ParseCurl(`curl 'https://api.example.com/users?page=2&limit=10'`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.URL != "https://api.example.com/users" {
		t.Errorf("url = %q, want base URL without query", req.URL)
	}
	if len(req.Params) != 2 || req.Params[0].Key != "page" || req.Params[1].Key != "limit" {
		t.Errorf("params = %+v", req.Params)
	}
}

func TestParseCurl_LineContinuationsJoined(t *testing.T) {
	cmd := "curl https://api.example.com/users \\\n  -H 'Accept: application/json'"
	req, err := ParseCurl(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(req.Headers) != 1 || req.Headers[0].Key != "Accept" {
		t.Errorf("headers = %+v", req.Headers)
	}
}

func TestParseCurl_NotACurlCommandReturnsError(t *testing.T) {
	_, err := ParseCurl(`wget https://api.example.com/users`)
	if err == nil {
		t.Fatal("expected an error for a non-curl command")
	}
}

func TestParseCurl_EmptyStringReturnsError(t *testing.T) {
	_, err := ParseCurl("")
	if err == nil {
		t.Fatal("expected an error for empty input")
	}
}

func TestParseCurl_UnterminatedQuoteReturnsErrorNotPanic(t *testing.T) {
	_, err := ParseCurl(`curl https://api.example.com -H 'unterminated`)
	if err == nil {
		t.Fatal("expected an error for an unterminated quote")
	}
}

func TestParseCurl_MissingURLReturnsError(t *testing.T) {
	_, err := ParseCurl(`curl -X GET`)
	if err == nil {
		t.Fatal("expected an error when no URL is present")
	}
}

func TestParseCurl_UrlFlagAccepted(t *testing.T) {
	req, err := ParseCurl(`curl --url https://api.example.com/users -X DELETE`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.URL != "https://api.example.com/users" || req.Method != collection.DELETE {
		t.Errorf("got %+v", req)
	}
}
