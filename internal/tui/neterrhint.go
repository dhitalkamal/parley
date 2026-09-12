package tui

import (
	"errors"
	"net"
	"strings"
)

// networkErrorHint turns a failed request's raw error into a short, plain-
// language hint - classifying the common network failures and noting the
// current VPN state (see netcheck.DetectVPNs) so a gated endpoint that times out
// reads as "connect your VPN" rather than a cryptic ETIMEDOUT. Returns "" for
// errors that aren't connectivity failures (those speak for themselves).
func networkErrorHint(err error, detectedVPN, expectedVPN string) string {
	if err == nil {
		return ""
	}
	msg := strings.ToLower(err.Error())

	vpnState := "No VPN detected."
	if detectedVPN != "" {
		vpnState = "VPN detected: " + detectedVPN + "."
	}
	// If this environment expects a specific VPN and it isn't among the tunnels
	// currently up, say so by name - that's almost certainly why it's timing out.
	if expectedVPN != "" && !strings.Contains(detectedVPN, expectedVPN) {
		vpnState = "Expected VPN '" + expectedVPN + "' is not connected."
	}

	switch {
	case isTimeout(err),
		containsAny(msg, "no route to host", "network is unreachable", "host is down", "host unreachable", "i/o timeout", "deadline exceeded", "timed out", "timeout"):
		return "Couldn't reach the host. If this endpoint is behind a VPN or tunnel, make sure it's connected.  (" + vpnState + ")"
	case containsAny(msg, "no such host", "server misbehaving", "name resolution", "dns"):
		return "DNS lookup failed - the host name couldn't be resolved. If it's an internal name behind a VPN, connect it first.  (" + vpnState + ")"
	case strings.Contains(msg, "connection refused"):
		return "Connection refused - the host answered but nothing is listening on that port. Check the URL and port."
	}
	return ""
}

func isTimeout(err error) bool {
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
