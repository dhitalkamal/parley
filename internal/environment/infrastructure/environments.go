package environmentstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	environment "github.com/dhitalkamal/parley/internal/environment/domain"
	"github.com/dhitalkamal/parley/internal/platform/fsstore"
)

// EnvStore implements environment.EnvironmentStore. Environments and globals live
// at the project root, as siblings of collections/ - see the storage layout
// in the project brief.
type EnvStore struct {
	ProjectRoot string
}

var _ environment.EnvironmentStore = (*EnvStore)(nil)

// New returns an EnvStore rooted at projectRoot. Neither
// environments/ nor globals.json need exist yet; both are created lazily.
func New(projectRoot string) *EnvStore {
	return &EnvStore{ProjectRoot: projectRoot}
}

func (s *EnvStore) environmentsDir() string {
	return filepath.Join(s.ProjectRoot, "environments")
}

func (s *EnvStore) ListEnvironments() ([]string, error) {
	entries, err := os.ReadDir(s.environmentsDir())
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

func (s *EnvStore) LoadEnvironment(name string) (environment.Environment, error) {
	data, err := os.ReadFile(filepath.Join(s.environmentsDir(), name+".json"))
	if err != nil {
		return environment.Environment{}, err
	}
	env, err := decodeEnvironment(data)
	if err != nil {
		return environment.Environment{}, err
	}
	env.Name = name
	return env, nil
}

func (s *EnvStore) SaveEnvironment(env environment.Environment) error {
	if err := fsstore.ValidateName(env.Name); err != nil {
		return err
	}
	// environments/ can hold secret-flagged variables and captured auth tokens,
	// so keep the dir owner-only (0700) and each file owner read/write only
	// (0600). Never world/group readable.
	if err := os.MkdirAll(s.environmentsDir(), 0o700); err != nil {
		return err
	}
	data, err := encodeEnvironment(env)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.environmentsDir(), env.Name+".json"), data, 0o600)
}

func (s *EnvStore) DeleteEnvironment(name string) error {
	return os.Remove(filepath.Join(s.environmentsDir(), name+".json"))
}

func (s *EnvStore) globalsPath() string {
	return filepath.Join(s.ProjectRoot, "globals.json")
}

func (s *EnvStore) LoadGlobals() (environment.Environment, error) {
	data, err := os.ReadFile(s.globalsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return environment.Environment{Name: "globals"}, nil
		}
		return environment.Environment{}, err
	}
	env, err := decodeEnvironment(data)
	if err != nil {
		return environment.Environment{}, err
	}
	env.Name = "globals"
	return env, nil
}

func (s *EnvStore) SaveGlobals(env environment.Environment) error {
	if err := os.MkdirAll(s.ProjectRoot, 0o755); err != nil {
		return err
	}
	data, err := encodeEnvironment(env)
	if err != nil {
		return err
	}
	// globals.json can hold secret-flagged variables, so keep it owner-only.
	return os.WriteFile(s.globalsPath(), data, 0o600)
}

func (s *EnvStore) activeNamePath() string {
	return filepath.Join(s.ProjectRoot, "active_environment")
}

// ActiveName returns the environment name last passed to SetActiveName, or
// "" if none has been set yet (a fresh project, or one that's never had an
// environment made active).
func (s *EnvStore) ActiveName() (string, error) {
	data, err := os.ReadFile(s.activeNamePath())
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// SetActiveName persists which environment is active, so a restart can
// restore it - see ActiveName. Passing "" clears it (e.g. the active
// environment was just deleted).
func (s *EnvStore) SetActiveName(name string) error {
	if name == "" {
		err := os.Remove(s.activeNamePath())
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(s.ProjectRoot, 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.activeNamePath(), []byte(name), 0o600)
}

type variableFile struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
	Secret  bool   `json:"secret,omitempty"`
}

type environmentFile struct {
	// ExpectedVPN is omitempty so environments without one (and every file
	// written before this field existed) stay byte-for-byte the same and load
	// back with an empty expectation.
	ExpectedVPN string         `json:"expectedVpn,omitempty"`
	Variables   []variableFile `json:"variables,omitempty"`
}

func encodeEnvironment(env environment.Environment) ([]byte, error) {
	f := environmentFile{ExpectedVPN: env.ExpectedVPN}
	for _, v := range env.Variables {
		f.Variables = append(f.Variables, variableFile{Key: v.Key, Value: v.Value, Enabled: v.Enabled, Secret: v.Secret})
	}
	return json.MarshalIndent(f, "", "  ")
}

func decodeEnvironment(data []byte) (environment.Environment, error) {
	var f environmentFile
	if err := json.Unmarshal(data, &f); err != nil {
		return environment.Environment{}, err
	}
	env := environment.Environment{ExpectedVPN: f.ExpectedVPN}
	for _, v := range f.Variables {
		env.Variables = append(env.Variables, environment.Variable{Key: v.Key, Value: v.Value, Enabled: v.Enabled, Secret: v.Secret})
	}
	return env, nil
}
