package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	collectionstore "github.com/dhitalkamal/parley/internal/collection/infrastructure"
	historystore "github.com/dhitalkamal/parley/internal/history/infrastructure"
	workspace "github.com/dhitalkamal/parley/internal/workspace/domain"
	workspacestore "github.com/dhitalkamal/parley/internal/workspace/infrastructure"
)

func TestSelectWorkspace_NoWorkspacesReturnsError(t *testing.T) {
	_, err := selectWorkspace(nil, "")
	if err == nil {
		t.Fatal("expected an error when there are no workspaces")
	}
}

func TestSelectWorkspace_MatchingNameReturnsIt(t *testing.T) {
	workspaces := []workspace.Workspace{{Name: "Personal"}, {Name: "Work"}}
	got, err := selectWorkspace(workspaces, "Work")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "Work" {
		t.Errorf("got %q, want Work", got.Name)
	}
}

func TestSelectWorkspace_UnknownNameFallsBackToFirst(t *testing.T) {
	workspaces := []workspace.Workspace{{Name: "Personal"}, {Name: "Work"}}
	got, err := selectWorkspace(workspaces, "gone")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "Personal" {
		t.Errorf("got %q, want Personal (soft fallback to the first workspace)", got.Name)
	}
}

func TestWorkspaceExists_TrueForKnownNameFalseOtherwise(t *testing.T) {
	workspaces := []workspace.Workspace{{Name: "Personal"}}
	if !workspaceExists(workspaces, "Personal") {
		t.Error("expected Personal to exist")
	}
	if workspaceExists(workspaces, "gone") {
		t.Error("expected gone to not exist")
	}
}

// seedHome creates a parley home directory at dir with one workspace named
// wsName, active, whose collection root has a single saved request named
// "ping" pointing at url.
func seedHome(t *testing.T, dir, wsName, url string) {
	t.Helper()
	collectionsRoot := filepath.Join(dir, wsName, "collections")
	store := collectionstore.New(collectionsRoot)
	if _, err := store.SaveRequest("", "ping", collection.Request{
		Method:     collection.GET,
		URL:        url,
		TestScript: `pm.test("status is 200", function () { pm.expect(pm.response.code).to.equal(200); });`,
	}); err != nil {
		t.Fatalf("seed save request: %v", err)
	}
	wsStore := workspacestore.New(filepath.Join(dir, "workspaces.json"))
	ws := workspace.Workspace{Name: wsName, CollectionsRoot: collectionsRoot, ProjectRoot: filepath.Join(dir, wsName)}
	if err := wsStore.Save(ws); err != nil {
		t.Fatalf("seed save workspace: %v", err)
	}
	if err := wsStore.SetActiveName(wsName); err != nil {
		t.Fatalf("seed set active: %v", err)
	}
}

func TestRunCLICommand_NoPathRunsWholeCollection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	home := t.TempDir()
	seedHome(t, home, "Personal", server.URL)

	// no path arg means "the whole collection" per the README and
	// cli.ResolveRequestPaths - it must run, not print a usage error.
	var stdout, stderr bytes.Buffer
	code := runCLICommand([]string{}, home, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "PASS") {
		t.Errorf("expected PASS in stdout, got:\n%s", stdout.String())
	}
}

