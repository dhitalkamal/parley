package collectionstore

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	"github.com/dhitalkamal/parley/internal/platform/fsstore"
)

// Store implements collection.Store against a directory tree rooted at Root.
type Store struct {
	Root string
}

var _ collection.Store = (*Store)(nil)

// New returns a Store rooted at the given directory. The directory need not
// exist yet; it's created lazily on first write.
func New(root string) *Store {
	return &Store{Root: root}
}

type siblingEntry struct {
	fsName string
	name   string
	order  int
	isDir  bool
}

func (s *Store) listSiblings(parentAbs string) ([]siblingEntry, error) {
	entries, err := os.ReadDir(parentAbs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []siblingEntry
	for _, e := range entries {
		if e.IsDir() {
			if strings.HasSuffix(e.Name(), examplesSuffix) {
				continue
			}
			name, order := DecodeName(e.Name(), false)
			out = append(out, siblingEntry{fsName: e.Name(), name: name, order: order, isDir: true})
		} else if strings.HasSuffix(e.Name(), ".json") {
			name, order := DecodeName(e.Name(), true)
			out = append(out, siblingEntry{fsName: e.Name(), name: name, order: order, isDir: false})
		}
	}
	// Folders group before requests regardless of manual order or creation
	// time - a user wants a folder created after several requests to still
	// read as a group at the top, not interleaved wherever its order prefix
	// happened to land it. The manual order (then name) still decides
	// standing within each group, so ctrl+up/ctrl+down keep working exactly
	// as before there.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].isDir != out[j].isDir {
			return out[i].isDir
		}
		if out[i].order != out[j].order {
			return out[i].order < out[j].order
		}
		return out[i].name < out[j].name
	})
	return out, nil
}

// dirOf is filepath.Dir but returns "" instead of "." for a top-level path,
// matching the convention that the collections root is the empty path.
func dirOf(p string) string {
	d := filepath.Dir(p)
	if d == "." {
		return ""
	}
	return d
}

func (s *Store) Tree() (collection.TreeNode, error) {
	node, err := s.buildNode(s.Root, "")
	if err != nil {
		return collection.TreeNode{}, err
	}
	node.Name = "collections"
	return node, nil
}

func (s *Store) buildNode(absDir, relPath string) (collection.TreeNode, error) {
	siblings, err := s.listSiblings(absDir)
	if err != nil {
		return collection.TreeNode{}, err
	}
	node := collection.TreeNode{Kind: collection.KindFolder, Path: relPath}
	for _, sib := range siblings {
		childRel := filepath.Join(relPath, sib.fsName)
		if sib.isDir {
			child, err := s.buildNode(filepath.Join(absDir, sib.fsName), childRel)
			if err != nil {
				return collection.TreeNode{}, err
			}
			child.Name = sib.name
			node.Children = append(node.Children, child)
		} else {
			// Reading the file here (rather than just the directory entry)
			// costs one small file read per request, but it's what lets the
			// sidebar show a method badge without loading every request by
			// hand first.
			method := collection.Method("")
			if data, err := os.ReadFile(filepath.Join(absDir, sib.fsName)); err == nil {
				if req, err := decodeRequest(data); err == nil {
					method = req.Method
				}
			}
			node.Children = append(node.Children, collection.TreeNode{
				Kind:   collection.KindRequest,
				Name:   sib.name,
				Path:   childRel,
				Method: method,
			})
		}
	}
	return node, nil
}

func (s *Store) LoadRequest(path string) (collection.Request, error) {
	data, err := os.ReadFile(filepath.Join(s.Root, path))
	if err != nil {
		return collection.Request{}, err
	}
	return decodeRequest(data)
}

