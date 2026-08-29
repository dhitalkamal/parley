package collectionstore

import (
	"path/filepath"
	"testing"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
)

func TestStore_TreeOnEmptyRootReturnsEmptyFolder(t *testing.T) {
	s := New(t.TempDir())
	tree, err := s.Tree()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tree.Kind != collection.KindFolder {
		t.Errorf("root kind = %v, want KindFolder", tree.Kind)
	}
	if len(tree.Children) != 0 {
		t.Errorf("expected no children, got %+v", tree.Children)
	}
}

func TestStore_SaveRequestThenLoadRoundTrips(t *testing.T) {
	s := New(t.TempDir())
	req := collection.Request{Method: collection.GET, URL: "https://example.com/a"}

	path, err := s.SaveRequest("", "get-a", req)
	if err != nil {
		t.Fatalf("save error: %v", err)
	}

	loaded, err := s.LoadRequest(path)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if loaded.URL != req.URL || loaded.Method != req.Method {
		t.Errorf("loaded = %+v, want %+v", loaded, req)
	}

	tree, err := s.Tree()
	if err != nil {
		t.Fatalf("tree error: %v", err)
	}
	if len(tree.Children) != 1 || tree.Children[0].Name != "get-a" || tree.Children[0].Kind != collection.KindRequest {
		t.Errorf("tree children = %+v", tree.Children)
	}
}

func TestStore_SaveRequestAssignsIncrementingOrder(t *testing.T) {
	s := New(t.TempDir())
	req := collection.Request{Method: collection.GET, URL: "https://example.com"}

	if _, err := s.SaveRequest("", "first", req); err != nil {
		t.Fatalf("save error: %v", err)
	}
	if _, err := s.SaveRequest("", "second", req); err != nil {
		t.Fatalf("save error: %v", err)
	}

	tree, err := s.Tree()
	if err != nil {
		t.Fatalf("tree error: %v", err)
	}
	if len(tree.Children) != 2 {
		t.Fatalf("expected 2 children, got %+v", tree.Children)
	}
	if tree.Children[0].Name != "first" || tree.Children[1].Name != "second" {
		t.Errorf("order = [%s, %s], want [first, second]", tree.Children[0].Name, tree.Children[1].Name)
	}
}

func TestStore_CreateFolderThenSaveRequestInsideIt(t *testing.T) {
	s := New(t.TempDir())
	folderPath, err := s.CreateFolder("", "users")
	if err != nil {
		t.Fatalf("create folder error: %v", err)
	}

	req := collection.Request{Method: collection.GET, URL: "https://example.com/users"}
	reqPath, err := s.SaveRequest(folderPath, "list-users", req)
	if err != nil {
		t.Fatalf("save error: %v", err)
	}

	tree, err := s.Tree()
	if err != nil {
		t.Fatalf("tree error: %v", err)
	}
	if len(tree.Children) != 1 || tree.Children[0].Kind != collection.KindFolder || tree.Children[0].Name != "users" {
		t.Fatalf("expected one users folder, got %+v", tree.Children)
	}
	folder := tree.Children[0]
	if len(folder.Children) != 1 || folder.Children[0].Kind != collection.KindRequest || folder.Children[0].Name != "list-users" {
		t.Fatalf("expected list-users request inside users folder, got %+v", folder.Children)
	}
	if folder.Children[0].Path != reqPath {
		t.Errorf("path mismatch: tree has %q, save returned %q", folder.Children[0].Path, reqPath)
	}
}

func TestStore_UpdateRequestOverwritesInPlace(t *testing.T) {
	s := New(t.TempDir())
	req := collection.Request{Method: collection.GET, URL: "https://example.com/old"}
	path, err := s.SaveRequest("", "req", req)
	if err != nil {
		t.Fatalf("save error: %v", err)
	}

	updated := req
	updated.URL = "https://example.com/new"
	if err := s.UpdateRequest(path, updated); err != nil {
		t.Fatalf("update error: %v", err)
	}

	loaded, err := s.LoadRequest(path)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if loaded.URL != "https://example.com/new" {
		t.Errorf("URL = %q, want updated URL", loaded.URL)
	}

	tree, err := s.Tree()
	if err != nil {
		t.Fatalf("tree error: %v", err)
	}
	if len(tree.Children) != 1 {
		t.Errorf("expected update to not create a new file, got %+v", tree.Children)
	}
}

func TestStore_RenameChangesDisplayNameKeepsOrder(t *testing.T) {
	s := New(t.TempDir())
	req := collection.Request{Method: collection.GET, URL: "https://example.com"}
	if _, err := s.SaveRequest("", "first", req); err != nil {
		t.Fatalf("save error: %v", err)
	}
	path, err := s.SaveRequest("", "second", req)
	if err != nil {
		t.Fatalf("save error: %v", err)
	}

	newPath, err := s.Rename(path, "renamed")
	if err != nil {
		t.Fatalf("rename error: %v", err)
	}

	tree, err := s.Tree()
	if err != nil {
		t.Fatalf("tree error: %v", err)
	}
	if len(tree.Children) != 2 || tree.Children[0].Name != "first" || tree.Children[1].Name != "renamed" {
		t.Errorf("expected [first, renamed] preserving order, got %+v", tree.Children)
	}
	if tree.Children[1].Path != newPath {
		t.Errorf("path mismatch: tree has %q, rename returned %q", tree.Children[1].Path, newPath)
	}
}

