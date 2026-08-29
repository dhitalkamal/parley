// Package httpclient implements execution.HTTPClient against net/http.
package httpclient

import (
	"context"
	"crypto/tls"
	collection "github.com/dhitalkamal/parley/internal/collection/domain"
	execution "github.com/dhitalkamal/parley/internal/execution/domain"
	"io"
	"net/http"
	"net/http/httptrace"
	"strings"
	"time"
)

// Client executes collection.Request values over the network.
type Client struct{}

// New returns a ready-to-use Client.
func New() *Client {
	return &Client{}
}

// timingCollector accumulates an execution.Timing breakdown from
// net/http/httptrace callbacks, all of which fire synchronously on the
// same goroutine that calls client.Do below - no locking needed.
type timingCollector struct {
	dnsStart     time.Time
	connectStart time.Time
	tlsStart     time.Time
	wroteRequest time.Time
	firstByte    time.Time
	timing       execution.Timing
}

func (tc *timingCollector) trace() *httptrace.ClientTrace {
	return &httptrace.ClientTrace{
		DNSStart: func(httptrace.DNSStartInfo) { tc.dnsStart = time.Now() },
		DNSDone: func(httptrace.DNSDoneInfo) {
			if !tc.dnsStart.IsZero() {
				tc.timing.DNSLookup = time.Since(tc.dnsStart)
			}
		},
		ConnectStart: func(network, addr string) { tc.connectStart = time.Now() },
		ConnectDone: func(network, addr string, err error) {
			if !tc.connectStart.IsZero() && err == nil {
				tc.timing.TCPConnect = time.Since(tc.connectStart)
			}
		},
		TLSHandshakeStart: func() { tc.tlsStart = time.Now() },
		TLSHandshakeDone: func(tls.ConnectionState, error) {
			if !tc.tlsStart.IsZero() {
				tc.timing.TLSHandshake = time.Since(tc.tlsStart)
			}
		},
		WroteRequest: func(httptrace.WroteRequestInfo) { tc.wroteRequest = time.Now() },
		GotFirstResponseByte: func() {
			tc.firstByte = time.Now()
			if !tc.wroteRequest.IsZero() {
				tc.timing.ServerWait = tc.firstByte.Sub(tc.wroteRequest)
			}
		},
	}
}

// Do builds and executes an *http.Request from req, applying its timeout,
// redirect, and TLS-verification settings, and returns the result as a
// execution.Response.
func (c *Client) Do(ctx context.Context, req collection.Request) (execution.Response, error) {
	url, err := collection.BuildURL(req.URL, req.Params)
	if err != nil {
		return execution.Response{}, err
	}

	bodyReader, bodyContentType, err := buildRequestBody(req.Body)
	if err != nil {
		return execution.Response{}, err
	}

	tc := &timingCollector{}
	traceCtx := httptrace.WithClientTrace(ctx, tc.trace())
	httpReq, err := http.NewRequestWithContext(traceCtx, string(req.Method), url, bodyReader)
	if err != nil {
		return execution.Response{}, err
	}
	for _, h := range collection.EffectiveHeaders(req.Headers) {
		httpReq.Header.Set(h.Key, h.Value)
	}
	if bodyContentType != "" && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", bodyContentType)
	}

	client := &http.Client{
		Timeout: req.Timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: req.InsecureSkipVerify},
		},
	}
	if !req.FollowRedirects {
		client.CheckRedirect = func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	start := time.Now()
	resp, err := client.Do(httpReq)
	elapsed := time.Since(start)
	if err != nil {
		return execution.Response{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return execution.Response{}, err
	}
	if !tc.firstByte.IsZero() {
		tc.timing.ContentTransfer = time.Since(tc.firstByte)
	}

	// Set-Cookie must never be comma-joined (RFC 6265): a cookie's own
	// Expires attribute contains a literal comma, so joining multiple
	// Set-Cookie values the way other repeated headers are joined would
	// produce an unparseable mess. Emit one collection.Header per cookie instead.
	var headers []collection.Header
	for k, values := range resp.Header {
		if strings.EqualFold(k, "Set-Cookie") {
			for _, v := range values {
				headers = append(headers, collection.Header{Key: k, Value: v, Enabled: true})
			}
			continue
		}
		headers = append(headers, collection.Header{Key: k, Value: strings.Join(values, ", "), Enabled: true})
	}

	return execution.Response{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    headers,
		Body:       respBody,
		Elapsed:    elapsed,
		Timing:     tc.timing,
	}, nil
}