func (s *Store) SaveRequest(parentPath, name string, req collection.Request) (string, error) {
	if err := fsstore.ValidateName(name); err != nil {
		return "", err
	}
	parentAbs := filepath.Join(s.Root, parentPath)
	if err := os.MkdirAll(parentAbs, 0o755); err != nil {
		return "", err
	}
	siblings, err := s.listSiblings(parentAbs)
	if err != nil {
		return "", err
	}
	orders := make([]int, len(siblings))
	for i, sib := range siblings {
		orders[i] = sib.order
	}
	fileName := EncodeRequestFileName(NextOrder(orders), name)
	data, err := encodeRequest(req)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(parentAbs, fileName), data, 0o644); err != nil {
		return "", err
	}
	return filepath.Join(parentPath, fileName), nil
}

func (s *Store) UpdateRequest(path string, req collection.Request) error {
	data, err := encodeRequest(req)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.Root, path), data, 0o644)
}

func (s *Store) CreateFolder(parentPath, name string) (string, error) {
	if err := fsstore.ValidateName(name); err != nil {
		return "", err
	}
	parentAbs := filepath.Join(s.Root, parentPath)
	if err := os.MkdirAll(parentAbs, 0o755); err != nil {
		return "", err
	}
	siblings, err := s.listSiblings(parentAbs)
	if err != nil {
		return "", err
	}
	orders := make([]int, len(siblings))
	for i, sib := range siblings {
		orders[i] = sib.order
	}
	dirName := EncodeFolderName(NextOrder(orders), name)
	if err := os.Mkdir(filepath.Join(parentAbs, dirName), 0o755); err != nil {
		return "", err
	}
	return filepath.Join(parentPath, dirName), nil
}

func (s *Store) Rename(path, newName string) (string, error) {
	if err := fsstore.ValidateName(newName); err != nil {
		return "", err
	}
	isRequest := strings.HasSuffix(path, ".json")
	_, order := DecodeName(filepath.Base(path), isRequest)
	var newBase string
	if isRequest {
		newBase = EncodeRequestFileName(order, newName)
	} else {
		newBase = EncodeFolderName(order, newName)
	}
	parentPath := dirOf(path)
	newPath := filepath.Join(parentPath, newBase)
	if err := os.Rename(filepath.Join(s.Root, path), filepath.Join(s.Root, newPath)); err != nil {
		return "", err
	}
	return newPath, nil
}

func (s *Store) Delete(path string) error {
	return os.RemoveAll(filepath.Join(s.Root, path))
}

func (s *Store) MoveUp(path string) error {
	return s.swapWithSibling(path, -1)
}

func (s *Store) MoveDown(path string) error {
	return s.swapWithSibling(path, 1)
}

// swapWithSibling exchanges the manual-sort order between the node at path
// and its adjacent sibling (offset -1 for the previous one, +1 for the
// next), renaming both on disk. A no-op at either end of the sibling list.
func (s *Store) swapWithSibling(path string, offset int) error {
	parentPath := dirOf(path)
	parentAbs := filepath.Join(s.Root, parentPath)
	siblings, err := s.listSiblings(parentAbs)
	if err != nil {
		return err
	}
	base := filepath.Base(path)
	idx := -1
	for i, sib := range siblings {
		if sib.fsName == base {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("store: not found: %s", path)
	}
	j := idx + offset
	if j < 0 || j >= len(siblings) {
		return nil
	}
	a, b := siblings[idx], siblings[j]
	// Folders always sort before requests (see listSiblings) regardless of
	// their manual order number - swapping order across that boundary would
	// change a's order field without ever changing its visible position, a
	// silent no-op that would look like a broken keybinding. Treat crossing
	// into the other kind's group the same as running off the end of the
	// list.
	if a.isDir != b.isDir {
		return nil
	}
	aNew := encodeSibling(a, b.order)
	bNew := encodeSibling(b, a.order)
	if err := os.Rename(filepath.Join(parentAbs, a.fsName), filepath.Join(parentAbs, aNew)); err != nil {
		return err
	}
	return os.Rename(filepath.Join(parentAbs, b.fsName), filepath.Join(parentAbs, bNew))
}

func encodeSibling(e siblingEntry, order int) string {
	if e.isDir {
		return EncodeFolderName(order, e.name)
	}
	return EncodeRequestFileName(order, e.name)
}
