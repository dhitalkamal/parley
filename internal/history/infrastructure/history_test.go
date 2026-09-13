package historystore

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	history "github.com/dhitalkamal/parley/internal/history/domain"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHistoryStore_ListHistoryOnMissingFileReturnsEmpty(t *testing.T) {
	s := New(t.TempDir())
	entries, err := s.ListHistory()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("got %v, want none", entries)
	}
}

func TestHistoryStore_AppendThenListRoundTrips(t *testing.T) {
	s := New(t.TempDir())
	entry := history.HistoryEntry{
		Time:       time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		Request:    collection.Request{Method: collection.GET, URL: "https://example.com"},
		StatusCode: 200,
		Status:     "200 OK",
		ElapsedMS:  42,
	}
	if err := s.AppendHistory(entry); err != nil {
		t.Fatalf("append error: %v", err)
	}

	entries, err := s.ListHistory()
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	got := entries[0]
	if got.Request.URL != entry.Request.URL || got.StatusCode != 200 || got.ElapsedMS != 42 {
		t.Errorf("got %+v", got)
	}
	if !got.Time.Equal(entry.Time) {
		t.Errorf("time = %v, want %v", got.Time, entry.Time)
	}
}

func TestHistoryStore_ListHistoryIsMostRecentFirst(t *testing.T) {
	s := New(t.TempDir())
	first := history.HistoryEntry{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Request: collection.Request{URL: "https://example.com/first"}}
	second := history.HistoryEntry{Time: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), Request: collection.Request{URL: "https://example.com/second"}}
	if err := s.AppendHistory(first); err != nil {
		t.Fatalf("append error: %v", err)
	}
	if err := s.AppendHistory(second); err != nil {
		t.Fatalf("append error: %v", err)
	}

	entries, err := s.ListHistory()
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(entries) != 2 || entries[0].Request.URL != "https://example.com/second" || entries[1].Request.URL != "https://example.com/first" {
		t.Errorf("got %+v, want [second, first]", entries)
	}
}

func TestHistoryStore_RecordsFailedSendError(t *testing.T) {
	s := New(t.TempDir())
	entry := history.HistoryEntry{Time: time.Now(), Request: collection.Request{URL: "https://example.com"}, Err: "connection refused"}
	if err := s.AppendHistory(entry); err != nil {
		t.Fatalf("append error: %v", err)
	}
	entries, err := s.ListHistory()
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(entries) != 1 || entries[0].Err != "connection refused" {
		t.Errorf("got %+v", entries)
	}
}

// TestHistoryStore_RoundTripsRequestPath guards the Collections screen's
// per-request "Recent History" panel, which filters ListHistory's results
// by this field to show only sends that came from a specific saved request.
func TestHistoryStore_RoundTripsRequestPath(t *testing.T) {
	s := New(t.TempDir())
	entry := history.HistoryEntry{
		Time:        time.Now(),
		RequestPath: "010_users/020_get.json",
		Request:     collection.Request{Method: collection.GET, URL: "https://example.com/users"},
	}
	if err := s.AppendHistory(entry); err != nil {
		t.Fatalf("append error: %v", err)
	}
	entries, err := s.ListHistory()
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(entries) != 1 || entries[0].RequestPath != "010_users/020_get.json" {
		t.Errorf("got %+v, want RequestPath round-tripped", entries)
	}
}

// TestHistoryStore_ReadsOlderLinesMissingRequestPath guards backward
// compatibility: history.jsonl lines written before this field existed
// have no "requestPath" key at all, and must still decode (with an empty
// RequestPath) rather than fail to parse.
func TestHistoryStore_ReadsOlderLinesMissingRequestPath(t *testing.T) {
	root := t.TempDir()
	oldLine := `{"time":"2026-01-01T00:00:00Z","request":{"method":"GET","url":"https://example.com"},"statusCode":200}`
	if err := os.WriteFile(filepath.Join(root, "history.jsonl"), []byte(oldLine+"\n"), 0o644); err != nil {
		t.Fatalf("seed write error: %v", err)
	}

	entries, err := New(root).ListHistory()
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	if len(entries) != 1 || entries[0].RequestPath != "" || entries[0].StatusCode != 200 {
		t.Errorf("got %+v, want one entry with an empty RequestPath", entries)
	}
}