func TestRunCLICommand_TooManyArgsReturnsUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runCLICommand([]string{"one", "two"}, t.TempDir(), &stdout, &stderr)
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestDispatch_NoArgsLaunchesTUI(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, launchTUI := dispatch(nil, t.TempDir(), &stdout, &stderr)
	if !launchTUI {
		t.Error("expected bare parley to launch the TUI")
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

func TestDispatch_VersionPrintsVersionWithoutTUI(t *testing.T) {
	for _, arg := range []string{"version", "--version", "-v"} {
		var stdout, stderr bytes.Buffer
		code, launchTUI := dispatch([]string{arg}, t.TempDir(), &stdout, &stderr)
		if launchTUI {
			t.Errorf("%s: expected no TUI launch", arg)
		}
		if code != 0 {
			t.Errorf("%s: exit code = %d, want 0", arg, code)
		}
		if !strings.Contains(stdout.String(), "parley") {
			t.Errorf("%s: expected version output to mention parley, got:\n%s", arg, stdout.String())
		}
	}
}

func TestDispatch_HelpPrintsUsageWithoutTUI(t *testing.T) {
	for _, arg := range []string{"help", "--help", "-h"} {
		var stdout, stderr bytes.Buffer
		code, launchTUI := dispatch([]string{arg}, t.TempDir(), &stdout, &stderr)
		if launchTUI {
			t.Errorf("%s: expected no TUI launch", arg)
		}
		if code != 0 {
			t.Errorf("%s: exit code = %d, want 0", arg, code)
		}
		out := stdout.String()
		if !strings.Contains(out, "Usage") || !strings.Contains(out, "run") {
			t.Errorf("%s: expected usage text mentioning run, got:\n%s", arg, out)
		}
	}
}

func TestDispatch_UnknownCommandReturnsUsageErrorWithoutTUI(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, launchTUI := dispatch([]string{"bogus"}, t.TempDir(), &stdout, &stderr)
	if launchTUI {
		t.Error("expected an unknown command to error, not launch the TUI")
	}
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Errorf("expected an unknown-command message on stderr, got:\n%s", stderr.String())
	}
}

func TestDispatch_RunDelegatesToRunCLICommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	home := t.TempDir()
	seedHome(t, home, "Personal", server.URL)

	var stdout, stderr bytes.Buffer
	code, launchTUI := dispatch([]string{"run", "ping"}, home, &stdout, &stderr)
	if launchTUI {
		t.Error("expected run to execute headlessly, not launch the TUI")
	}
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstderr:\n%s", code, stderr.String())
	}
}

func TestRunCLICommand_RunsAgainstActiveWorkspaceAndReturnsSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	home := t.TempDir()
	seedHome(t, home, "Personal", server.URL)

	var stdout, stderr bytes.Buffer
	code := runCLICommand([]string{"ping"}, home, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "PASS") {
		t.Errorf("expected PASS in stdout, got:\n%s", stdout.String())
	}
}

func TestRunCLICommand_RecordsRunHistoryUnderWorkspaceProjectRoot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	home := t.TempDir()
	seedHome(t, home, "Personal", server.URL)

	var stdout, stderr bytes.Buffer
	code := runCLICommand([]string{"ping"}, home, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstderr:\n%s", code, stderr.String())
	}

	runStore := historystore.NewRunStore(filepath.Join(home, "Personal"))
	runs, err := runStore.ListRuns()
	if err != nil {
		t.Fatalf("list runs error: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("got %d run history entries, want 1", len(runs))
	}
	if runs[0].Path != "ping" {
		t.Errorf("run path = %q, want ping", runs[0].Path)
	}
}

func TestRunCLICommand_UnknownWorkspaceFlagReturnsUsageError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	home := t.TempDir()
	seedHome(t, home, "Personal", server.URL)

	var stdout, stderr bytes.Buffer
	code := runCLICommand([]string{"--workspace", "NoSuchWorkspace", "ping"}, home, &stdout, &stderr)
	if code != 2 {
		t.Errorf("exit code = %d, want 2\nstderr:\n%s", code, stderr.String())
	}
}

func TestRunCLICommand_ExplicitWorkspaceFlagSelectsThatWorkspace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	home := t.TempDir()
	seedHome(t, home, "Personal", "http://127.0.0.1:1") // active workspace: unreachable
	seedHome(t, home, "Work", server.URL)               // non-active workspace: reachable

	var stdout, stderr bytes.Buffer
	code := runCLICommand([]string{"--workspace", "Work", "ping"}, home, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
}

func TestRunCLICommand_UnknownReporterReturnsUsageError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	home := t.TempDir()
	seedHome(t, home, "Personal", server.URL)

	var stdout, stderr bytes.Buffer
	code := runCLICommand([]string{"--reporter", "bogus", "ping"}, home, &stdout, &stderr)
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}