func TestStore_DeleteRemovesFileAndFolder(t *testing.T) {
	s := New(t.TempDir())
	req := collection.Request{Method: collection.GET, URL: "https://example.com"}
	reqPath, err := s.SaveRequest("", "req", req)
	if err != nil {
		t.Fatalf("save error: %v", err)
	}
	folderPath, err := s.CreateFolder("", "folder")
	if err != nil {
		t.Fatalf("create folder error: %v", err)
	}
	if _, err := s.SaveRequest(folderPath, "nested", req); err != nil {
		t.Fatalf("save error: %v", err)
	}

	if err := s.Delete(reqPath); err != nil {
		t.Fatalf("delete request error: %v", err)
	}
	if err := s.Delete(folderPath); err != nil {
		t.Fatalf("delete folder error: %v", err)
	}

	tree, err := s.Tree()
	if err != nil {
		t.Fatalf("tree error: %v", err)
	}
	if len(tree.Children) != 0 {
		t.Errorf("expected empty tree after deleting both, got %+v", tree.Children)
	}
}

func TestStore_MoveUpSwapsOrderWithPreviousSibling(t *testing.T) {
	s := New(t.TempDir())
	req := collection.Request{Method: collection.GET, URL: "https://example.com"}
	if _, err := s.SaveRequest("", "first", req); err != nil {
		t.Fatalf("save error: %v", err)
	}
	secondPath, err := s.SaveRequest("", "second", req)
	if err != nil {
		t.Fatalf("save error: %v", err)
	}

	if err := s.MoveUp(secondPath); err != nil {
		t.Fatalf("move up error: %v", err)
	}

	tree, err := s.Tree()
	if err != nil {
		t.Fatalf("tree error: %v", err)
	}
	if len(tree.Children) != 2 || tree.Children[0].Name != "second" || tree.Children[1].Name != "first" {
		t.Errorf("expected [second, first] after move up, got %+v", tree.Children)
	}
}

func TestStore_MoveDownSwapsOrderWithNextSibling(t *testing.T) {
	s := New(t.TempDir())
	req := collection.Request{Method: collection.GET, URL: "https://example.com"}
	firstPath, err := s.SaveRequest("", "first", req)
	if err != nil {
		t.Fatalf("save error: %v", err)
	}
	if _, err := s.SaveRequest("", "second", req); err != nil {
		t.Fatalf("save error: %v", err)
	}

	if err := s.MoveDown(firstPath); err != nil {
		t.Fatalf("move down error: %v", err)
	}

	tree, err := s.Tree()
	if err != nil {
		t.Fatalf("tree error: %v", err)
	}
	if len(tree.Children) != 2 || tree.Children[0].Name != "second" || tree.Children[1].Name != "first" {
		t.Errorf("expected [second, first] after move down, got %+v", tree.Children)
	}
}

func TestStore_MoveUpAtTopIsNoOp(t *testing.T) {
	s := New(t.TempDir())
	req := collection.Request{Method: collection.GET, URL: "https://example.com"}
	firstPath, err := s.SaveRequest("", "first", req)
	if err != nil {
		t.Fatalf("save error: %v", err)
	}

	if err := s.MoveUp(firstPath); err != nil {
		t.Fatalf("expected no-op, got error: %v", err)
	}

	tree, err := s.Tree()
	if err != nil {
		t.Fatalf("tree error: %v", err)
	}
	if len(tree.Children) != 1 || tree.Children[0].Name != "first" {
		t.Errorf("expected unchanged single child, got %+v", tree.Children)
	}
}

func TestStore_SaveRequestRejectsInvalidName(t *testing.T) {
	s := New(t.TempDir())
	req := collection.Request{Method: collection.GET, URL: "https://example.com"}
	if _, err := s.SaveRequest("", "../escape", req); err == nil {
		t.Fatal("expected error for path-escaping name")
	}
}