// TestHistoryStore_AppendUsesOwnerOnlyPerms guards that history.jsonl, which
// stores full requests verbatim (Authorization headers, secret params/body),
// is not readable by other local users or broad-scope backup tools.
func TestHistoryStore_AppendUsesOwnerOnlyPerms(t *testing.T) {
	root := t.TempDir()
	s := New(root)
	if err := s.AppendHistory(history.HistoryEntry{Time: time.Now(), Request: collection.Request{URL: "https://example.com"}}); err != nil {
		t.Fatalf("append error: %v", err)
	}
	info, err := os.Stat(filepath.Join(root, "history.jsonl"))
	if err != nil {
		t.Fatalf("stat error: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("history.jsonl perm = %04o, want 0600", perm)
	}
}

// TestHistoryStore_AppendTightensLegacyPerms guards that a history.jsonl left
// world-readable by an older build is re-tightened to 0600 on the next append.
func TestHistoryStore_AppendTightensLegacyPerms(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "history.jsonl"), []byte(""), 0o644); err != nil {
		t.Fatalf("seed write error: %v", err)
	}
	s := New(root)
	if err := s.AppendHistory(history.HistoryEntry{Time: time.Now(), Request: collection.Request{URL: "https://example.com"}}); err != nil {
		t.Fatalf("append error: %v", err)
	}
	info, err := os.Stat(filepath.Join(root, "history.jsonl"))
	if err != nil {
		t.Fatalf("stat error: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("history.jsonl perm = %04o, want 0600", perm)
	}
}

func TestHistoryStore_StoredAsJSONLAtProjectRoot(t *testing.T) {
	root := t.TempDir()
	s := New(root)
	if err := s.AppendHistory(history.HistoryEntry{Time: time.Now(), Request: collection.Request{URL: "https://example.com"}}); err != nil {
		t.Fatalf("append error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "history.jsonl")); err != nil {
		t.Errorf("expected history.jsonl at project root: %v", err)
	}
}

func TestHistoryStore_AppendEvictsOldestBeyondCap(t *testing.T) {
	// shrink the on-disk cap for the test, restore after.
	prev := maxHistoryEntries
	maxHistoryEntries = 3
	t.Cleanup(func() { maxHistoryEntries = prev })

	s := New(t.TempDir())
	for i := 0; i < 6; i++ {
		entry := history.HistoryEntry{
			Time:    time.Date(2026, 1, 1, 0, 0, i, 0, time.UTC),
			Request: collection.Request{Method: collection.GET, URL: "https://example.com"},
			Status:  "200 OK",
			// tag each entry so we can tell which survived; RequestPath round-trips.
			RequestPath: "req-" + string(rune('0'+i)),
		}
		if err := s.AppendHistory(entry); err != nil {
			t.Fatalf("append %d error: %v", i, err)
		}
	}

	entries, err := s.ListHistory()
	if err != nil {
		t.Fatalf("list error: %v", err)
	}
	// only the 3 newest are kept, and ListHistory returns newest-first.
	if len(entries) != 3 {
		t.Fatalf("kept %d entries, want 3 (cap)", len(entries))
	}
	wantNewestFirst := []string{"req-5", "req-4", "req-3"}
	for i, want := range wantNewestFirst {
		if entries[i].RequestPath != want {
			t.Errorf("entries[%d].RequestPath = %q, want %q", i, entries[i].RequestPath, want)
		}
	}

	// the file on disk is bounded too, not just the returned slice.
	data, err := os.ReadFile(filepath.Join(s.ProjectRoot, "history.jsonl"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	lines := 0
	for _, b := range data {
		if b == '\n' {
			lines++
		}
	}
	if lines != 3 {
		t.Errorf("history.jsonl has %d lines, want 3", lines)
	}
}
