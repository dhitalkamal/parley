package environmentstore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnvStore_CorruptEnvironmentErrorNamesTheFile(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "environments")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "dev.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := New(root).LoadEnvironment("dev")
	if err == nil {
		t.Fatal("expected an error for a corrupt environment file")
	}
	if !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), "invalid environment file") {
		t.Errorf("error should name the file, got: %v", err)
	}
}

func TestEnvStore_CorruptGlobalsErrorNamesTheFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "globals.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := New(root).LoadGlobals()
	if err == nil || !strings.Contains(err.Error(), path) {
		t.Errorf("error should name globals.json path, got: %v", err)
	}
}
