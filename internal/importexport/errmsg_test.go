package importexport

import (
	"strings"
	"testing"
)

func TestParseAnyMessage_UnrecognizedFormatListsWhatWasTriedAndOffersFilePath(t *testing.T) {
	_, err := ParseAny("hello world")
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"unrecognized import format", "curl", "Postman", `"item"`, "OpenAPI", `"openapi"`, "file path"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("message %q is missing %q", err, want)
		}
	}
}

func TestParseAnyMessage_BadCurlNamesTheFormat(t *testing.T) {
	_, err := ParseAny(`curl https://api.example.com -H 'unterminated`)
	if err == nil || !strings.HasPrefix(err.Error(), "invalid curl command: ") {
		t.Errorf("want invalid curl command prefix, got: %v", err)
	}
}

func TestParseAnyMessage_CurlWithoutURLSaysWhatToAdd(t *testing.T) {
	_, err := ParseAny(`curl -X POST`)
	if err == nil || !strings.Contains(err.Error(), "no URL found") {
		t.Errorf("want a no URL found message, got: %v", err)
	}
}

func TestParseAnyMessage_BrokenPostmanNamesTheFormat(t *testing.T) {
	_, err := ParseAny(`{"info": {"name": 1}, "item": "x"}`)
	if err == nil || !strings.HasPrefix(err.Error(), "invalid Postman collection: ") {
		t.Errorf("want invalid Postman collection prefix, got: %v", err)
	}
}

func TestParseAnyMessage_BrokenOpenAPINamesTheFormat(t *testing.T) {
	_, err := ParseAny(`{"openapi": "3.0.0", "paths": "x"}`)
	if err == nil || !strings.HasPrefix(err.Error(), "invalid OpenAPI spec: ") {
		t.Errorf("want invalid OpenAPI spec prefix, got: %v", err)
	}
}
