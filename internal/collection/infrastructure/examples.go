package collectionstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	"github.com/dhitalkamal/parley/internal/platform/fsstore"
)

// examplesSuffix marks the directory holding a request's saved response
// examples, kept as a sibling of the request file (e.g. "010_get-user.json"
// -> "010_get-user.examples/"). listSiblings filters this suffix out of
// Tree() so examples don't show up as browsable requests/folders.
const examplesSuffix = ".examples"

func examplesDirFor(requestPath string) string {
	return strings.TrimSuffix(requestPath, ".json") + examplesSuffix
}

func (s *Store) SaveExample(requestPath, name string, ex collection.Example) (string, error) {
	if err := fsstore.ValidateName(name); err != nil {
		return "", err
	}
	dir := filepath.Join(s.Root, examplesDirFor(requestPath))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	data, err := encodeExample(ex)
	if err != nil {
		return "", err
	}
	relPath := filepath.Join(examplesDirFor(requestPath), name+".json")
	if err := os.WriteFile(filepath.Join(s.Root, relPath), data, 0o644); err != nil {
		return "", err
	}
	return relPath, nil
}

func (s *Store) ListExamples(requestPath string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(s.Root, examplesDirFor(requestPath)))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			names = append(names, strings.TrimSuffix(e.Name(), ".json"))
		}
	}
	sort.Strings(names)
	return names, nil
}

func (s *Store) LoadExample(path string) (collection.Example, error) {
	data, err := os.ReadFile(filepath.Join(s.Root, path))
	if err != nil {
		return collection.Example{}, err
	}
	ex, err := decodeExample(data)
	if err != nil {
		return collection.Example{}, err
	}
	ex.Name = strings.TrimSuffix(filepath.Base(path), ".json")
	return ex, nil
}

type exampleFile struct {
	StatusCode int              `json:"statusCode"`
	Status     string           `json:"status,omitempty"`
	Headers    []fsstore.KVFile `json:"headers,omitempty"`
	Body       string           `json:"body,omitempty"`
}

func encodeExample(ex collection.Example) ([]byte, error) {
	f := exampleFile{StatusCode: ex.StatusCode, Status: ex.Status, Body: ex.Body}
	for _, h := range ex.Headers {
		f.Headers = append(f.Headers, fsstore.KVFile{Key: h.Key, Value: h.Value, Enabled: h.Enabled})
	}
	return json.MarshalIndent(f, "", "  ")
}

func decodeExample(data []byte) (collection.Example, error) {
	var f exampleFile
	if err := json.Unmarshal(data, &f); err != nil {
		return collection.Example{}, err
	}
	ex := collection.Example{StatusCode: f.StatusCode, Status: f.Status, Body: f.Body}
	for _, h := range f.Headers {
		ex.Headers = append(ex.Headers, collection.Header{Key: h.Key, Value: h.Value, Enabled: h.Enabled})
	}
	return ex, nil
}