func TestStore_SaveRequestCreatesRootDirIfMissing(t *testing.T) {
	root := filepath.Join(t.TempDir(), "does-not-exist-yet")
	s := New(root)
	req := collection.Request{Method: collection.GET, URL: "https://example.com"}
	if _, err := s.SaveRequest("", "req", req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestStore_TreePopulatesRequestMethod guards a sidebar feature: showing a
// method badge next to each request without opening it needs the tree
// itself to carry the method, not just the name/path DecodeName already
// gives it for free.
func TestStore_TreePopulatesRequestMethod(t *testing.T) {
	s := New(t.TempDir())
	if _, err := s.SaveRequest("", "delete-user", collection.Request{Method: collection.DELETE, URL: "https://example.com"}); err != nil {
		t.Fatalf("save error: %v", err)
	}

	tree, err := s.Tree()
	if err != nil {
		t.Fatalf("tree error: %v", err)
	}
	if len(tree.Children) != 1 {
		t.Fatalf("got %d children, want 1", len(tree.Children))
	}
	if got := tree.Children[0].Method; got != collection.DELETE {
		t.Errorf("Children[0].Method = %q, want %q", got, collection.DELETE)
	}
}

// TestStore_TreeLeavesFolderMethodEmpty documents that Method is only
// meaningful for request nodes - a folder has no HTTP method of its own.
func TestStore_TreeLeavesFolderMethodEmpty(t *testing.T) {
	s := New(t.TempDir())
	if _, err := s.CreateFolder("", "users"); err != nil {
		t.Fatalf("create folder error: %v", err)
	}

	tree, err := s.Tree()
	if err != nil {
		t.Fatalf("tree error: %v", err)
	}
	if got := tree.Children[0].Method; got != "" {
		t.Errorf("folder Method = %q, want empty", got)
	}
}

// TestStore_TreePopulatesRequestURL guards the sidebar's same-endpoint
// discoverability: two requests can share a URL with different methods
// (e.g. GET/POST /api/v1/organization), and the only way to tell without
// opening each one is if the tree node carries the URL alongside Method -
// mirrors TestStore_TreePopulatesRequestMethod above.
// TestStore_TreeGroupsFoldersBeforeRequestsRegardlessOfCreationOrder guards
// a real usability complaint: a folder created after several requests used
// to sort wherever its manual order prefix landed it, interleaved among
// the requests - a user wants folders to always read as a group at the
// top, no matter which order things were added in.
func TestStore_TreeGroupsFoldersBeforeRequestsRegardlessOfCreationOrder(t *testing.T) {
	s := New(t.TempDir())
	req := collection.Request{Method: collection.GET, URL: "https://example.com"}

	if _, err := s.SaveRequest("", "a-request", req); err != nil {
		t.Fatalf("save error: %v", err)
	}
	if _, err := s.SaveRequest("", "b-request", req); err != nil {
		t.Fatalf("save error: %v", err)
	}
	// Created last, with the highest manual order prefix - the old sort
	// would have placed this after both requests.
	if _, err := s.CreateFolder("", "z-folder"); err != nil {
		t.Fatalf("create folder error: %v", err)
	}

	tree, err := s.Tree()
	if err != nil {
		t.Fatalf("tree error: %v", err)
	}
	if len(tree.Children) != 3 {
		t.Fatalf("got %d children, want 3", len(tree.Children))
	}
	if tree.Children[0].Kind != collection.KindFolder || tree.Children[0].Name != "z-folder" {
		t.Errorf("Children[0] = %+v, want the folder first despite being created last", tree.Children[0])
	}
	if tree.Children[1].Kind != collection.KindRequest || tree.Children[1].Name != "a-request" {
		t.Errorf("Children[1] = %+v, want a-request", tree.Children[1])
	}
	if tree.Children[2].Kind != collection.KindRequest || tree.Children[2].Name != "b-request" {
		t.Errorf("Children[2] = %+v, want b-request", tree.Children[2])
	}
}

// TestStore_MoveUpAtTopOfRequestGroupIsNoOp guards the boundary between the
// folders group and the requests group: since folders always sort before
// requests now, the first request's "previous sibling" on disk is a
// folder. Without a guard, swapWithSibling would still swap their order
// numbers and rename both files on disk - the folders-first grouping masks
// this from the rendered tree order (a request that got a lower order
// number than a folder still renders after it), so checking Name/Kind
// alone isn't enough to catch a regression here; Path encodes the order
// prefix (see naming.go's EncodeFolderName/EncodeRequestFileName), so it's
// what actually changes if the swap wrongly goes through - a real cost
// (needless renames/order churn on every keypress at this boundary) even
// though nothing looks different on screen.
func TestStore_MoveUpAtTopOfRequestGroupIsNoOp(t *testing.T) {
	s := New(t.TempDir())
	req := collection.Request{Method: collection.GET, URL: "https://example.com"}
	folderPath, err := s.CreateFolder("", "a-folder")
	if err != nil {
		t.Fatalf("create folder error: %v", err)
	}
	firstRequestPath, err := s.SaveRequest("", "b-request", req)
	if err != nil {
		t.Fatalf("save error: %v", err)
	}
	if _, err := s.SaveRequest("", "c-request", req); err != nil {
		t.Fatalf("save error: %v", err)
	}

	if err := s.MoveUp(firstRequestPath); err != nil {
		t.Fatalf("expected no-op, got error: %v", err)
	}

	tree, err := s.Tree()
	if err != nil {
		t.Fatalf("tree error: %v", err)
	}
	if len(tree.Children) != 3 || tree.Children[0].Name != "a-folder" ||
		tree.Children[1].Name != "b-request" || tree.Children[2].Name != "c-request" {
		t.Errorf("expected unchanged [a-folder, b-request, c-request], got %+v", tree.Children)
	}
	if tree.Children[0].Path != folderPath || tree.Children[1].Path != firstRequestPath {
		t.Errorf("Path changed despite being a no-op: folder %q (want %q), request %q (want %q) - order numbers got swapped across the folder/request boundary", tree.Children[0].Path, folderPath, tree.Children[1].Path, firstRequestPath)
	}
}
