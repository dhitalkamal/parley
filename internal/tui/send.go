package tui

import (
	"context"
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	environment "github.com/dhitalkamal/parley/internal/environment/domain"
	execapp "github.com/dhitalkamal/parley/internal/execution/application"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	history "github.com/dhitalkamal/parley/internal/history/domain"
	scripting "github.com/dhitalkamal/parley/internal/scripting/domain"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type sendResultMsg struct {
	rawReq  collection.Request // pre-substitution - needed to re-resolve {{vars}} after a refresh changes them
	req     collection.Request // as actually sent (post-substitution)
	resp    execution.Response
	err     error
	retried bool // true once this send is itself a post-refresh retry - caps reactive refresh at one attempt (see handleSendResult)
}

// sendTickMsg carries no data of its own - it exists purely to force a
// repaint every 200ms while a request is in flight, so the Response
// panel's "Sending... 123 ms" elapsed counter (see resppanel.go) actually
// ticks instead of freezing at whatever it showed the moment the send
// started.
type sendTickMsg time.Time

func sendTickCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg { return sendTickMsg(t) })
}

// beginSending marks a request as in flight - shared by a fresh send
// (prepareAndSend) and a history quick re-run (history.go), so neither has
// to duplicate the bookkeeping the "Sending..." indicator depends on.
func (m *Model) beginSending() {
	m.sending = true
	m.sendStartedAt = time.Now()
}

func (m Model) buildRequest() collection.Request {
	params := make([]collection.QueryParam, len(m.params.Rows()))
	for i, r := range m.params.Rows() {
		params[i] = collection.QueryParam{Key: r.Key, Value: r.Value, Enabled: r.Enabled, Description: r.Description}
	}
	headers := make([]collection.Header, len(m.headers.Rows()))
	for i, r := range m.headers.Rows() {
		headers[i] = collection.Header{Key: r.Key, Value: r.Value, Enabled: r.Enabled, Description: r.Description}
	}
	return collection.Request{
		Method:             collection.Methods[m.methodIdx],
		URL:                strings.TrimSpace(m.urlInput.Value()),
		Params:             params,
		Headers:            headers,
		Body:               m.body.Body(),
		Timeout:            m.reqSettings.Timeout(),
		FollowRedirects:    m.reqSettings.FollowRedirects(),
		InsecureSkipVerify: m.reqSettings.InsecureSkipVerify(),
		PreRequestScript:   m.scripts.PreRequest(),
		TestScript:         m.scripts.Test(),
		AuthCapture:        m.auth.AuthCapture(),
		Refresh:            m.auth.RefreshConfig(),
	}
}

// trySend guards prepareAndSend with the same empty-URL check the direct
// keybinding and the command palette both need, so neither has to duplicate it.
func (m Model) trySend() (Model, tea.Cmd) {
	if strings.TrimSpace(m.urlInput.Value()) == "" {
		m.status = "Enter a URL first"
		return m, nil
	}
	return m.prepareAndSend()
}

// prepareAndSend builds the request from the editor and dispatches it - see
// prepareAndSendRequest for the actual logic, split out so tests (and,
// later, any other caller with an already-built request) can drive it
// directly without going through the editor widgets.
func (m Model) prepareAndSend() (Model, tea.Cmd) {
	return m.prepareAndSendRequest(m.buildRequest())
}

// prepareAndSendRequest runs the pre-request script (if any) against the
// raw, unsubstituted request first - matching Postman's real order, so a
// script can set a variable that the {{...}} substitution below then picks
// up - then, if the request depends on a token tracked as expired,
// refreshes it proactively before ever substituting/sending (see
// send_refresh.go) rather than sending a request already known to fail.
// Otherwise resolves variables and dispatches the send as usual.
func (m Model) prepareAndSendRequest(rawReq collection.Request) (Model, tea.Cmd) {
	if strings.TrimSpace(rawReq.PreRequestScript) != "" {
		newReq, newCtx, err := m.scriptRunner.RunPreRequest(rawReq.PreRequestScript, rawReq, m.scriptContext())
		if err != nil {
			m.status = "Pre-request script error: " + err.Error()
			return m, nil
		}
		rawReq = newReq
		m.applyScriptContext(newCtx)
	}
	if rawReq.Refresh.Enabled() && execution.IsExpired(m.resolvedVars()[rawReq.Refresh.ExpiresAtVar].Value, time.Now()) {
		return m.triggerRefresh(rawReq)
	}
	req, _ := execution.SubstituteRequest(rawReq, m.resolvedVars())
	m.status = "Sending..."
	m.response.SetTestResults(nil, "")
	m.beginSending()
	return m, tea.Batch(sendReqCmd(m.client, rawReq, req, false), sendTickCmd())
}

