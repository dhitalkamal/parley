package importexport

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	"testing"
)

func TestParseAny_DetectsCurl(t *testing.T) {
	node, err := ParseAny(`curl https://api.example.com/users`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node.Request == nil || node.Request.Method != collection.GET {
		t.Errorf("got %+v", node)
	}
}

func TestParseAny_DetectsPostman(t *testing.T) {
	node, err := ParseAny(samplePostmanCollection)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(node.Children) != 2 {
		t.Errorf("got %+v, want the sample's 2 top-level items", node.Children)
	}
}

func TestParseAny_DetectsOpenAPIJSON(t *testing.T) {
	node, err := ParseAny(sampleOpenAPIJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(node.Children) == 0 {
		t.Errorf("got %+v, want parsed OpenAPI operations", node.Children)
	}
}

func TestParseAny_DetectsOpenAPIYAML(t *testing.T) {
	node, err := ParseAny(sampleOpenAPIYAML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(node.Children) == 0 {
		t.Errorf("got %+v, want parsed OpenAPI operations", node.Children)
	}
}

func TestParseAny_UnrecognizedFormatReturnsError(t *testing.T) {
	_, err := ParseAny("this is neither curl nor json nor yaml: [[[")
	if err == nil {
		t.Fatal("expected an error for unrecognized input")
	}
}

func TestParseAny_PlainTextReturnsErrorNotPanic(t *testing.T) {
	_, err := ParseAny("hello world")
	if err == nil {
		t.Fatal("expected an error for plain text")
	}
}
