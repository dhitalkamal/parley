package collection

import (
	"reflect"
	"testing"
)

func TestRequestPathsUnder_FlatFolder(t *testing.T) {
	node := TreeNode{
		Kind: KindFolder,
		Children: []TreeNode{
			{Kind: KindRequest, Name: "a", Path: "010_a.json"},
			{Kind: KindRequest, Name: "b", Path: "020_b.json"},
		},
	}
	got := RequestPathsUnder(node)
	want := []string{"010_a.json", "020_b.json"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRequestPathsUnder_NestedFolders(t *testing.T) {
	node := TreeNode{
		Kind: KindFolder,
		Children: []TreeNode{
			{Kind: KindRequest, Name: "top", Path: "010_top.json"},
			{
				Kind: KindFolder,
				Name: "users",
				Path: "020_users",
				Children: []TreeNode{
					{Kind: KindRequest, Name: "list", Path: "020_users/010_list.json"},
					{Kind: KindRequest, Name: "get", Path: "020_users/020_get.json"},
				},
			},
		},
	}
	got := RequestPathsUnder(node)
	want := []string{"010_top.json", "020_users/010_list.json", "020_users/020_get.json"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRequestPathsUnder_EmptyFolderReturnsNil(t *testing.T) {
	got := RequestPathsUnder(TreeNode{Kind: KindFolder})
	if len(got) != 0 {
		t.Errorf("got %v, want none", got)
	}
}

func TestRequestPathsUnder_SkipsEmptyNestedFolders(t *testing.T) {
	node := TreeNode{
		Kind: KindFolder,
		Children: []TreeNode{
			{Kind: KindFolder, Name: "empty", Path: "010_empty"},
			{Kind: KindRequest, Name: "req", Path: "020_req.json"},
		},
	}
	got := RequestPathsUnder(node)
	want := []string{"020_req.json"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
