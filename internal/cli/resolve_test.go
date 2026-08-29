package cli

import (
	"reflect"
	"testing"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
)

func sampleTree() collection.TreeNode {
	return collection.TreeNode{
		Kind: collection.KindFolder,
		Children: []collection.TreeNode{
			{Kind: collection.KindRequest, Name: "top", Path: "010_top.json"},
			{
				Kind: collection.KindFolder,
				Name: "users",
				Path: "020_users",
				Children: []collection.TreeNode{
					{Kind: collection.KindRequest, Name: "list", Path: "020_users/010_list.json"},
					{Kind: collection.KindRequest, Name: "get", Path: "020_users/020_get.json"},
				},
			},
		},
	}
}

func TestResolveRequestPaths_EmptyPathRunsWholeCollection(t *testing.T) {
	got, err := ResolveRequestPaths(sampleTree(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"010_top.json", "020_users/010_list.json", "020_users/020_get.json"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestResolveRequestPaths_FolderNameRunsOnlyThatFolder(t *testing.T) {
	got, err := ResolveRequestPaths(sampleTree(), "users")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"020_users/010_list.json", "020_users/020_get.json"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestResolveRequestPaths_SingleRequestNameRunsJustThatRequest(t *testing.T) {
	got, err := ResolveRequestPaths(sampleTree(), "users/get")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"020_users/020_get.json"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestResolveRequestPaths_TopLevelRequestNameByItself(t *testing.T) {
	got, err := ResolveRequestPaths(sampleTree(), "top")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"010_top.json"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestResolveRequestPaths_UnknownNameReturnsError(t *testing.T) {
	_, err := ResolveRequestPaths(sampleTree(), "nope")
	if err == nil {
		t.Fatal("expected an error for an unknown name, got nil")
	}
}

func TestResolveRequestPaths_UnknownNestedNameReturnsError(t *testing.T) {
	_, err := ResolveRequestPaths(sampleTree(), "users/nope")
	if err == nil {
		t.Fatal("expected an error for an unknown nested name, got nil")
	}
}

func TestResolveRequestPaths_EmptyFolderReturnsError(t *testing.T) {
	tree := collection.TreeNode{
		Kind: collection.KindFolder,
		Children: []collection.TreeNode{
			{Kind: collection.KindFolder, Name: "empty", Path: "010_empty"},
		},
	}
	_, err := ResolveRequestPaths(tree, "empty")
	if err == nil {
		t.Fatal("expected an error for a folder with no requests, got nil")
	}
}
