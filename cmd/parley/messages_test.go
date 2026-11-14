package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCLIMessage_UnknownWorkspaceListsAvailableOnes(t *testing.T) {
	home := t.TempDir()
	seedHome(t, home, "Personal", "http://127.0.0.1:1")
	var stdout, stderr bytes.Buffer
	code := runCLICommand([]string{"--workspace", "nope"}, home, &stdout, &stderr)
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	got := stderr.String()
	for _, want := range []string{`no workspace named "nope"`, "available: Personal"} {
		if !strings.Contains(got, want) {
			t.Errorf("stderr %q is missing %q", got, want)
		}
	}
}

func TestRunCLIMessage_ExtraPathArgsNamesThemAndHintsAtQuoting(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLICommand([]string{"my", "folder"}, t.TempDir(), &stdout, &stderr)
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	got := stderr.String()
	for _, want := range []string{"at most one path", "got 2", "quote", "usage: parley run"} {
		if !strings.Contains(got, want) {
			t.Errorf("stderr %q is missing %q", got, want)
		}
	}
}

func TestRunCLIMessage_UnwritableOutFileNamesTheFlagAndPath(t *testing.T) {
	home := t.TempDir()
	seedHome(t, home, "Personal", "http://127.0.0.1:1")
	bad := filepath.Join(home, "no-such-dir", "report.txt")
	var stdout, stderr bytes.Buffer
	code := runCLICommand([]string{"--out", bad}, home, &stdout, &stderr)
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	got := stderr.String()
	for _, want := range []string{"cannot create the --out file", bad, "directory exists"} {
		if !strings.Contains(got, want) {
			t.Errorf("stderr %q is missing %q", got, want)
		}
	}
}
