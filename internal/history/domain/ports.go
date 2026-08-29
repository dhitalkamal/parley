package history

// HistoryStore appends and reads the send history log. Implemented by
// history/infrastructure.
type HistoryStore interface {
	AppendHistory(entry HistoryEntry) error
	ListHistory() ([]HistoryEntry, error)
}
