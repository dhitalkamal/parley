package historystore

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	history "github.com/dhitalkamal/parley/internal/history/domain"
	"github.com/dhitalkamal/parley/internal/platform/fsstore"
)

// HistoryStore appends to and reads history.jsonl at the project root - an
// append-only, one-JSON-object-per-line log of every send, per the storage
// layout in the project brief.
type HistoryStore struct {
	ProjectRoot string
}

var _ history.HistoryStore = (*HistoryStore)(nil)

// New returns a HistoryStore rooted at projectRoot. The log
// file need not exist yet; it's created on first append.
func New(projectRoot string) *HistoryStore {
	return &HistoryStore{ProjectRoot: projectRoot}
}

func (s *HistoryStore) path() string {
	return filepath.Join(s.ProjectRoot, "history.jsonl")
}

type historyLine struct {
	Time        time.Time           `json:"time"`
	RequestPath string              `json:"requestPath,omitempty"`
	Request     fsstore.RequestFile `json:"request"`
	StatusCode  int                 `json:"statusCode,omitempty"`
	Status      string              `json:"status,omitempty"`
	ElapsedMS   int64               `json:"elapsedMs,omitempty"`
	Err         string              `json:"err,omitempty"`
}

// maxHistoryEntries bounds how many sent-request records history.jsonl keeps
// on disk. Older entries are evicted on append so a long-lived install cannot
// grow the file (or ListHistory's whole-file parse) without bound. The history
// modal shows fewer than this, so the on-disk cap sits comfortably above what
// is ever displayed. A var, not a const, so tests can shrink it. Appends
// happen per-send (not a hot path), so trimming on each append is cheap enough.
var maxHistoryEntries = 500

func (s *HistoryStore) AppendHistory(entry history.HistoryEntry) error {
	if err := s.appendLine(entry); err != nil {
		return err
	}
	return s.trimToCap(maxHistoryEntries)
}

func (s *HistoryStore) appendLine(entry history.HistoryEntry) error {
	if err := os.MkdirAll(s.ProjectRoot, 0o755); err != nil {
		return err
	}
	// history.jsonl records full requests verbatim (headers like Authorization,
	// secret-valued params/body), so it must not be world- or group-readable.
	// 0o600 on create; explicit Chmod tightens any file left at looser perms by
	// an older build (O_CREATE only sets the mode when the file is new).
	f, err := os.OpenFile(s.path(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.Chmod(0o600); err != nil {
		return err
	}

	line := historyLine{
		Time:        entry.Time,
		RequestPath: entry.RequestPath,
		Request:     fsstore.RequestToFile(entry.Request),
		StatusCode:  entry.StatusCode,
		Status:      entry.Status,
		ElapsedMS:   entry.ElapsedMS,
		Err:         entry.Err,
	}
	data, err := json.Marshal(line)
	if err != nil {
		return err
	}
	_, err = f.Write(append(data, '\n'))
	return err
}

// trimToCap rewrites history.jsonl to keep only its last cap lines (the newest
// entries, since the file is append-order oldest-first). It reads the whole
// file, and when over the cap writes the kept lines to a temp file and renames
// it into place - an atomic replace, so a crash mid-trim cannot corrupt or
// truncate the log. A no-op when the file is within the cap.
func (s *HistoryStore) trimToCap(cap int) error {
	data, err := os.ReadFile(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	lines := bytes.Split(bytes.TrimRight(data, "\n"), []byte{'\n'})
	if len(lines) <= cap {
		return nil
	}
	kept := lines[len(lines)-cap:]
	out := append(bytes.Join(kept, []byte{'\n'}), '\n')

	tmp, err := os.CreateTemp(s.ProjectRoot, "history-*.jsonl.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if _, err := tmp.Write(out); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, s.path())
}

func (s *HistoryStore) ListHistory() ([]history.HistoryEntry, error) {
	f, err := os.Open(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var entries []history.HistoryEntry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var hl historyLine
		if err := json.Unmarshal(line, &hl); err != nil {
			continue
		}
		entries = append(entries, history.HistoryEntry{
			Time:        hl.Time,
			RequestPath: hl.RequestPath,
			Request:     fsstore.RequestFromFile(hl.Request),
			StatusCode:  hl.StatusCode,
			Status:      hl.Status,
			ElapsedMS:   hl.ElapsedMS,
			Err:         hl.Err,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	return entries, nil
}