// scriptContext snapshots the currently resolved environment/globals values
// for scripts to read and mutate via pm.environment/pm.globals.
func (m Model) scriptContext() scripting.ScriptContext {
	env := make(map[string]string)
	for _, v := range m.activeEnv.Variables {
		if v.Enabled {
			env[v.Key] = v.Value
		}
	}
	globals := make(map[string]string)
	for _, v := range m.globals.Variables {
		if v.Enabled {
			globals[v.Key] = v.Value
		}
	}
	return scripting.ScriptContext{Environment: env, Globals: globals}
}

// applyScriptContext merges a script's variable mutations back into the
// active environment and globals, persisting both so chained requests (and
// future runs) see the update - this is what makes
// pm.environment.set("token", pm.response.json().token) actually work.
func (m *Model) applyScriptContext(ctx scripting.ScriptContext) {
	m.activeEnv.Variables = environment.ApplyVariableUpdates(m.activeEnv.Variables, ctx.Environment)
	m.globals.Variables = environment.ApplyVariableUpdates(m.globals.Variables, ctx.Globals)
	if m.activeEnvName != "" {
		// Save the whole record (not just variables) so a script's variable
		// write doesn't drop the env's ExpectedVPN.
		m.activeEnv.Name = m.activeEnvName
		_ = m.envStore.SaveEnvironment(m.activeEnv)
	}
	_ = m.envStore.SaveGlobals(m.globals)
}

// sendReqCmd executes an already-resolved request. Shared by a normal send
// (built and substituted from the editor), a history quick re-run (already
// resolved, replayed exactly as it went out originally), and a post-refresh
// retry (send_refresh.go). rawReq is carried through purely so a 401 on
// this send can re-substitute and retry once more after another refresh -
// callers with no meaningful "raw" form (history re-runs) just pass the
// same request for both.
func sendReqCmd(client execution.HTTPClient, rawReq, req collection.Request, retried bool) tea.Cmd {
	return func() tea.Msg {
		resp, err := execapp.SendRequest(context.Background(), client, req)
		return sendResultMsg{rawReq: rawReq, req: req, resp: resp, err: err, retried: retried}
	}
}

// handleSendResult updates the response viewer, runs the test script (if
// any), captures an auth token if configured, and appends a history entry -
// for both a fresh send and a history re-run. A 401 on a request that
// hasn't already been retried once triggers a reactive refresh-then-retry
// (see send_refresh.go) instead of just showing the failure.
func (m Model) handleSendResult(msg sendResultMsg) (Model, tea.Cmd) {
	m.sending = false
	entry := history.HistoryEntry{Time: time.Now(), RequestPath: m.loadedRequestPath, Request: msg.req}
	if msg.err != nil {
		entry.Err = msg.err.Error()
		m.status = "Error"
		m.response.SetError(msg.err, networkErrorHint(msg.err, m.detectedVPN, m.activeEnv.ExpectedVPN))
		_ = m.historyStore.AppendHistory(entry)
		return m, nil
	}
	entry.StatusCode = msg.resp.StatusCode
	entry.Status = msg.resp.Status
	entry.ElapsedMS = msg.resp.Elapsed.Milliseconds()
	m.status = "Done"
	m.response.SetResponse(msg.resp, entry.ElapsedMS)
	m = m.runTestScript(msg.req, msg.resp)
	m = m.applyAuthCapture(msg.req, msg.resp)
	// Persisted (not just cached in memory - see responseCache) so this
	// request still shows what it last returned even after the app
	// restarts. Only on success: an old network/DNS error isn't worth
	// carrying across a restart the way an actual response is.
	if m.loadedRequestPath != "" {
		_ = m.lastResponseStore.Save(m.loadedRequestPath, msg.resp, entry.ElapsedMS)
	}
	// History logging is best-effort: a disk error here shouldn't hide the
	// response that was already shown above.
	_ = m.historyStore.AppendHistory(entry)

	if msg.resp.StatusCode == 401 && !msg.retried && msg.rawReq.Refresh.Enabled() {
		return m.triggerRefresh(msg.rawReq)
	}
	return m, nil
}

func (m Model) runTestScript(req collection.Request, resp execution.Response) Model {
	if strings.TrimSpace(req.TestScript) == "" {
		return m
	}
	results, newCtx, err := m.scriptRunner.RunTest(req.TestScript, req, resp, m.scriptContext())
	if err != nil {
		m.response.SetTestResults(nil, err.Error())
		return m
	}
	m.response.SetTestResults(results, "")
	m.applyScriptContext(newCtx)
	return m
}
