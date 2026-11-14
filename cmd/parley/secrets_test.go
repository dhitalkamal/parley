package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	workspace "github.com/dhitalkamal/parley/internal/workspace/domain"
)

func writeLoose(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
}

func modeOf(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode().Perm()
}

func TestRun_TightensLooseEnvFileInWorkspace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix permission bits")
	}
	home := t.TempDir()
	seedHome(t, home, "Personal", "http://127.0.0.1:1")
	env := filepath.Join(home, "Personal", "environments", "x.json")
	writeLoose(t, env)

	var stdout, stderr bytes.Buffer
	dispatch([]string{"run"}, home, &stdout, &stderr)

	if got := modeOf(t, env); got != 0o600 {
		t.Errorf("env perm = %o, want 600", got)
	}
	if strings.Contains(stderr.String(), "restrict permissions") {
		t.Errorf("unexpected warning: %s", stderr.String())
	}
}

func TestRun_TightensEveryRegisteredWorkspace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix permission bits")
	}
	home := t.TempDir()
	seedHome(t, home, "Other", "http://127.0.0.1:1")
	seedHome(t, home, "Personal", "http://127.0.0.1:1")
	other := filepath.Join(home, "Other", "globals.json")
	writeLoose(t, other)

	var stdout, stderr bytes.Buffer
	dispatch([]string{"run"}, home, &stdout, &stderr)

	if got := modeOf(t, other); got != 0o600 {
		t.Errorf("non-active workspace globals perm = %o, want 600", got)
	}
}

func TestRun_PermissionFailureWarnsOnceAndKeepsExitCode(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs unix permissions and a non-root user")
	}
	home := t.TempDir()
	seedHome(t, home, "Personal", "http://127.0.0.1:1")
	envDir := filepath.Join(home, "Personal", "environments")
	writeLoose(t, filepath.Join(envDir, "x.json"))
	if err := os.Chmod(envDir, 0o000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(envDir, 0o755)

	var out1, err1, out2, err2 bytes.Buffer
	withFailure, _ := dispatch([]string{"run"}, home, &out1, &err1)
	// same run with nothing to fail on gives the baseline exit code
	if err := os.Chmod(envDir, 0o755); err != nil {
		t.Fatal(err)
	}
	baseline, _ := dispatch([]string{"run"}, home, &out2, &err2)

	if withFailure != baseline {
		t.Errorf("exit code = %d with a chmod failure, baseline %d", withFailure, baseline)
	}
	n := strings.Count(err1.String(), "could not restrict permissions")
	if n != 1 {
		t.Errorf("warning count = %d, want 1\nstderr:\n%s", n, err1.String())
	}
	if !strings.HasPrefix(err1.String(), "parley: warning: could not restrict permissions on ") {
		t.Errorf("unexpected warning text: %q", err1.String())
	}
}

func TestBuildModel_TightensLooseFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix permission bits")
	}
	home := t.TempDir()
	t.Setenv("PARLEY_HOME", home)
	seedHome(t, home, "Personal", "http://127.0.0.1:1")
	hist := filepath.Join(home, "Personal", "history.jsonl")
	writeLoose(t, hist)

	if _, err := buildModel(); err != nil {
		t.Fatal(err)
	}
	if got := modeOf(t, hist); got != 0o600 {
		t.Errorf("history perm = %o, want 600", got)
	}
}

func TestCountErrors_FlattensNestedJoins(t *testing.T) {
	a, b, c := errors.New("a"), errors.New("b"), errors.New("c")
	err := errors.Join(errors.Join(a, b), c)
	if got := countErrors(err); got != 3 {
		t.Errorf("countErrors = %d, want 3", got)
	}
	if got := countErrors(nil); got != 0 {
		t.Errorf("countErrors(nil) = %d, want 0", got)
	}
}

func TestTightenWorkspaces_DedupesRoots(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs unix permissions and a non-root user")
	}
	root := t.TempDir()
	envDir := filepath.Join(root, "environments")
	writeLoose(t, filepath.Join(envDir, "x.json"))
	if err := os.Chmod(envDir, 0o000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(envDir, 0o755)

	ws := []workspace.Workspace{{Name: "a", ProjectRoot: root}, {Name: "b", ProjectRoot: root}}
	if got := countErrors(tightenWorkspaces(ws)); got != 1 {
		t.Errorf("failures = %d, want 1 (shared root swept once)", got)
	}
}
