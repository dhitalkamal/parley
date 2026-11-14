package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunCLICommand_BadFlagReturnsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLICommand([]string{"--nope"}, t.TempDir(), &stdout, &stderr)
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestRunCLICommand_NoWorkspacesReturnsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLICommand(nil, t.TempDir(), &stdout, &stderr)
	if code != 2 {
		t.Errorf("exit code = %d, want 2\nstderr:\n%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "no workspaces") {
		t.Errorf("stderr should mention missing workspaces: %q", stderr.String())
	}
}

func TestRunCLICommand_CorruptRegistryReturnsUsageError(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "workspaces.json"), []byte("{bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := runCLICommand(nil, home, &stdout, &stderr); code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	if !strings.HasPrefix(stderr.String(), "error:") {
		t.Errorf("stderr = %q, want an error: line", stderr.String())
	}
}

func TestRunCLICommand_OutFlagWritesReportToFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	home := t.TempDir()
	seedHome(t, home, "Personal", server.URL)

	out := filepath.Join(t.TempDir(), "report.txt")
	var stdout, stderr bytes.Buffer
	code := runCLICommand([]string{"--out", out, "ping"}, home, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d\nstderr:\n%s", code, stderr.String())
	}
	data, err := os.ReadFile(out)
	if err != nil || len(data) == 0 {
		t.Fatalf("report file empty or missing: %v", err)
	}
	if stdout.Len() != 0 {
		t.Errorf("report leaked to stdout: %q", stdout.String())
	}
}

func TestRunCLICommand_OutPathUncreatableReturnsUsageError(t *testing.T) {
	home := t.TempDir()
	seedHome(t, home, "Personal", "http://127.0.0.1:1")
	bad := filepath.Join(t.TempDir(), "missing-dir", "report.txt")
	var stdout, stderr bytes.Buffer
	if code := runCLICommand([]string{"--out", bad, "ping"}, home, &stdout, &stderr); code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}
