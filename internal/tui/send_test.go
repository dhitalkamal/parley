package tui

import (
	"errors"
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	history "github.com/dhitalkamal/parley/internal/history/domain"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TestTrySend_MarksSending guards the fix for a real complaint: sending a
// request gave no feedback at all that anything was happening until the
// result eventually came back. trySend must mark the request as in flight
// so the Response panel can show a live indicator (see resppanel.go).
func TestTrySend_MarksSending(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.urlInput.SetValue("https://example.com")

	got, cmd := m.trySend()

	if !got.sending {
		t.Error("expected trySend to mark the request as sending")
	}
	if got.sendStartedAt.IsZero() {
		t.Error("expected trySend to record when the send started")
	}
	if cmd == nil {
		t.Error("expected trySend to return a non-nil command (send + tick)")
	}
}

// TestTrySend_EmptyURLDoesNotMarkSending guards against the empty-URL
// short-circuit accidentally showing a "Sending..." indicator for a send
// that never actually happened.
func TestTrySend_EmptyURLDoesNotMarkSending(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())

	got, _ := m.trySend()

	if got.sending {
		t.Error("expected an empty URL to never mark the request as sending")
	}
}

func TestHandleSendResult_ClearsSendingOnSuccess(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.sending = true

	got, _ := m.handleSendResult(sendResultMsg{resp: execution.Response{StatusCode: 200, Status: "200 OK"}})

	if got.sending {
		t.Error("expected a successful result to clear sending")
	}
}

func TestHandleSendResult_ClearsSendingOnError(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.sending = true

	got, _ := m.handleSendResult(sendResultMsg{err: errors.New("boom")})

	if got.sending {
		t.Error("expected a failed result to clear sending too")
	}
}

// TestResponsePanelView_ShowsSendingIndicatorWhilePending guards the
// visible half of the fix - the indicator has to actually render somewhere,
// not just flip an internal flag.
func TestResponsePanelView_ShowsSendingIndicatorWhilePending(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.sending = true
	m.sendStartedAt = time.Now()

	got := stripANSI(m.responsePanelView(60, 20))

	if !strings.Contains(got, "Sending") {
		t.Errorf("got %q, want a \"Sending\" indicator while a request is in flight", got)
	}
}

// TestResponsePanelView_HidesSendingIndicatorWhenNotPending is the
// regression guard: outside of an in-flight send, the status line must
// still show the response/Ready as before.
func TestResponsePanelView_HidesSendingIndicatorWhenNotPending(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())

	got := stripANSI(m.responsePanelView(60, 20))

	if strings.Contains(got, "Sending") {
		t.Errorf("got %q, want no \"Sending\" indicator when nothing is in flight", got)
	}
}

// TestHandleHistoryKey_ReRunAlsoMarksSending guards the other place a send
// can be triggered from (re-running a past request) - it must give the
// same in-flight feedback a fresh send does, not silently skip it.
func TestHandleHistoryKey_ReRunAlsoMarksSending(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.history.entries = []history.HistoryEntry{{Request: collection.Request{Method: collection.GET, URL: "https://example.com"}}}

	got, cmd := m.handleHistoryKey(tea.KeyMsg{Type: tea.KeyEnter})

	if !got.sending {
		t.Error("expected re-running a history entry to mark the request as sending")
	}
	if cmd == nil {
		t.Error("expected a non-nil command (send + tick)")
	}
}
