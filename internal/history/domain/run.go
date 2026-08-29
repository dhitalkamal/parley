package history

import (
	"time"

	execution "github.com/dhitalkamal/parley/internal/execution/domain"
)

// CollectionRunEntry is one row of the append-only collection-run log: every
// time a folder or whole collection is run - interactively from the TUI's
// "run collection" action, or headlessly via `parley run` - one entry is
// recorded here, so a dashboard can show pass/fail and latency trends across
// runs over time instead of just the most recent one.
type CollectionRunEntry struct {
	Time    time.Time
	Path    string // the folder/request name that was run, "" for the whole collection
	Results []execution.RunResult
	TotalMS int64
}

// RunHistoryStore appends, reads, and deletes rows in the collection-run
// log. Implemented by history/infrastructure.
type RunHistoryStore interface {
	AppendRun(entry CollectionRunEntry) error
	ListRuns() ([]CollectionRunEntry, error)
	// DeleteRun removes the first stored entry matching target (see
	// infrastructure's own equality check - entries have no separate ID, so
	// a whole-entry match is what identifies "this one"). A no-op, not an
	// error, if nothing matches.
	DeleteRun(target CollectionRunEntry) error
}
