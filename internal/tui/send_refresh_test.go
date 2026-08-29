package tui

import (
	"context"
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	environment "github.com/dhitalkamal/parley/internal/environment/domain"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// fakeHTTPClient records every request it's given and returns queued
// responses in order - deterministic stand-in for the real httpclient.Client
// so refresh-chain tests never touch the network.
type fakeHTTPClient struct {
	responses []execution.Response
	err       error
	calls     []collection.Request
}

func (f *fakeHTTPClient) Do(ctx context.Context, req collection.Request) (execution.Response, error) {
	f.calls = append(f.calls, req)
	if f.err != nil {
		return execution.Response{}, f.err
	}
	if len(f.responses) == 0 {
		return execution.Response{StatusCode: 200, Status: "200 OK"}, nil
	}
	resp := f.responses[0]
	f.responses = f.responses[1:]
	return resp, nil
}

// runCmd executes a tea.Cmd and, if it's a tea.BatchMsg, drills into it to
// find the first message of the requested type - sendTickCmd is always
// batched alongside the real send, and tests only care about the latter.
func firstMsgOfType[T any](t *testing.T, cmd tea.Cmd) T {
	t.Helper()
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if m, ok := c().(T); ok {
				return m
			}
		}
		t.Fatalf("no message of the requested type in batch")
	}
	m, ok := msg.(T)
	if !ok {
		t.Fatalf("got message of type %T, want a different type", msg)
	}
	return m
}

