package importexport

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	collectionstore "github.com/dhitalkamal/parley/internal/collection/infrastructure"
	"testing"
)

func TestPersistImportedNode_SingleRequestNoWrappingFolder(t *testing.T) {
	s := collectionstore.New(t.TempDir())
	req := collection.Request{Method: collection.GET, URL: "https://example.com"}
	node := ImportedNode{Name: "my-request", Request: &req}

	count, err := PersistImportedNode(s, "", node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	tree, err := s.Tree()
	if err != nil {
		t.Fatalf("tree error: %v", err)
	}
	if len(tree.Children) != 1 || tree.Children[0].Kind != collection.KindRequest || tree.Children[0].Name != "my-request" {
		t.Errorf("tree = %+v, want a single request named my-request at root", tree.Children)
	}
}

func TestPersistImportedNode_CreatesWrappingFolderForNamedCollection(t *testing.T) {
	s := collectionstore.New(t.TempDir())
	req := collection.Request{Method: collection.GET, URL: "https://example.com"}
	node := ImportedNode{
		Name: "My Collection",
		Children: []ImportedNode{
			{Name: "leaf", Request: &req},
		},
	}

	count, err := PersistImportedNode(s, "", node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}

	tree, err := s.Tree()
	if err != nil {
		t.Fatalf("tree error: %v", err)
	}
	if len(tree.Children) != 1 || tree.Children[0].Kind != collection.KindFolder || tree.Children[0].Name != "My Collection" {
		t.Fatalf("tree = %+v, want a My Collection folder", tree.Children)
	}
	folder := tree.Children[0]
	if len(folder.Children) != 1 || folder.Children[0].Name != "leaf" {
		t.Errorf("folder children = %+v", folder.Children)
	}
}

func TestPersistImportedNode_SanitizesNamesWithPathSeparators(t *testing.T) {
	s := collectionstore.New(t.TempDir())
	req := collection.Request{Method: collection.GET, URL: "https://example.com"}
	node := ImportedNode{Name: "GET /users/{id}", Request: &req}

	_, err := PersistImportedNode(s, "", node)
	if err != nil {
		t.Fatalf("unexpected error for a name containing '/': %v", err)
	}
}

func TestPersistImportedNode_CountsAllNestedRequests(t *testing.T) {
	s := collectionstore.New(t.TempDir())
	req := collection.Request{Method: collection.GET, URL: "https://example.com"}
	node := ImportedNode{
		Name: "root",
		Children: []ImportedNode{
			{Name: "a", Request: &req},
			{
				Name: "folder",
				Children: []ImportedNode{
					{Name: "b", Request: &req},
					{Name: "c", Request: &req},
				},
			},
		},
	}

	count, err := PersistImportedNode(s, "", node)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
}

func TestBuildImportedNode_RoundTripsFromRealStore(t *testing.T) {
	s := collectionstore.New(t.TempDir())
	req := collection.Request{Method: collection.POST, URL: "https://example.com/users", TestScript: `pm.test("ok", function(){});`}
	folderPath, err := s.CreateFolder("", "users")
	if err != nil {
		t.Fatalf("create folder error: %v", err)
	}
	if _, err := s.SaveRequest(folderPath, "create-user", req); err != nil {
		t.Fatalf("save error: %v", err)
	}

	tree, err := s.Tree()
	if err != nil {
		t.Fatalf("tree error: %v", err)
	}
	node, err := BuildImportedNode(s, tree)
	if err != nil {
		t.Fatalf("build error: %v", err)
	}

	if len(node.Children) != 1 || node.Children[0].Name != "users" {
		t.Fatalf("children = %+v", node.Children)
	}
	folder := node.Children[0]
	if len(folder.Children) != 1 || folder.Children[0].Name != "create-user" {
		t.Fatalf("folder children = %+v", folder.Children)
	}
	got := folder.Children[0].Request
	if got == nil || got.URL != req.URL || got.TestScript != req.TestScript {
		t.Errorf("got %+v, want matching %+v", got, req)
	}
}

func TestBuildImportedNode_PropagatesLoadError(t *testing.T) {
	s := collectionstore.New(t.TempDir())
	bogus := collection.TreeNode{
		Kind: collection.KindFolder,
		Children: []collection.TreeNode{
			{Kind: collection.KindRequest, Name: "missing", Path: "does-not-exist.json"},
		},
	}
	_, err := BuildImportedNode(s, bogus)
	if err == nil {
		t.Fatal("expected an error for a request path that doesn't exist on disk")
	}
}
