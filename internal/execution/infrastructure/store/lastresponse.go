package executionstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
)

// LastResponseStore implements execution.LastResponseStore: one JSON file at
// the project root mapping each saved request's Store path to whatever it
// last returned - so the Response panel can restore that after the app
// restarts, not just for the rest of the current session.
type LastResponseStore struct {
	ProjectRoot string
}

var _ execution.LastResponseStore = (*LastResponseStore)(nil)

// New returns a LastResponseStore rooted at projectRoot.
// The file need not exist yet; it's created lazily on first Save.
func New(projectRoot string) *LastResponseStore {
	return &LastResponseStore{ProjectRoot: projectRoot}
}

func (s *LastResponseStore) path() string {
	return filepath.Join(s.ProjectRoot, "last_responses.json")
}

type lastResponseEntry struct {
	StatusCode int                 `json:"statusCode"`
	Status     string              `json:"status"`
	Headers    []collection.Header `json:"headers,omitempty"`
	Body       []byte              `json:"body,omitempty"`
	ElapsedMS  int64               `json:"elapsedMs"`
}

func (s *LastResponseStore) readAll() (map[string]lastResponseEntry, error) {
	data, err := os.ReadFile(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]lastResponseEntry{}, nil
		}
		return nil, err
	}
	var entries map[string]lastResponseEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	if entries == nil {
		entries = map[string]lastResponseEntry{}
	}
	return entries, nil
}

func (s *LastResponseStore) writeAll(entries map[string]lastResponseEntry) error {
	if err := os.MkdirAll(s.ProjectRoot, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path(), data, 0o644)
}

func (s *LastResponseStore) Save(requestPath string, resp execution.Response, elapsedMS int64) error {
	entries, err := s.readAll()
	if err != nil {
		return err
	}
	entries[requestPath] = lastResponseEntry{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    resp.Headers,
		Body:       resp.Body,
		ElapsedMS:  elapsedMS,
	}
	return s.writeAll(entries)
}

func (s *LastResponseStore) Load(requestPath string) (resp execution.Response, elapsedMS int64, ok bool, err error) {
	entries, err := s.readAll()
	if err != nil {
		return execution.Response{}, 0, false, err
	}
	entry, found := entries[requestPath]
	if !found {
		return execution.Response{}, 0, false, nil
	}
	return execution.Response{
		StatusCode: entry.StatusCode,
		Status:     entry.Status,
		Headers:    entry.Headers,
		Body:       entry.Body,
	}, entry.ElapsedMS, true, nil
}

// Delete removes the entry for requestPath. If requestPath is a folder, it
// also removes every entry underneath it (keys prefixed by requestPath +
// separator), matching the collection Store's recursive folder delete so no
// stale responses linger for requests that no longer exist on disk.
func (s *LastResponseStore) Delete(requestPath string) error {
	entries, err := s.readAll()
	if err != nil {
		return err
	}
	prefix := requestPath + string(filepath.Separator)
	changed := false
	for key := range entries {
		if key == requestPath || strings.HasPrefix(key, prefix) {
			delete(entries, key)
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return s.writeAll(entries)
}
