package fsstore

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func skipOnWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("unix permission bits")
	}
}

func perm(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode().Perm()
}

// writeMode creates path with the given mode regardless of umask.
func writeMode(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

func TestWriteOwnerOnly_CreatesAndTightens(t *testing.T) {
	skipOnWindows(t)
	dir := t.TempDir()
	fresh := filepath.Join(dir, "fresh")
	if err := WriteOwnerOnly(fresh, []byte("a")); err != nil {
		t.Fatal(err)
	}
	if got := perm(t, fresh); got != 0o600 {
		t.Errorf("fresh perm = %o, want 600", got)
	}

	old := filepath.Join(dir, "old")
	writeMode(t, old, 0o644)
	if err := WriteOwnerOnly(old, []byte("new")); err != nil {
		t.Fatal(err)
	}
	if got := perm(t, old); got != 0o600 {
		t.Errorf("existing perm = %o, want 600", got)
	}
	if b, _ := os.ReadFile(old); string(b) != "new" {
		t.Errorf("content = %q, want new", b)
	}
}

func TestTightenSecretFiles_TightensListedFilesOnly(t *testing.T) {
	skipOnWindows(t)
	root := t.TempDir()
	secret := []string{
		"environments/a.json", "globals.json", "active_environment",
		"history.jsonl", "last_responses.json", "runs.jsonl",
	}
	untouched := []string{
		"collections/req.json", "workspaces.json", "environments/sub/deep.json",
		"environments/notes.txt",
	}
	for _, p := range append(append([]string{}, secret...), untouched...) {
		writeMode(t, filepath.Join(root, p), 0o644)
	}

	if err := TightenSecretFiles(root); err != nil {
		t.Fatal(err)
	}
	for _, p := range secret {
		if got := perm(t, filepath.Join(root, p)); got != 0o600 {
			t.Errorf("%s perm = %o, want 600", p, got)
		}
	}
	for _, p := range untouched {
		if got := perm(t, filepath.Join(root, p)); got != 0o644 {
			t.Errorf("%s perm = %o, want untouched 644", p, got)
		}
	}
	if err := TightenSecretFiles(root); err != nil {
		t.Errorf("second run: %v", err)
	}
}

func TestTightenSecretFiles_MissingRootAndFilesAreSilent(t *testing.T) {
	if err := TightenSecretFiles(filepath.Join(t.TempDir(), "nope")); err != nil {
		t.Errorf("missing root: %v", err)
	}
	if err := TightenSecretFiles(t.TempDir()); err != nil {
		t.Errorf("empty root: %v", err)
	}
}

func TestTightenSecretFiles_LeavesPrivateFilesAlone(t *testing.T) {
	skipOnWindows(t)
	root := t.TempDir()
	p := filepath.Join(root, "globals.json")
	writeMode(t, p, 0o400) // private but not 0600: must not be rewritten to 0600
	if err := TightenSecretFiles(root); err != nil {
		t.Fatal(err)
	}
	if got := perm(t, p); got != 0o400 {
		t.Errorf("perm = %o, want 400 untouched", got)
	}
}

func TestTightenSecretFiles_SkipsSymlinks(t *testing.T) {
	skipOnWindows(t)
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.json")
	writeMode(t, outside, 0o644)
	if err := os.MkdirAll(filepath.Join(root, "environments"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, link := range []string{"globals.json", "environments/x.json"} {
		if err := os.Symlink(outside, filepath.Join(root, link)); err != nil {
			t.Fatal(err)
		}
	}
	if err := TightenSecretFiles(root); err != nil {
		t.Fatal(err)
	}
	if got := perm(t, outside); got != 0o644 {
		t.Errorf("symlink target perm = %o, want untouched 644", got)
	}
}

func TestTightenSecretFiles_KeepsGoingAndJoinsFailures(t *testing.T) {
	skipOnWindows(t)
	root := t.TempDir()
	good := filepath.Join(root, "runs.jsonl")
	writeMode(t, good, 0o644)
	// an unreadable environments dir makes the listing fail
	envDir := filepath.Join(root, "environments")
	writeMode(t, filepath.Join(envDir, "a.json"), 0o644)
	if err := os.Chmod(envDir, 0o000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(envDir, 0o755)
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}

	err := TightenSecretFiles(root)
	if err == nil {
		t.Fatal("expected an error for the unreadable environments dir")
	}
	if !strings.Contains(err.Error(), "environments") {
		t.Errorf("error should name the failing path: %v", err)
	}
	if got := perm(t, good); got != 0o600 {
		t.Errorf("runs.jsonl perm = %o, want 600 (sweep must continue)", got)
	}
}
