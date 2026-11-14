// Package cli implements parley's headless collection runner - the same
// send/pre-request/test-script engine the TUI drives, invoked from
// cmd/parley without a terminal UI so a collection can run in CI.
package cli

import (
	"fmt"
	"strings"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
)

// ResolveRequestPaths finds the folder or request named by namePath inside
// tree and returns the store paths of every request that should run for it,
// in the same depth-first order collection.RequestPathsUnder defines. An
// empty namePath means "the whole collection". namePath segments are the
// human-readable Name at each level (e.g. "users/get"), not the on-disk
// Path - callers of parley run shouldn't need to know the order-prefixed
// file naming scheme.
func ResolveRequestPaths(tree collection.TreeNode, namePath string) ([]string, error) {
	node, err := resolveNode(tree, namePath)
	if err != nil {
		return nil, err
	}
	if node.Kind == collection.KindRequest {
		return []string{node.Path}, nil
	}
	paths := collection.RequestPathsUnder(node)
	if len(paths) == 0 {
		if namePath == "" {
			return nil, fmt.Errorf("cli: the collection has no requests to run (add some in the parley TUI first)")
		}
		return nil, fmt.Errorf("cli: folder %q has no requests to run", namePath)
	}
	return paths, nil
}

func resolveNode(tree collection.TreeNode, namePath string) (collection.TreeNode, error) {
	if namePath == "" {
		return tree, nil
	}
	current := tree
	for _, seg := range strings.Split(namePath, "/") {
		child, ok := findChildByName(current, seg)
		if !ok {
			return collection.TreeNode{}, fmt.Errorf("cli: no folder or request named %q in path %q (paths match folder and request names, like users/get;%s omit the path to run everything)",
				seg, namePath, availableNames(current))
		}
		current = child
	}
	return current, nil
}

func findChildByName(node collection.TreeNode, name string) (collection.TreeNode, bool) {
	for _, child := range node.Children {
		if child.Name == name {
			return child, true
		}
	}
	return collection.TreeNode{}, false
}

// availableNames renders " available here: a, b;" for the children of node,
// or "" when it has none, so the unknown-path message can point at real choices.
func availableNames(node collection.TreeNode) string {
	if len(node.Children) == 0 {
		return ""
	}
	names := make([]string, len(node.Children))
	for i, c := range node.Children {
		names[i] = c.Name
	}
	return " available here: " + strings.Join(names, ", ") + ";"
}
