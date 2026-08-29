package httpclient

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	"io"
	"os"
	"strings"
	"testing"
)

func TestBuildRequestBody_URLEncodedEncodesEnabledFieldsOnly(t *testing.T) {
	body := collection.Body{
		Type: collection.BodyURLEncoded,
		FormFields: []collection.BodyFormField{
			{Key: "user", Value: "ada", Enabled: true},
			{Key: "skip", Value: "nope", Enabled: false},
		},
	}

	r, contentType, err := buildRequestBody(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if contentType != "application/x-www-form-urlencoded" {
		t.Errorf("contentType = %q", contentType)
	}
	got, _ := io.ReadAll(r)
	if string(got) != "user=ada" {
		t.Errorf("body = %q, want %q", got, "user=ada")
	}
}

func TestBuildRequestBody_GraphQLWrapsQueryAndVariables(t *testing.T) {
	body := collection.Body{
		Type:             collection.BodyGraphQL,
		GraphQLQuery:     "query { me { id } }",
		GraphQLVariables: `{"id": 1}`,
	}

	r, contentType, err := buildRequestBody(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if contentType != "application/json" {
		t.Errorf("contentType = %q", contentType)
	}
	got, _ := io.ReadAll(r)
	want := `{"query":"query { me { id } }","variables":{"id":1}}`
	if string(got) != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

func TestBuildRequestBody_GraphQLWithoutVariablesOmitsTheKey(t *testing.T) {
	body := collection.Body{Type: collection.BodyGraphQL, GraphQLQuery: "query { me { id } }"}

	r, _, err := buildRequestBody(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, _ := io.ReadAll(r)
	want := `{"query":"query { me { id } }"}`
	if string(got) != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

func TestBuildRequestBody_BinaryReadsFileContents(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "upload-*.bin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := f.Write([]byte("raw bytes")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f.Close()

	body := collection.Body{Type: collection.BodyBinary, BinaryFilePath: f.Name()}
	r, contentType, err := buildRequestBody(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if contentType != "application/octet-stream" {
		t.Errorf("contentType = %q", contentType)
	}
	got, _ := io.ReadAll(r)
	if string(got) != "raw bytes" {
		t.Errorf("body = %q", got)
	}
}

func TestBuildRequestBody_BinaryMissingFileReturnsError(t *testing.T) {
	body := collection.Body{Type: collection.BodyBinary, BinaryFilePath: "/no/such/file"}
	if _, _, err := buildRequestBody(body); err == nil {
		t.Fatal("expected an error for a missing file")
	}
}

func TestBuildRequestBody_MultipartWritesTextAndFileFields(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "photo-*.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := f.Write([]byte("PNGDATA")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f.Close()

	body := collection.Body{
		Type: collection.BodyMultipart,
		FormFields: []collection.BodyFormField{
			{Key: "title", Value: "photo", Enabled: true},
			{Key: "file", FilePath: f.Name(), IsFile: true, Enabled: true},
			{Key: "skip", Value: "nope", Enabled: false},
		},
	}

	r, contentType, err := buildRequestBody(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(contentType, "multipart/form-data; boundary=") {
		t.Errorf("contentType = %q, want a multipart/form-data boundary", contentType)
	}
	got, _ := io.ReadAll(r)
	if !strings.Contains(string(got), `name="title"`) || !strings.Contains(string(got), "photo") {
		t.Errorf("body missing title field: %q", got)
	}
	if !strings.Contains(string(got), "PNGDATA") {
		t.Errorf("body missing file contents: %q", got)
	}
	if strings.Contains(string(got), "nope") {
		t.Errorf("body should not contain the disabled field's value: %q", got)
	}
}
