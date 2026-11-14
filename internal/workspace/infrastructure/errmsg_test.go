package workspacestore

import (
	"os"
	"strings"
	"testing"
)

func TestWorkspaceStore_CorruptRegistryErrorNamesTheFile(t *testing.T) {
	path := registryPath(t)
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := New(path).List()
	if err == nil {
		t.Fatal("expected an error for a corrupt registry")
	}
	if !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), "not valid JSON") {
		t.Errorf("error should name the file and say it is invalid JSON, got: %v", err)
	}
}
