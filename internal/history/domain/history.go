package history

import (
	"time"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
)

// HistoryEntry is one row of the append-only send history: the exact
// (already variable-resolved) request that went out, and what came back.
// Err is set instead of StatusCode/Status when the send itself failed.
type HistoryEntry struct {
	Time time.Time
	// RequestPath is the saved collection request this send came from, in
	// the same store-path form workspace.Layout.SelectedPath and
	// sidebarItem.path use - empty for a send with nothing saved yet (an
	// ad hoc request never written to disk), or for entries recorded
	// before this field existed. Backs the Collections screen's per-request
	// "Recent History" panel, which has no other reliable way to tell which
	// saved request a given send belonged to (URL/Method alone can collide,
	// or drift after variable substitution).
	RequestPath string
	Request     collection.Request
	StatusCode  int
	Status      string
	ElapsedMS   int64
	Err         string
}
