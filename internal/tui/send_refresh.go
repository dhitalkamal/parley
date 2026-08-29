package tui

import (
	"context"
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	execapp "github.com/dhitalkamal/parley/internal/execution/application"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	scripting "github.com/dhitalkamal/parley/internal/scripting/domain"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// refreshResultMsg carries the outcome of running the saved "refresh"
// request (collection.RefreshConfig.RequestPath) that a protected request
// pointed at - originalRawReq is the protected request's own
// pre-substitution form, kept around so it can be re-substituted with the
// just-refreshed token and retried (see handleRefreshResult).
type refreshResultMsg struct {
	refreshReq     collection.Request
	originalRawReq collection.Request
	resp           execution.Response
	err            error
}

func sendRefreshCmd(client execution.HTTPClient, refreshReq, originalRawReq collection.Request) tea.Cmd {
	return func() tea.Msg {
		resp, err := execapp.SendRequest(context.Background(), client, refreshReq)
		return refreshResultMsg{refreshReq: refreshReq, originalRawReq: originalRawReq, resp: resp, err: err}
	}
}

// triggerRefresh loads and sends the saved request rawReq.Refresh points
// at, carrying rawReq along so handleRefreshResult can retry it once the
// refresh completes. Used both proactively (prepareAndSendRequest, before
// ever substituting/sending the protected request) and reactively
// (handleSendResult, after a 401). A missing or unloadable refresh request
// fails open - falls through to sending rawReq normally rather than
// blocking the send entirely over a broken refresh pointer.
func (m Model) triggerRefresh(rawReq collection.Request) (Model, tea.Cmd) {
	refreshReq, err := m.store.LoadRequest(rawReq.Refresh.RequestPath)
	if err != nil {
		// A broken refresh pointer shouldn't be retried on every single
		// 401 that follows - retried=true here for the same reason it's
		// true after an actual refresh attempt: this is the one retry
		// this request gets.
		req, _ := execution.SubstituteRequest(rawReq, m.resolvedVars())
		m.beginSending()
		return m, tea.Batch(sendReqCmd(m.client, rawReq, req, true), sendTickCmd())
	}
	resolvedRefreshReq, _ := execution.SubstituteRequest(refreshReq, m.resolvedVars())
	m.status = "Refreshing token..."
	m.beginSending()
	return m, tea.Batch(sendRefreshCmd(m.client, resolvedRefreshReq, rawReq), sendTickCmd())
}

// handleRefreshResult applies the refresh response's own AuthCapture (the
// refresh request is expected to return a fresh token the same way a login
// request does) and retries the original request exactly once, now that
// the token it depends on has changed. Fails open on any refresh failure -
// leaving whatever the original request's send already showed (its error,
// or the 401) rather than looping or blocking on a broken refresh.
func (m Model) handleRefreshResult(msg refreshResultMsg) (Model, tea.Cmd) {
	m.sending = false
	if msg.err != nil || msg.resp.StatusCode >= 400 {
		m.status = "Token refresh failed"
		return m, nil
	}
	m = m.applyAuthCapture(msg.refreshReq, msg.resp)
	req, _ := execution.SubstituteRequest(msg.originalRawReq, m.resolvedVars())
	m.status = "Sending..."
	m.beginSending()
	return m, tea.Batch(sendReqCmd(m.client, msg.originalRawReq, req, true), sendTickCmd())
}

// applyAuthCapture extracts req.AuthCapture's configured fields out of
// resp's JSON body and saves them as environment variables - the
// structured equivalent of a hand-written
// pm.environment.set("token", pm.response.json().token) test script (see
// execution.ExtractJSONField), so a login (or refresh) request can make its
// token available to every other request without any scripting.
// applyScriptContext already persists to disk and updates m.activeEnv/
// m.globals in place - reused here rather than duplicated.
func (m Model) applyAuthCapture(req collection.Request, resp execution.Response) Model {
	if !req.AuthCapture.Enabled() {
		return m
	}
	token, ok := execution.ExtractJSONField(resp.Body, req.AuthCapture.TokenField)
	if !ok {
		return m
	}
	updates := map[string]string{req.AuthCapture.TokenVar: token}
	if req.AuthCapture.TracksExpiry() {
		if seconds, ok := execution.ExtractJSONField(resp.Body, req.AuthCapture.ExpiresInField); ok {
			if expiresAt, ok := execution.ComputeExpiresAt(time.Now(), seconds); ok {
				updates[req.AuthCapture.ExpiresAtVar] = expiresAt
			}
		}
	}
	m.applyScriptContext(scripting.ScriptContext{Environment: updates})
	return m
}
