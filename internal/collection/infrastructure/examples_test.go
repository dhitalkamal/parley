package collectionstore

import (
	"testing"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
)

func TestStore_SaveThenLoadExampleRoundTrips(t *testing.T) {
	s := New(t.TempDir())
	reqPath, err := s.SaveRequest("", "get-user", collection.Request{Method: collection.GET})
	if err != nil {
		t.Fatalf("save request error: %v", err)
	}

	ex := collection.Example{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    []collection.Header{{Key: "Content-Type", Value: "application/json", Enabled: true}},
		Body:       `{"ok":true}`,
	}
	exPath, err := s.SaveExample(reqPath, "success", ex)
	if err != nil {
		t.Fatalf("save example error: %v", err)
	}

	loaded, err := s.LoadExample(exPath)
	if err != nil {
		t.Fatalf("load example error: %v", err)
	}
	if loaded.Name != "success" || loaded.StatusCode != 200 || loaded.Body != `{"ok":true}` {
		t.Errorf("loaded = %+v", loaded)
	}
	if len(loaded.Headers) != 1 || loaded.Headers[0].Key != "Content-Type" {
		t.Errorf("headers = %+v", loaded.Headers)
	}
}

func TestStore_ListExamplesReturnsSavedNamesSorted(t *testing.T) {
	s := New(t.TempDir())
	reqPath, err := s.SaveRequest("", "get-user", collection.Request{Method: collection.GET})
	if err != nil {
		t.Fatalf("save request error: %v", err)
	}
	for _, name := range []string{"zzz", "aaa"} {
		if _, err := s.SaveExample(reqPath, name, collection.Example{StatusCode: 200}); err != nil {
			t.Fatalf("save example %s error: %v", name, err)
		}
	}
	names, err := s.ListExamples(reqPath)
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	want := []string{"aaa", "zzz"}
	if len(names) != len(want) || names[0] != want[0] || names[1] != want[1] {
		t.Errorf("got %v, want %v", names, want)
	}
}

func TestStore_ListExamplesOnRequestWithNoneReturnsEmpty(t *testing.T) {
	s := New(t.TempDir())
	reqPath, err := s.SaveRequest("", "get-user", collection.Request{Method: collection.GET})
	if err != nil {
		t.Fatalf("save request error: %v", err)
	}
	names, err := s.ListExamples(reqPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("got %v, want none", names)
	}
}

func TestStore_ExamplesDirDoesNotAppearInTree(t *testing.T) {
	s := New(t.TempDir())
	reqPath, err := s.SaveRequest("", "get-user", collection.Request{Method: collection.GET})
	if err != nil {
		t.Fatalf("save request error: %v", err)
	}
	if _, err := s.SaveExample(reqPath, "success", collection.Example{StatusCode: 200}); err != nil {
		t.Fatalf("save example error: %v", err)
	}

	tree, err := s.Tree()
	if err != nil {
		t.Fatalf("tree error: %v", err)
	}
	if len(tree.Children) != 1 {
		t.Fatalf("expected only the request in the tree, got %+v", tree.Children)
	}
	if tree.Children[0].Kind != collection.KindRequest || tree.Children[0].Name != "get-user" {
		t.Errorf("unexpected tree child: %+v", tree.Children[0])
	}
}

func TestStore_SaveExampleRejectsInvalidName(t *testing.T) {
	s := New(t.TempDir())
	reqPath, err := s.SaveRequest("", "get-user", collection.Request{Method: collection.GET})
	if err != nil {
		t.Fatalf("save request error: %v", err)
	}
	if _, err := s.SaveExample(reqPath, "../escape", collection.Example{}); err == nil {
		t.Fatal("expected error for path-escaping name")
	}
}
