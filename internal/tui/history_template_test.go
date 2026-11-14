package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	environment "github.com/dhitalkamal/parley/internal/environment/domain"
	history "github.com/dhitalkamal/parley/internal/history/domain"
	historystore "github.com/dhitalkamal/parley/internal/history/infrastructure"

	tea "github.com/charmbracelet/bubbletea"
)

func tokenModel(t *testing.T, token string) (Model, string) {
	t.Helper()
	root := t.TempDir()
	m := New(root, t.TempDir())
	m.historyStore = historystore.New(root)
	m.activeEnv = environment.Environment{Variables: []environment.Variable{
		{Key: "token", Value: token, Enabled: true},
	}}
	return m, filepath.Join(root, "history.jsonl")
}

func authReq() collection.Request {
	return collection.Request{
		Method:  collection.GET,
		URL:     "https://api.example.com/me",
		Headers: []collection.Header{{Key: "Authorization", Value: "Bearer {{token}}", Enabled: true}},
	}
}

func TestSend_HistoryStoresTemplateNotSecret(t *testing.T) {
	cases := map[string]error{"success": nil, "error": os.ErrDeadlineExceeded}
	for name, sendErr := range cases {
		t.Run(name, func(t *testing.T) {
			m, file := tokenModel(t, "s3cret")
			fc := &fakeHTTPClient{err: sendErr}
			m.client = fc
			m, cmd := m.prepareAndSendRequest(authReq())
			msg := firstMsgOfType[sendResultMsg](t, cmd)
			m, _ = m.handleSendResult(msg)

			if got := fc.calls[0].Headers[0].Value; got != "Bearer s3cret" {
				t.Fatalf("sent header = %q, want the resolved secret", got)
			}
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("read history file: %v", err)
			}
			if strings.Contains(string(raw), "s3cret") {
				t.Errorf("history file leaks the secret: %s", raw)
			}
			entries, _ := m.historyStore.ListHistory()
			if len(entries) != 1 || entries[0].Request.Headers[0].Value != "Bearer {{token}}" {
				t.Errorf("stored entry = %+v, want the {{token}} template", entries)
			}
		})
	}
}

func rerun(t *testing.T, m Model, e history.HistoryEntry) (Model, sendResultMsg) {
	t.Helper()
	m.history = historyState{active: true, entries: []history.HistoryEntry{e}}
	m, cmd := m.handleHistoryKey(tea.KeyMsg{Type: tea.KeyEnter})
	return m, firstMsgOfType[sendResultMsg](t, cmd)
}

func TestHistoryRerun_UsesCurrentVariables(t *testing.T) {
	m, _ := tokenModel(t, "new-token")
	fc := &fakeHTTPClient{}
	m.client = fc

	_, msg := rerun(t, m, history.HistoryEntry{Request: authReq()})

	if got := fc.calls[0].Headers[0].Value; got != "Bearer new-token" {
		t.Errorf("sent header = %q, want current token", got)
	}
	if got := msg.rawReq.Headers[0].Value; got != "Bearer {{token}}" {
		t.Errorf("rawReq header = %q, want the template kept for reactive refresh", got)
	}
}

func TestHistoryRerun_OldResolvedEntryUnchanged(t *testing.T) {
	m, _ := tokenModel(t, "new-token")
	fc := &fakeHTTPClient{}
	m.client = fc
	old := authReq()
	old.Headers[0].Value = "Bearer old-resolved"

	rerun(t, m, history.HistoryEntry{Request: old})

	if got := fc.calls[0].Headers[0].Value; got != "Bearer old-resolved" {
		t.Errorf("sent header = %q, want the stored value untouched", got)
	}
}
