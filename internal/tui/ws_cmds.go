package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
)

// wsDialTimeout bounds only the opening handshake, not the connection's lifetime.
const wsDialTimeout = 15 * time.Second

// wsIncoming is one item the read pump pushes toward the Bubbletea loop: a
// received message, or the error/closure that ends the stream.
type wsIncoming struct {
	msg execution.WSMessage
	err error
}

// Every WS message carries pane so Update can route it to the pane that owns
// the connection - the two panes run fully independent connections.

type wsConnectedMsg struct {
	pane     int
	conn     execution.WSConn
	incoming chan wsIncoming
	ctx      context.Context
	cancel   context.CancelFunc
}

type wsConnectErrMsg struct {
	pane int
	err  error
}
type wsIncomingMsg struct {
	pane int
	msg  execution.WSMessage
}
type wsRecvErrMsg struct {
	pane int
	err  error
}
type wsClosedMsg struct{ pane int }
type wsSendErrMsg struct {
	pane int
	err  error
}
type wsPingResultMsg struct {
	pane int
	dur  time.Duration
	err  error
}

// wsConnectCmd dials in the background and, on success, starts the read pump
// before handing the connection back tagged with its pane.
func wsConnectCmd(pane int, dialer execution.WSDialer, url string, headers []collection.Header) tea.Cmd {
	return func() tea.Msg {
		pumpCtx, cancel := context.WithCancel(context.Background())
		dialCtx, dialCancel := context.WithTimeout(pumpCtx, wsDialTimeout)
		defer dialCancel()

		conn, err := dialer.Dial(dialCtx, url, headers)
		if err != nil {
			cancel()
			return wsConnectErrMsg{pane: pane, err: err}
		}
		ch := make(chan wsIncoming, 16)
		go wsReadPump(pumpCtx, conn, ch)
		return wsConnectedMsg{pane: pane, conn: conn, incoming: ch, ctx: pumpCtx, cancel: cancel}
	}
}

// wsReadPump loops reads off the connection into ch until it errors or the
// context is cancelled, then closes ch so wsWaitCmd resolves to wsClosedMsg.
func wsReadPump(ctx context.Context, conn execution.WSConn, ch chan<- wsIncoming) {
	defer close(ch)
	for {
		msg, err := conn.Read(ctx)
		if err != nil {
			select {
			case ch <- wsIncoming{err: err}:
			case <-ctx.Done():
			}
			return
		}
		select {
		case ch <- wsIncoming{msg: msg}:
		case <-ctx.Done():
			return
		}
	}
}

// wsWaitCmd blocks on the next pump item and tags the resulting message with
// its pane. Update re-issues it after each wsIncomingMsg to keep listening.
func wsWaitCmd(pane int, ch chan wsIncoming) tea.Cmd {
	return func() tea.Msg {
		item, ok := <-ch
		if !ok {
			return wsClosedMsg{pane: pane}
		}
		if item.err != nil {
			return wsRecvErrMsg{pane: pane, err: item.err}
		}
		return wsIncomingMsg{pane: pane, msg: item.msg}
	}
}

// wsSendCmd writes one message of the given kind. Success produces no message;
// only a failure comes back (tagged with its pane).
func wsSendCmd(pane int, ctx context.Context, conn execution.WSConn, kind execution.WSMessageKind, data []byte) tea.Cmd {
	return func() tea.Msg {
		if err := conn.Send(ctx, kind, data); err != nil {
			return wsSendErrMsg{pane: pane, err: err}
		}
		return nil
	}
}

// wsPingCmd sends a ping and times the round trip to the pong.
func wsPingCmd(pane int, ctx context.Context, conn execution.WSConn) tea.Cmd {
	return func() tea.Msg {
		start := time.Now()
		err := conn.Ping(ctx)
		return wsPingResultMsg{pane: pane, dur: time.Since(start), err: err}
	}
}
