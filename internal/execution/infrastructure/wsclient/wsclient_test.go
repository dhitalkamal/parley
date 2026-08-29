package wsclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
)

// echoServer accepts a connection and echoes every frame back with the same
// type, so a text frame round-trips as text and a binary frame as binary. Its
// read loop also lets the library process incoming control frames (ping/pong).
func echoServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close(websocket.StatusNormalClosure, "")
		for {
			typ, data, err := c.Read(r.Context())
			if err != nil {
				return
			}
			if err := c.Write(r.Context(), typ, data); err != nil {
				return
			}
		}
	}))
}

func wsURL(srv *httptest.Server) string {
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

// TestDialer_TextAndBinaryRoundTrip drives the real client: a text frame comes
// back as text, a binary frame as binary, and an enabled header reaches the
// server while a disabled one doesn't.
func TestDialer_TextAndBinaryRoundTrip(t *testing.T) {
	var gotAuth, gotSkipped string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotSkipped = r.Header.Get("X-Skip")
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close(websocket.StatusNormalClosure, "")
		for {
			typ, data, err := c.Read(r.Context())
			if err != nil {
				return
			}
			if err := c.Write(r.Context(), typ, data); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := New().Dial(ctx, wsURL(srv), []collection.Header{
		{Key: "Authorization", Value: "Bearer t", Enabled: true},
		{Key: "X-Skip", Value: "nope", Enabled: false},
	})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// text
	if err := conn.Send(ctx, execution.WSText, []byte("hello")); err != nil {
		t.Fatalf("send text: %v", err)
	}
	msg, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read text: %v", err)
	}
	if msg.Kind != execution.WSText || string(msg.Data) != "hello" {
		t.Errorf("text echo = %+v, want text 'hello'", msg)
	}

	// binary
	if err := conn.Send(ctx, execution.WSBinary, []byte{0x00, 0x01, 0x02}); err != nil {
		t.Fatalf("send binary: %v", err)
	}
	msg, err = conn.Read(ctx)
	if err != nil {
		t.Fatalf("read binary: %v", err)
	}
	if msg.Kind != execution.WSBinary || len(msg.Data) != 3 || msg.Data[2] != 0x02 {
		t.Errorf("binary echo = %+v, want binary [0 1 2]", msg)
	}

	if gotAuth != "Bearer t" {
		t.Errorf("Authorization = %q, want 'Bearer t'", gotAuth)
	}
	if gotSkipped != "" {
		t.Errorf("disabled header leaked: %q", gotSkipped)
	}
}

// TestConn_Ping sends a ping and expects the pong. coder/websocket processes
// the pong during a concurrent Read, so a background reader runs alongside.
func TestConn_Ping(t *testing.T) {
	srv := echoServer()
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := New().Dial(ctx, wsURL(srv), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	go func() { _, _ = conn.Read(ctx) }() // lets the library handle the pong

	if err := conn.Ping(ctx); err != nil {
		t.Errorf("ping: %v", err)
	}
}

func TestDialer_DialErrorOnBadURL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := New().Dial(ctx, "ws://127.0.0.1:1/nope", nil); err == nil {
		t.Error("expected an error dialing an unreachable server")
	}
}
