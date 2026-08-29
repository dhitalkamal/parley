package collection

// Store persists collections (folders + request files) on disk. Implemented
// by collection/infrastructure. Path values are opaque identifiers returned
// by Tree/CreateFolder/SaveRequest; callers pass them back to address a node.
type Store interface {
	Tree() (TreeNode, error)
	LoadRequest(path string) (Request, error)
	SaveRequest(parentPath, name string, req Request) (path string, err error)
	UpdateRequest(path string, req Request) error
	CreateFolder(parentPath, name string) (path string, err error)
	Rename(path, newName string) (newPath string, err error)
	Delete(path string) error
	MoveUp(path string) error
	MoveDown(path string) error
	SaveExample(requestPath, name string, ex Example) (path string, err error)
	ListExamples(requestPath string) ([]string, error)
	LoadExample(path string) (Example, error)
}
