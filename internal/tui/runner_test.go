package tui

import (
	"errors"
	"testing"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	historystore "github.com/dhitalkamal/parley/internal/history/infrastructure"
)

func TestHandleRunResult_AppendsToRunHistoryStore(t *testing.T) {
	projectRoot := t.TempDir()
	m := New(t.TempDir(), projectRoot)

	msg := runResultMsg{
		path: "users",
		results: []execution.RunResult{
			{RequestPath: "010_get.json", Method: collection.GET, URL: "https://example.com/users", StatusCode: 200, ElapsedMS: 15},
		},
	}
	m.handleRunResult(msg)

	runs, err := historystore.NewRunStore(projectRoot).ListRuns()
	if err != nil {
		t.Fatalf("list runs error: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("got %d run history entries, want 1", len(runs))
	}
	if runs[0].Path != "users" {
		t.Errorf("run path = %q, want users", runs[0].Path)
	}
	if len(runs[0].Results) != 1 || runs[0].Results[0].URL != "https://example.com/users" {
		t.Errorf("run results = %+v", runs[0].Results)
	}
	if runs[0].TotalMS != 15 {
		t.Errorf("run total ms = %d, want 15", runs[0].TotalMS)
	}
}

func TestHandleRunResult_ErrorDoesNotAppendToRunHistory(t *testing.T) {
	projectRoot := t.TempDir()
	m := New(t.TempDir(), projectRoot)

	m.handleRunResult(runResultMsg{err: errors.New("boom")})

	runs, err := historystore.NewRunStore(projectRoot).ListRuns()
	if err != nil {
		t.Fatalf("list runs error: %v", err)
	}
	if len(runs) != 0 {
		t.Errorf("expected no run history entry for a failed run, got %+v", runs)
	}
}

func TestStartCollectionRun_RecordsSelectedFolderDisplayName(t *testing.T) {
	collectionsRoot := t.TempDir()
	projectRoot := t.TempDir()
	m := New(collectionsRoot, projectRoot)

	folderPath, err := m.store.CreateFolder("", "users")
	if err != nil {
		t.Fatalf("create folder error: %v", err)
	}
	if _, err := m.store.SaveRequest(folderPath, "get", collection.Request{Method: collection.GET, URL: "https://example.com"}); err != nil {
		t.Fatalf("save request error: %v", err)
	}
	m.refreshTree()
	m.selectSidebarPath(folderPath)

	_, cmd := m.startCollectionRun()
	if cmd == nil {
		t.Fatal("expected startCollectionRun to return a command")
	}
	msg, ok := cmd().(runResultMsg)
	if !ok {
		t.Fatalf("expected a runResultMsg, got %T", cmd())
	}
	if msg.path != "users" {
		t.Errorf("recorded path = %q, want the folder's display name %q, not its raw store path %q", msg.path, "users", folderPath)
	}
}

func TestStartCollectionRun_WholeCollectionRecordsEmptyPath(t *testing.T) {
	collectionsRoot := t.TempDir()
	projectRoot := t.TempDir()
	m := New(collectionsRoot, projectRoot)

	if _, err := m.store.SaveRequest("", "ping", collection.Request{Method: collection.GET, URL: "https://example.com"}); err != nil {
		t.Fatalf("save request error: %v", err)
	}
	m.refreshTree()

	_, cmd := m.startCollectionRun()
	if cmd == nil {
		t.Fatal("expected startCollectionRun to return a command")
	}
	msg, ok := cmd().(runResultMsg)
	if !ok {
		t.Fatalf("expected a runResultMsg, got %T", cmd())
	}
	if msg.path != "" {
		t.Errorf("recorded path = %q, want empty for a whole-collection run", msg.path)
	}
}
