package collectionstore

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	"github.com/dhitalkamal/parley/internal/platform/fsstore"
)

// Store implements collection.Store against a directory tree rooted at Root.
type Store struct {
	Root string

	// methodCache remembers the decoded method per request file so that
	// Tree() - called on every autosave and structural change via
	// refreshTree - does not re-read and re-parse unchanged files on disk.
	// Keyed by absolute file path; an entry is only trusted when the file's
	// current modtime and size still match the ones recorded when it was read.
	mu          sync.Mutex
	methodCache map[string]methodCacheEntry
}

// methodCacheEntry is a cached request method plus the file identity it was
// read from. A mismatch on modTime or size means the file changed and the
// method must be re-read.
type methodCacheEntry struct {
	method  collection.Method
	modTime time.Time
	size    int64
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
	// modTime and size identify a request file for the method cache; zero
	// for folders and for files whose Info() could not be stat'd (which
	// simply forces a cache miss and a fresh read).
	modTime time.Time
	size    int64
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
			var modTime time.Time
			var size int64
			if info, err := e.Info(); err == nil {
				modTime = info.ModTime()
				size = info.Size()
			}
			out = append(out, siblingEntry{fsName: e.Name(), name: name, order: order, isDir: false, modTime: modTime, size: size})
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

// resolve joins the caller-supplied relative path onto Root and confirms the
// cleaned result stays inside Root. filepath.Join cleans ".." segments but
// does not contain them, so a path like "../../etc/x.json" would otherwise
// read or write outside the collections root. Some paths originate from
// imported or shared request files (e.g. refresh.requestPath deserialized
// verbatim), so this guard is what stops a crafted collection from reaching
// arbitrary files on disk.
func (s *Store) resolve(path string) (string, error) {
	abs := filepath.Join(s.Root, path)
	rel, err := filepath.Rel(s.Root, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("store: path escapes collections root: %s", path)
	}
	return abs, nil
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
			// The sidebar shows a method badge per request without opening it,
			// which needs the method from inside each file. Reading and parsing
			// every file on every Tree() call is costly because refreshTree
			// calls Tree() on each autosave and structural change; so cache the
			// method keyed by modtime+size and only re-read files that changed.
			abs := filepath.Join(absDir, sib.fsName)
			method := s.cachedMethod(abs, sib.modTime, sib.size)
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

// cachedMethod returns the method for the request file at abs, reading and
// decoding it only when there is no cache entry matching the given modtime and
// size. A zero modTime (Info() failed at listing time) always misses the
// cache, falling back to a fresh read. A decode failure yields an empty method
// and is not cached, so a transiently-bad file is retried next time.
func (s *Store) cachedMethod(abs string, modTime time.Time, size int64) collection.Method {
	s.mu.Lock()
	if entry, ok := s.methodCache[abs]; ok && !modTime.IsZero() && entry.modTime.Equal(modTime) && entry.size == size {
		s.mu.Unlock()
		return entry.method
	}
	s.mu.Unlock()

	method := collection.Method("")
	data, err := os.ReadFile(abs)
	if err != nil {
		return method
	}
	req, err := decodeRequest(data)
	if err != nil {
		return method
	}
	method = req.Method

	// Only cache when we have a valid file identity to validate against later.
	if !modTime.IsZero() {
		s.mu.Lock()
		if s.methodCache == nil {
			s.methodCache = make(map[string]methodCacheEntry)
		}
		s.methodCache[abs] = methodCacheEntry{method: method, modTime: modTime, size: size}
		s.mu.Unlock()
	}
	return method
}

func (s *Store) LoadRequest(path string) (collection.Request, error) {
	abs, err := s.resolve(path)
	if err != nil {
		return collection.Request{}, err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return collection.Request{}, err
	}
	return decodeRequest(data)
}

func (s *Store) SaveRequest(parentPath, name string, req collection.Request) (string, error) {
	if err := fsstore.ValidateName(name); err != nil {
		return "", err
	}
	parentAbs, err := s.resolve(parentPath)
	if err != nil {
		return "", err
	}
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
	abs, err := s.resolve(path)
	if err != nil {
		return err
	}
	data, err := encodeRequest(req)
	if err != nil {
		return err
	}
	return os.WriteFile(abs, data, 0o644)
}

func (s *Store) CreateFolder(parentPath, name string) (string, error) {
	if err := fsstore.ValidateName(name); err != nil {
		return "", err
	}
	parentAbs, err := s.resolve(parentPath)
	if err != nil {
		return "", err
	}
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
	oldAbs, err := s.resolve(path)
	if err != nil {
		return "", err
	}
	newAbs, err := s.resolve(newPath)
	if err != nil {
		return "", err
	}
	if err := os.Rename(oldAbs, newAbs); err != nil {
		return "", err
	}
	return newPath, nil
}

func (s *Store) Delete(path string) error {
	abs, err := s.resolve(path)
	if err != nil {
		return err
	}
	return os.RemoveAll(abs)
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
	parentAbs, err := s.resolve(parentPath)
	if err != nil {
		return err
	}
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
