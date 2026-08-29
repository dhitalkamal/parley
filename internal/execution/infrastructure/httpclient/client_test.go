package httpclient

import (
	"context"
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDo_MultipleSetCookieHeadersStaySeparate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Set-Cookie", "a=1; Path=/; Expires=Wed, 09 Jun 2021 10:18:14 GMT")
		w.Header().Add("Set-Cookie", "b=2; Path=/")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New()
	resp, err := c.Do(context.Background(), collection.Request{Method: collection.GET, URL: srv.URL})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var cookieHeaders []string
	for _, h := range resp.Headers {
		if h.Key == "Set-Cookie" {
			cookieHeaders = append(cookieHeaders, h.Value)
		}
	}
	if len(cookieHeaders) != 2 {
		t.Fatalf("got %d Set-Cookie headers, want 2 (not comma-joined): %v", len(cookieHeaders), cookieHeaders)
	}
	if cookieHeaders[0] != "a=1; Path=/; Expires=Wed, 09 Jun 2021 10:18:14 GMT" {
		t.Errorf("first Set-Cookie = %q", cookieHeaders[0])
	}
	if cookieHeaders[1] != "b=2; Path=/" {
		t.Errorf("second Set-Cookie = %q", cookieHeaders[1])
	}
}

func TestDo_SendsMethodURLHeadersAndBody(t *testing.T) {
	var gotMethod, gotPath, gotQuery, gotAuth, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("X-Reply", "pong")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	req := collection.Request{
		Method: collection.POST,
		URL:    srv.URL + "/users",
		Params: []collection.QueryParam{
			{Key: "page", Value: "2", Enabled: true},
		},
		Headers: []collection.Header{
			{Key: "Authorization", Value: "Bearer xyz", Enabled: true},
			{Key: "X-Skip", Value: "nope", Enabled: false},
		},
		Body: collection.Body{
			Type:           collection.BodyRaw,
			RawContentType: "application/json",
			RawText:        `{"name":"ada"}`,
		},
		Timeout:         5 * time.Second,
		FollowRedirects: true,
	}

	c := New()
	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/users" {
		t.Errorf("path = %q, want /users", gotPath)
	}
	if gotQuery != "page=2" {
		t.Errorf("query = %q, want page=2", gotQuery)
	}
	if gotAuth != "Bearer xyz" {
		t.Errorf("Authorization header = %q, want Bearer xyz", gotAuth)
	}
	if gotBody != `{"name":"ada"}` {
		t.Errorf("body = %q, want {\"name\":\"ada\"}", gotBody)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status code = %d, want 201", resp.StatusCode)
	}
	if string(resp.Body) != `{"ok":true}` {
		t.Errorf("resp body = %q", resp.Body)
	}
	if resp.Elapsed <= 0 {
		t.Errorf("elapsed = %v, want > 0", resp.Elapsed)
	}
	found := false
	for _, h := range resp.Headers {
		if h.Key == "X-Reply" && h.Value == "pong" {
			found = true
		}
	}
	if !found {
		t.Errorf("response headers missing X-Reply: %+v", resp.Headers)
	}
}

func TestDo_TimeoutReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
	}))
	defer srv.Close()

	req := collection.Request{
		Method:  collection.GET,
		URL:     srv.URL,
		Timeout: 5 * time.Millisecond,
	}

	c := New()
	_, err := c.Do(context.Background(), req)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

func TestDo_NoFollowRedirectsStopsAtFirstHop(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer redirector.Close()

	req := collection.Request{
		Method:          collection.GET,
		URL:             redirector.URL,
		Timeout:         2 * time.Second,
		FollowRedirects: false,
	}

	c := New()
	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusFound {
		t.Errorf("status code = %d, want 302 (redirect not followed)", resp.StatusCode)
	}
}

func TestDo_InsecureSkipVerifyAllowsSelfSignedTLS(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	req := collection.Request{
		Method:             collection.GET,
		URL:                srv.URL,
		Timeout:            2 * time.Second,
		FollowRedirects:    true,
		InsecureSkipVerify: true,
	}

	c := New()
	resp, err := c.Do(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error with InsecureSkipVerify: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status code = %d, want 200", resp.StatusCode)
	}
}

// TestDo_CapturesServerWaitTiming guards the Timeline tab's data source: the
// server-wait phase (request fully sent to first response byte) should
// roughly track a deliberate handler delay, not stay at zero.
func TestDo_CapturesServerWaitTiming(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(30 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New()
	resp, err := c.Do(context.Background(), collection.Request{Method: collection.GET, URL: srv.URL, Timeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Timing.ServerWait < 25*time.Millisecond {
		t.Errorf("ServerWait = %v, want at least ~30ms to reflect the handler's deliberate delay", resp.Timing.ServerWait)
	}
	if resp.Timing.ServerWait > resp.Elapsed {
		t.Errorf("ServerWait = %v, want it no greater than total Elapsed %v", resp.Timing.ServerWait, resp.Elapsed)
	}
}

// TestDo_CapturesTLSHandshakeTiming guards TLS-specific timing: only an
// HTTPS request should report a nonzero TLSHandshake phase.
func TestDo_CapturesTLSHandshakeTiming(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New()
	resp, err := c.Do(context.Background(), collection.Request{
		Method: collection.GET, URL: srv.URL, Timeout: 2 * time.Second, InsecureSkipVerify: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Timing.TLSHandshake <= 0 {
		t.Errorf("TLSHandshake = %v, want greater than zero for an HTTPS request", resp.Timing.TLSHandshake)
	}
}

// TestDo_PlainHTTPHasNoTLSHandshakeTiming guards against a fake/estimated
// value: a plain HTTP request never handshakes, so this phase must stay
// exactly zero rather than reporting something misleading.
func TestDo_PlainHTTPHasNoTLSHandshakeTiming(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New()
	resp, err := c.Do(context.Background(), collection.Request{Method: collection.GET, URL: srv.URL, Timeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Timing.TLSHandshake != 0 {
		t.Errorf("TLSHandshake = %v, want exactly 0 for plain HTTP", resp.Timing.TLSHandshake)
	}
}

func TestDo_RejectsUntrustedTLSWhenNotSkipped(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	req := collection.Request{
		Method:             collection.GET,
		URL:                srv.URL,
		Timeout:            2 * time.Second,
		InsecureSkipVerify: false,
	}

	c := New()
	_, err := c.Do(context.Background(), req)
	if err == nil {
		t.Fatal("expected TLS verification error, got nil")
	}
}
