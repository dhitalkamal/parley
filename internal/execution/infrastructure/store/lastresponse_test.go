package executionstore

import (
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	"testing"
)

func TestLastResponseStore_LoadOnMissingEntryReturnsNotOk(t *testing.T) {
	s := New(t.TempDir())

	_, _, ok, err := s.Load("some/path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("got ok=true, want false for a path that was never saved")
	}
}

func TestLastResponseStore_SaveThenLoadRoundTrips(t *testing.T) {
	s := New(t.TempDir())
	resp := execution.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Headers:    []collection.Header{{Key: "Content-Type", Value: "application/json", Enabled: true}},
		Body:       []byte(`{"ok":true}`),
	}

	if err := s.Save("a.json", resp, 42); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, elapsedMS, ok, err := s.Load("a.json")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !ok {
		t.Fatal("got ok=false, want true")
	}
	if got.StatusCode != 200 || got.Status != "200 OK" || string(got.Body) != `{"ok":true}` {
		t.Errorf("got %+v", got)
	}
	if len(got.Headers) != 1 || got.Headers[0].Key != "Content-Type" {
		t.Errorf("got headers %+v", got.Headers)
	}
	if elapsedMS != 42 {
		t.Errorf("elapsedMS = %d, want 42", elapsedMS)
	}
}

// TestLastResponseStore_SavePersistsAcrossInstances guards the actual ask:
// a response saved by one Model must still be there for a Model built
// later (a restart, not just the current process's memory) - not just
// round-trip within the same store value.
func TestLastResponseStore_SavePersistsAcrossInstances(t *testing.T) {
	root := t.TempDir()
	first := New(root)
	if err := first.Save("a.json", execution.Response{StatusCode: 201, Status: "201 Created"}, 10); err != nil {
		t.Fatalf("Save: %v", err)
	}

	second := New(root)
	got, _, ok, err := second.Load("a.json")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !ok || got.StatusCode != 201 {
		t.Errorf("got (%+v, %v), want the response saved by a different store instance", got, ok)
	}
}

func TestLastResponseStore_SaveOverwritesThePreviousResponseForTheSamePath(t *testing.T) {
	s := New(t.TempDir())
	if err := s.Save("a.json", execution.Response{StatusCode: 500}, 10); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := s.Save("a.json", execution.Response{StatusCode: 200}, 20); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, elapsedMS, ok, err := s.Load("a.json")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !ok || got.StatusCode != 200 || elapsedMS != 20 {
		t.Errorf("got (%+v, %d, %v), want the latest save (200, 20, true)", got, elapsedMS, ok)
	}
}

func TestLastResponseStore_KeepsSeparateResponsesPerPath(t *testing.T) {
	s := New(t.TempDir())
	if err := s.Save("a.json", execution.Response{StatusCode: 200}, 10); err != nil {
		t.Fatalf("Save a: %v", err)
	}
	if err := s.Save("b.json", execution.Response{StatusCode: 404}, 20); err != nil {
		t.Fatalf("Save b: %v", err)
	}

	gotA, _, _, _ := s.Load("a.json")
	gotB, _, _, _ := s.Load("b.json")
	if gotA.StatusCode != 200 {
		t.Errorf("a's status = %d, want 200", gotA.StatusCode)
	}
	if gotB.StatusCode != 404 {
		t.Errorf("b's status = %d, want 404", gotB.StatusCode)
	}
}

func TestLastResponseStore_DeleteRemovesTheEntry(t *testing.T) {
	s := New(t.TempDir())
	if err := s.Save("a.json", execution.Response{StatusCode: 200}, 10); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := s.Delete("a.json"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, _, ok, err := s.Load("a.json")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if ok {
		t.Error("got ok=true after Delete, want false")
	}
}

// TestLastResponseStore_DeleteOnMissingEntryIsANoOp guards a delete of a
// request that was never sent (or already deleted) - it must not error just
// because there was nothing to remove.
func TestLastResponseStore_DeleteOnMissingEntryIsANoOp(t *testing.T) {
	s := New(t.TempDir())
	if err := s.Delete("never-saved.json"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
