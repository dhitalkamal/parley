package collectionstore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStore_CorruptRequestErrorNamesTheFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "010_ping.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := New(root).LoadRequest("010_ping.json")
	if err == nil {
		t.Fatal("expected an error for a corrupt request file")
	}
	if !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), "invalid request file") {
		t.Errorf("error should name the file, got: %v", err)
	}
}

func TestStore_CorruptExampleErrorNamesTheFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "ok.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := New(root).LoadExample("ok.json")
	if err == nil || !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), "invalid example file") {
		t.Errorf("error should name the file, got: %v", err)
	}
}
