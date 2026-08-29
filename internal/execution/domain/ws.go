package execution

import (
	"context"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
)

// WSMessageKind distinguishes a text frame from a binary one.
type WSMessageKind int

const (
	WSText   WSMessageKind = iota // a UTF-8 text frame
	WSBinary                      // a binary frame
)

// WSMessage is one frame read off a connection: its kind plus its raw bytes.
type WSMessage struct {
	Kind WSMessageKind
	Data []byte
}

// WSDialer opens a WebSocket connection. It's the long-lived counterpart to
// HTTPClient: instead of one request -> one response, a successful Dial hands
// back a WSConn that stays open for bidirectional messaging until closed.
// Implemented by execution/infrastructure/wsclient; faked in TUI tests so the
// connect/send/receive flow can be driven without a real socket.
type WSDialer interface {
	// Dial connects to url (ws:// or wss://) sending headers on the opening
	// handshake (for auth etc.). The context bounds only the handshake, not the
	// lifetime of the returned connection.
	Dial(ctx context.Context, url string, headers []collection.Header) (WSConn, error)
}

// WSConn is a single open WebSocket connection. Reads and writes may each run
// from their own goroutine concurrently (one reader, one writer) - the TUI
// reads in a background pump and writes from send/ping commands - but not two
// readers or two writers at once. Close unblocks a pending Read.
type WSConn interface {
	// Send writes one message of the given kind.
	Send(ctx context.Context, kind WSMessageKind, data []byte) error
	// Read blocks until the next message arrives, returning it, or an error
	// when the connection closes or fails.
	Read(ctx context.Context) (WSMessage, error)
	// Ping sends a ping and waits for the matching pong (a liveness check /
	// round-trip measurement). Errors if no pong arrives before ctx is done.
	Ping(ctx context.Context) error
	// Close tears down the connection; safe to call once.
	Close() error
}
