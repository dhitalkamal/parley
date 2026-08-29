// Package wsclient implements the execution.WSDialer/WSConn ports over
// coder/websocket. It's the only place the concrete WebSocket library is
// referenced - the rest of parley speaks to the execution.WSConn interface, so
// the library stays swappable and the flow stays fakeable in tests.
package wsclient

import (
	"context"
	"net/http"

	"github.com/coder/websocket"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
)

// readLimit caps a single incoming message. The library defaults to 32KiB,
// which silently fails a large JSON frame; API responses over a socket can be
// bigger, so we lift it to something generous but still bounded (a guard
// against a server streaming unbounded data into memory).
const readLimit = 10 << 20 // 10 MiB

// Dialer is the default execution.WSDialer.
type Dialer struct{}

var _ execution.WSDialer = Dialer{}

// New returns the default WebSocket dialer.
func New() Dialer { return Dialer{} }

func (Dialer) Dial(ctx context.Context, url string, headers []collection.Header) (execution.WSConn, error) {
	opts := &websocket.DialOptions{}
	if h := toHeader(headers); len(h) > 0 {
		opts.HTTPHeader = h
	}
	c, _, err := websocket.Dial(ctx, url, opts)
	if err != nil {
		return nil, err
	}
	c.SetReadLimit(readLimit)
	return &conn{c: c}, nil
}

// toHeader builds the handshake header set from the enabled request headers,
// skipping disabled and blank-keyed rows.
func toHeader(headers []collection.Header) http.Header {
	h := http.Header{}
	for _, x := range headers {
		if !x.Enabled || x.Key == "" {
			continue
		}
		h.Add(x.Key, x.Value)
	}
	return h
}

// conn adapts a *websocket.Conn to execution.WSConn.
type conn struct {
	c *websocket.Conn
}

func (w *conn) Send(ctx context.Context, kind execution.WSMessageKind, data []byte) error {
	return w.c.Write(ctx, toMessageType(kind), data)
}

func (w *conn) Read(ctx context.Context) (execution.WSMessage, error) {
	typ, data, err := w.c.Read(ctx)
	if err != nil {
		return execution.WSMessage{}, err
	}
	return execution.WSMessage{Kind: fromMessageType(typ), Data: data}, nil
}

func (w *conn) Ping(ctx context.Context) error {
	return w.c.Ping(ctx)
}

func (w *conn) Close() error {
	return w.c.Close(websocket.StatusNormalClosure, "")
}

func toMessageType(kind execution.WSMessageKind) websocket.MessageType {
	if kind == execution.WSBinary {
		return websocket.MessageBinary
	}
	return websocket.MessageText
}

func fromMessageType(t websocket.MessageType) execution.WSMessageKind {
	if t == websocket.MessageBinary {
		return execution.WSBinary
	}
	return execution.WSText
}
