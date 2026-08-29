package workspacestore

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/dhitalkamal/parley/internal/platform/fsstore"
	workspace "github.com/dhitalkamal/parley/internal/workspace/domain"
)

// WorkspaceStore implements workspace.WorkspaceStore, persisting the whole
// registry (every known workspace, plus which one is active) as a single
// JSON file at RegistryPath - unlike EnvStore/Store, this isn't rooted
// under any one workspace's own directory, since it's the thing that picks
// which workspace's roots to use in the first place.
type WorkspaceStore struct {
	RegistryPath string
}

var _ workspace.WorkspaceStore = (*WorkspaceStore)(nil)

// New returns a WorkspaceStore backed by the JSON file at
// registryPath. The file need not exist yet - it's created lazily on first
// Save/SetActiveName.
func New(registryPath string) *WorkspaceStore {
	return &WorkspaceStore{RegistryPath: registryPath}
}

type workspaceFile struct {
	Active     string                `json:"active"`
	Workspaces []workspace.Workspace `json:"workspaces"`
}

func (s *WorkspaceStore) load() (workspaceFile, error) {
	data, err := os.ReadFile(s.RegistryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return workspaceFile{}, nil
		}
		return workspaceFile{}, err
	}
	var f workspaceFile
	if err := json.Unmarshal(data, &f); err != nil {
		return workspaceFile{}, err
	}
	return f, nil
}

func (s *WorkspaceStore) save(f workspaceFile) error {
	if err := os.MkdirAll(filepath.Dir(s.RegistryPath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.RegistryPath, data, 0o644)
}

func (s *WorkspaceStore) List() ([]workspace.Workspace, error) {
	f, err := s.load()
	if err != nil {
		return nil, err
	}
	return f.Workspaces, nil
}

// Save adds ws to the registry, or overwrites the existing entry with the
// same name.
func (s *WorkspaceStore) Save(ws workspace.Workspace) error {
	if err := fsstore.ValidateName(ws.Name); err != nil {
		return err
	}
	f, err := s.load()
	if err != nil {
		return err
	}
	for i, e := range f.Workspaces {
		if e.Name == ws.Name {
			f.Workspaces[i] = ws
			return s.save(f)
		}
	}
	f.Workspaces = append(f.Workspaces, ws)
	return s.save(f)
}

func (s *WorkspaceStore) Delete(name string) error {
	f, err := s.load()
	if err != nil {
		return err
	}
	kept := f.Workspaces[:0]
	for _, e := range f.Workspaces {
		if e.Name != name {
			kept = append(kept, e)
		}
	}
	f.Workspaces = kept
	return s.save(f)
}

func (s *WorkspaceStore) ActiveName() (string, error) {
	f, err := s.load()
	if err != nil {
		return "", err
	}
	return f.Active, nil
}

func (s *WorkspaceStore) SetActiveName(name string) error {
	f, err := s.load()
	if err != nil {
		return err
	}
	f.Active = name
	return s.save(f)
}
