package historystore

import (
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	history "github.com/dhitalkamal/parley/internal/history/domain"
	scripting "github.com/dhitalkamal/parley/internal/scripting/domain"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunStore_ListRunsOnMissingFileReturnsEmpty(t *testing.T) {
	s := NewRunStore(t.TempDir())
	entries, err := s.ListRuns()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("got %v, want none", entries)
	}
}

func TestRunStore_AppendThenListRoundTrips(t *testing.T) {
	s := NewRunStore(t.TempDir())
	entry := history.CollectionRunEntry{
		Time: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		Path: "users",
		Results: []execution.RunResult{
			{RequestPath: "010_get.json", Method: "GET", URL: "https://example.com/users", StatusCode: 200, Status: "200 OK", ElapsedMS: 42,
				Tests: []scripting.TestResult{{Name: "status is 200", Passed: true}}},
		},
		TotalMS: 42,
	}
	if err := s.AppendRun(entry); err != nil {
		t.Fatalf("append error: %v", err)
	}

	entries, err := s.ListRuns()
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	got := entries[0]
	if got.Path != "users" || got.TotalMS != 42 {
		t.Errorf("got %+v", got)
	}
	if !got.Time.Equal(entry.Time) {
		t.Errorf("time = %v, want %v", got.Time, entry.Time)
	}
	if len(got.Results) != 1 || got.Results[0].URL != "https://example.com/users" || len(got.Results[0].Tests) != 1 || !got.Results[0].Tests[0].Passed {
		t.Errorf("results = %+v", got.Results)
	}
}

func TestRunStore_ListRunsIsMostRecentFirst(t *testing.T) {
	s := NewRunStore(t.TempDir())
	first := history.CollectionRunEntry{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Path: "first"}
	second := history.CollectionRunEntry{Time: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), Path: "second"}
	if err := s.AppendRun(first); err != nil {
		t.Fatalf("append error: %v", err)
	}
	if err := s.AppendRun(second); err != nil {
		t.Fatalf("append error: %v", err)
	}

	entries, err := s.ListRuns()
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(entries) != 2 || entries[0].Path != "second" || entries[1].Path != "first" {
		t.Errorf("got %+v, want [second, first]", entries)
	}
}

func TestRunStore_RecordsFailedRequestError(t *testing.T) {
	s := NewRunStore(t.TempDir())
	entry := history.CollectionRunEntry{
		Time:    time.Now(),
		Path:    "ping",
		Results: []execution.RunResult{{RequestPath: "010_ping.json", Err: "connection refused"}},
	}
	if err := s.AppendRun(entry); err != nil {
		t.Fatalf("append error: %v", err)
	}
	entries, err := s.ListRuns()
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(entries) != 1 || len(entries[0].Results) != 1 || entries[0].Results[0].Err != "connection refused" {
		t.Errorf("got %+v", entries)
	}
}

func TestRunStore_DeleteRunRemovesMatchingEntryOnly(t *testing.T) {
	s := NewRunStore(t.TempDir())
	first := history.CollectionRunEntry{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Path: "first", TotalMS: 10}
	second := history.CollectionRunEntry{Time: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), Path: "second", TotalMS: 20}
	third := history.CollectionRunEntry{Time: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC), Path: "third", TotalMS: 30}
	for _, e := range []history.CollectionRunEntry{first, second, third} {
		if err := s.AppendRun(e); err != nil {
			t.Fatalf("append error: %v", err)
		}
	}

	entries, err := s.ListRuns()
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if err := s.DeleteRun(entries[1]); err != nil { // "second", the middle one
		t.Fatalf("delete error: %v", err)
	}

	after, err := s.ListRuns()
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(after) != 2 || after[0].Path != "third" || after[1].Path != "first" {
		t.Errorf("got %+v, want [third, first]", after)
	}
}

func TestRunStore_DeleteRunNotFoundIsANoOp(t *testing.T) {
	s := NewRunStore(t.TempDir())
	if err := s.AppendRun(history.CollectionRunEntry{Time: time.Now(), Path: "kept"}); err != nil {
		t.Fatalf("append error: %v", err)
	}

	notPresent := history.CollectionRunEntry{Time: time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC), Path: "ghost"}
	if err := s.DeleteRun(notPresent); err != nil {
		t.Fatalf("expected a no-op, got error: %v", err)
	}

	entries, err := s.ListRuns()
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(entries) != 1 || entries[0].Path != "kept" {
		t.Errorf("got %+v, want the original entry untouched", entries)
	}
}

func TestRunStore_DeleteRunOnMissingFileIsANoOp(t *testing.T) {
	s := NewRunStore(t.TempDir())
	if err := s.DeleteRun(history.CollectionRunEntry{Time: time.Now(), Path: "anything"}); err != nil {
		t.Fatalf("expected a no-op, got error: %v", err)
	}
}

func TestRunStore_StoredAsJSONLAtProjectRoot(t *testing.T) {
	root := t.TempDir()
	s := NewRunStore(root)
	if err := s.AppendRun(history.CollectionRunEntry{Time: time.Now(), Path: "ping"}); err != nil {
		t.Fatalf("append error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "runs.jsonl")); err != nil {
		t.Errorf("expected runs.jsonl at project root: %v", err)
	}
}
