package tui

import (
	"context"
	"errors"
	"sync"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
)

// fakeWSConn is an in-memory execution.WSConn for driving the WebSocket flow
// without a real socket: sends are recorded, reads block until a message is
// pushed onto reads or the connection is closed.
type fakeSent struct {
	kind execution.WSMessageKind
	data string
}

type fakeWSConn struct {
	mu     sync.Mutex
	sent   []fakeSent
	pings  int
	reads  chan execution.WSMessage
	closed chan struct{}
	once   sync.Once
}

func newFakeWSConn() *fakeWSConn {
	return &fakeWSConn{reads: make(chan execution.WSMessage, 8), closed: make(chan struct{})}
}

func (f *fakeWSConn) Send(_ context.Context, kind execution.WSMessageKind, data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, fakeSent{kind: kind, data: string(data)})
	return nil
}

func (f *fakeWSConn) Read(ctx context.Context) (execution.WSMessage, error) {
	select {
	case m := <-f.reads:
		return m, nil
	case <-f.closed:
		return execution.WSMessage{}, errors.New("closed")
	case <-ctx.Done():
		return execution.WSMessage{}, ctx.Err()
	}
}

func (f *fakeWSConn) Ping(_ context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pings++
	return nil
}

func (f *fakeWSConn) Close() error {
	f.once.Do(func() { close(f.closed) })
	return nil
}

// pushText/pushBinary queue a frame for the next Read.
func (f *fakeWSConn) pushText(s string) {
	f.reads <- execution.WSMessage{Kind: execution.WSText, Data: []byte(s)}
}
func (f *fakeWSConn) pushBinary(b []byte) {
	f.reads <- execution.WSMessage{Kind: execution.WSBinary, Data: b}
}

// fakeWSDialer hands back a preset connection (or error) and records what it
// was asked to dial, so tests can assert on the resolved URL/headers.
type fakeWSDialer struct {
	conn          execution.WSConn
	err           error
	dialedURL     string
	dialedHeaders []collection.Header
}

func (f *fakeWSDialer) Dial(_ context.Context, url string, headers []collection.Header) (execution.WSConn, error) {
	f.dialedURL = url
	f.dialedHeaders = headers
	if f.err != nil {
		return nil, f.err
	}
	return f.conn, nil
}
