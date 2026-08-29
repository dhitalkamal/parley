package historystore

import (
	"bufio"
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

func (s *HistoryStore) AppendHistory(entry history.HistoryEntry) error {
	if err := os.MkdirAll(s.ProjectRoot, 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(s.path(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

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
