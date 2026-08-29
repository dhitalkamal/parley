package collection

// NodeKind distinguishes a folder from a request in a collection tree.
type NodeKind string

const (
	KindFolder  NodeKind = "folder"
	KindRequest NodeKind = "request"
)

// TreeNode is one entry in a collection's folder/request tree, as read from
// disk. Path is the identity used for load/save/rename/delete/reorder calls
// against the Store port; it is opaque to callers.
type TreeNode struct {
	Kind NodeKind
	Name string
	Path string
	// Method is only set for a KindRequest node - a quick-glance badge in
	// the sidebar without loading the full request. Empty for folders.
	Method   Method
	Children []TreeNode
}

// RequestPathsUnder returns the paths of every request under node, depth
// first - folders and their descendants included - the order the
// collection runner executes them in.
func RequestPathsUnder(node TreeNode) []string {
	var paths []string
	for _, child := range node.Children {
		if child.Kind == KindRequest {
			paths = append(paths, child.Path)
		} else {
			paths = append(paths, RequestPathsUnder(child)...)
		}
	}
	return paths
}
