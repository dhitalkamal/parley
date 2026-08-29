package importexport

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	"strings"
)

// PersistImportedNode writes an ImportedNode tree into store under
// parentPath, returning how many requests were saved. A node with a
// Request is saved as a single file; a folder node with a non-empty Name
// gets its own wrapping folder (e.g. the collection name on a Postman/
// OpenAPI import); a nameless wrapper (curl import has none) just persists
// its children directly under parentPath with no extra folder.
func PersistImportedNode(s collection.Store, parentPath string, node ImportedNode) (int, error) {
	if node.Request != nil {
		if _, err := s.SaveRequest(parentPath, sanitizeImportName(node.Name), *node.Request); err != nil {
			return 0, err
		}
		return 1, nil
	}

	target := parentPath
	if node.Name != "" {
		folderPath, err := s.CreateFolder(parentPath, sanitizeImportName(node.Name))
		if err != nil {
			return 0, err
		}
		target = folderPath
	}

	count := 0
	for _, child := range node.Children {
		n, err := PersistImportedNode(s, target, child)
		count += n
		if err != nil {
			return count, err
		}
	}
	return count, nil
}

// sanitizeImportName makes an externally-sourced name (Postman item name,
// OpenAPI operation id, curl-derived name) safe for the store's naming
// rules, which reject empty names, ".", "..", and path separators.
func sanitizeImportName(name string) string {
	name = strings.NewReplacer("/", "-", "\\", "-").Replace(strings.TrimSpace(name))
	if name == "" || name == "." || name == ".." {
		return "imported"
	}
	return name
}

// BuildImportedNode loads every request under node (as returned by the
// Store's Tree()) into an ImportedNode tree, ready for an exporter.
func BuildImportedNode(s collection.Store, node collection.TreeNode) (ImportedNode, error) {
	if node.Kind == collection.KindRequest {
		req, err := s.LoadRequest(node.Path)
		if err != nil {
			return ImportedNode{}, err
		}
		return ImportedNode{Name: node.Name, Request: &req}, nil
	}

	result := ImportedNode{Name: node.Name}
	for _, child := range node.Children {
		childNode, err := BuildImportedNode(s, child)
		if err != nil {
			return ImportedNode{}, err
		}
		result.Children = append(result.Children, childNode)
	}
	return result, nil
}
