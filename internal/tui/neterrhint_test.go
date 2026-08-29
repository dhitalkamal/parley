package tui

import (
	"errors"
	"strings"
	"testing"
)

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "context deadline exceeded" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return false }

func TestNetworkErrorHint(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		vpn      string
		expected string
		wantSub  string // "" means expect an empty hint
	}{
		{"no route, no vpn", errors.New("dial tcp 10.0.0.1:443: connect: no route to host"), "", "", "behind a VPN"},
		{"timeout mentions vpn name", timeoutErr{}, "backend-kamal", "", "backend-kamal"},
		{"unreachable notes no vpn", errors.New("connect: network is unreachable"), "", "", "No VPN detected"},
		{"dns failure", errors.New("dial tcp: lookup api.internal: no such host"), "tailscale0", "", "DNS lookup failed"},
		{"connection refused, no vpn suggestion", errors.New("connect: connection refused"), "", "", "nothing is listening"},
		{"non-network error has no hint", errors.New("json: cannot unmarshal"), "tailscale0", "", ""},
		{"nil error", nil, "", "", ""},
		// expected VPN set but not among the detected tunnels: the timeout hint
		// should call it out by name.
		{"expected vpn down", timeoutErr{}, "tailscale0", "backend-kamal", "backend-kamal"},
	}
	for _, c := range cases {
		got := networkErrorHint(c.err, c.vpn, c.expected)
		if c.wantSub == "" {
			if got != "" {
				t.Errorf("%s: got hint %q, want empty", c.name, got)
			}
			continue
		}
		if !strings.Contains(got, c.wantSub) {
			t.Errorf("%s: hint = %q, want it to contain %q", c.name, got, c.wantSub)
		}
	}
	// connection-refused should NOT suggest a VPN (host is reachable).
	if h := networkErrorHint(errors.New("connection refused"), "tailscale0", ""); strings.Contains(h, "VPN") {
		t.Errorf("connection refused should not mention VPN, got %q", h)
	}
}