func TestPrepareAndSendRequest_ProactivelyRefreshesExpiredToken(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	refreshPath, err := m.store.SaveRequest("", "refresh", collection.Request{
		Method:      collection.POST,
		URL:         "https://api.example.com/refresh",
		AuthCapture: collection.AuthCapture{TokenField: "token", TokenVar: "token"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m.activeEnv = environment.Environment{Variables: []environment.Variable{
		{Key: "token", Value: "stale", Enabled: true},
		{Key: "tokenExpiresAt", Value: "1", Enabled: true}, // long since expired
	}}
	m.client = &fakeHTTPClient{responses: []execution.Response{
		{StatusCode: 200, Status: "200 OK", Body: []byte(`{"token":"fresh"}`)},
		{StatusCode: 200, Status: "200 OK", Body: []byte(`{"ok":true}`)},
	}}

	rawReq := collection.Request{
		Method:  collection.GET,
		URL:     "https://api.example.com/protected",
		Refresh: collection.RefreshConfig{RequestPath: refreshPath, ExpiresAtVar: "tokenExpiresAt"},
	}
	m, cmd := m.prepareAndSendRequest(rawReq)
	if cmd == nil {
		t.Fatal("expected a non-nil command to trigger the refresh")
	}
	msg := firstMsgOfType[refreshResultMsg](t, cmd)
	if msg.refreshReq.URL != "https://api.example.com/refresh" {
		t.Errorf("refreshReq.URL = %q", msg.refreshReq.URL)
	}
	if msg.originalRawReq.URL != rawReq.URL {
		t.Errorf("originalRawReq.URL = %q, want %q", msg.originalRawReq.URL, rawReq.URL)
	}
}

func TestPrepareAndSendRequest_SendsNormallyWhenTokenNotExpired(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.activeEnv = environment.Environment{Variables: []environment.Variable{
		{Key: "tokenExpiresAt", Value: "9999999999", Enabled: true}, // far future
	}}
	m.client = &fakeHTTPClient{}

	rawReq := collection.Request{
		Method:  collection.GET,
		URL:     "https://api.example.com/protected",
		Refresh: collection.RefreshConfig{RequestPath: "refresh.json", ExpiresAtVar: "tokenExpiresAt"},
	}
	m, cmd := m.prepareAndSendRequest(rawReq)
	if cmd == nil {
		t.Fatal("expected a non-nil send command")
	}
	msg := firstMsgOfType[sendResultMsg](t, cmd)
	if msg.req.URL != rawReq.URL {
		t.Errorf("sent URL = %q, want %q", msg.req.URL, rawReq.URL)
	}
}

func TestHandleSendResult_TriggersReactiveRefreshOn401(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	refreshPath, err := m.store.SaveRequest("", "refresh", collection.Request{
		Method:      collection.POST,
		URL:         "https://api.example.com/refresh",
		AuthCapture: collection.AuthCapture{TokenField: "token", TokenVar: "token"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m.client = &fakeHTTPClient{responses: []execution.Response{
		{StatusCode: 200, Status: "200 OK", Body: []byte(`{"token":"fresh"}`)},
	}}

	rawReq := collection.Request{
		URL:     "https://api.example.com/protected",
		Refresh: collection.RefreshConfig{RequestPath: refreshPath, ExpiresAtVar: "tokenExpiresAt"},
	}
	msg := sendResultMsg{
		rawReq: rawReq,
		req:    rawReq,
		resp:   execution.Response{StatusCode: 401, Status: "401 Unauthorized"},
	}
	m, cmd := m.handleSendResult(msg)
	if cmd == nil {
		t.Fatal("expected a non-nil command to trigger the refresh")
	}
	got := firstMsgOfType[refreshResultMsg](t, cmd)
	if got.originalRawReq.URL != rawReq.URL {
		t.Errorf("originalRawReq.URL = %q", got.originalRawReq.URL)
	}
}

func TestHandleSendResult_DoesNotRefreshAgainOnARetriedRequest(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	rawReq := collection.Request{
		URL:     "https://api.example.com/protected",
		Refresh: collection.RefreshConfig{RequestPath: "refresh.json", ExpiresAtVar: "tokenExpiresAt"},
	}
	msg := sendResultMsg{
		rawReq:  rawReq,
		req:     rawReq,
		resp:    execution.Response{StatusCode: 401, Status: "401 Unauthorized"},
		retried: true,
	}
	_, cmd := m.handleSendResult(msg)
	if cmd != nil {
		t.Error("expected no further refresh once a request has already been retried once")
	}
}

func TestHandleRefreshResult_AppliesCaptureThenRetriesOriginal(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	m.client = &fakeHTTPClient{responses: []execution.Response{
		{StatusCode: 200, Status: "200 OK", Body: []byte(`{"ok":true}`)},
	}}

	refreshReq := collection.Request{
		URL:         "https://api.example.com/refresh",
		AuthCapture: collection.AuthCapture{TokenField: "token", TokenVar: "token"},
	}
	originalRawReq := collection.Request{URL: "https://api.example.com/protected/{{token}}"}
	msg := refreshResultMsg{
		refreshReq:     refreshReq,
		originalRawReq: originalRawReq,
		resp:           execution.Response{StatusCode: 200, Status: "200 OK", Body: []byte(`{"token":"fresh"}`)},
	}

	m, cmd := m.handleRefreshResult(msg)
	if got := m.resolvedVars()["token"].Value; got != "fresh" {
		t.Errorf("token variable = %q, want fresh", got)
	}
	if cmd == nil {
		t.Fatal("expected a non-nil command to retry the original request")
	}
	got := firstMsgOfType[sendResultMsg](t, cmd)
	if got.req.URL != "https://api.example.com/protected/fresh" {
		t.Errorf("retried URL = %q, want the token substituted in", got.req.URL)
	}
	if !got.retried {
		t.Error("expected the retried send to be marked retried, so a second 401 doesn't refresh again")
	}
}

func TestHandleRefreshResult_FailsOpenWhenRefreshItselfErrors(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	msg := refreshResultMsg{
		refreshReq:     collection.Request{URL: "https://api.example.com/refresh"},
		originalRawReq: collection.Request{URL: "https://api.example.com/protected"},
		err:            context.DeadlineExceeded,
	}
	_, cmd := m.handleRefreshResult(msg)
	if cmd != nil {
		t.Error("expected no retry when the refresh call itself failed")
	}
}

func TestHandleRefreshResult_FailsOpenWhenRefreshReturnsError(t *testing.T) {
	m := New(t.TempDir(), t.TempDir())
	msg := refreshResultMsg{
		refreshReq:     collection.Request{URL: "https://api.example.com/refresh"},
		originalRawReq: collection.Request{URL: "https://api.example.com/protected"},
		resp:           execution.Response{StatusCode: 500, Status: "500 Internal Server Error"},
	}
	_, cmd := m.handleRefreshResult(msg)
	if cmd != nil {
		t.Error("expected no retry when the refresh endpoint itself failed")
	}
}
