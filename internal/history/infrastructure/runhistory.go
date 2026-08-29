package historystore

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"

	history "github.com/dhitalkamal/parley/internal/history/domain"
)

// RunStore appends to and reads runs.jsonl at the project root - an
// append-only, one-JSON-object-per-line log of every collection run
// (interactive or via `parley run`), the same shape as HistoryStore's
// history.jsonl.
type RunStore struct {
	ProjectRoot string
}

var _ history.RunHistoryStore = (*RunStore)(nil)

// NewRunStore returns a RunStore rooted at projectRoot. The log file need
// not exist yet; it's created on first append.
func NewRunStore(projectRoot string) *RunStore {
	return &RunStore{ProjectRoot: projectRoot}
}

func (s *RunStore) path() string {
	return filepath.Join(s.ProjectRoot, "runs.jsonl")
}

// CollectionRunEntry's fields (time.Time, plain execution.RunResult /
// scripting.TestResult structs) already marshal directly - unlike
// HistoryEntry, there's no collection.Request needing fsstore's file
// encoding, so no separate line-DTO type is needed here.

func (s *RunStore) AppendRun(entry history.CollectionRunEntry) error {
	if err := os.MkdirAll(s.ProjectRoot, 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(s.path(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = f.Write(append(data, '\n'))
	return err
}

func (s *RunStore) ListRuns() ([]history.CollectionRunEntry, error) {
	f, err := os.Open(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var entries []history.CollectionRunEntry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry history.CollectionRunEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	return entries, nil
}

// sameRun reports whether a and b are the same recorded run - entries have
// no separate ID field, so a whole-value match (through the same JSON
// marshal/unmarshal round trip on both sides) is what identifies "this one".
// time.Time uses Equal rather than == since a round trip can drop the
// monotonic reading == would otherwise compare.
func sameRun(a, b history.CollectionRunEntry) bool {
	return a.Time.Equal(b.Time) && a.Path == b.Path && a.TotalMS == b.TotalMS && reflect.DeepEqual(a.Results, b.Results)
}

// DeleteRun rewrites runs.jsonl without the first line matching target - a
// no-op, not an error, if the log doesn't exist yet or nothing matches (the
// same best-effort shape AppendRun's callers already expect).
func (s *RunStore) DeleteRun(target history.CollectionRunEntry) error {
	f, err := os.Open(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var kept []string
	deleted := false
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if !deleted {
			var entry history.CollectionRunEntry
			if err := json.Unmarshal([]byte(line), &entry); err == nil && sameRun(entry, target) {
				deleted = true
				continue
			}
		}
		kept = append(kept, line)
	}
	scanErr := scanner.Err()
	f.Close()
	if scanErr != nil {
		return scanErr
	}
	if !deleted {
		return nil
	}

	tmpPath := s.path() + ".tmp"
	out, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	for _, line := range kept {
		if _, err := out.WriteString(line + "\n"); err != nil {
			out.Close()
			return err
		}
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.path())
}
