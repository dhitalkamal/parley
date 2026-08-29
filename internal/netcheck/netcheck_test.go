package netcheck

import (
	"testing"
)

func withIfaces(t *testing.T, ifs []iface) {
	t.Helper()
	prev := listInterfaces
	listInterfaces = func() []iface { return ifs }
	t.Cleanup(func() { listInterfaces = prev })
}

// TestDetectVPN_FindsPointToPointTunnel guards the tailscale0-style case: a
// point-to-point, up, IP'd interface is detected.
func TestDetectVPN_FindsPointToPointTunnel(t *testing.T) {
	withIfaces(t, []iface{
		{name: "lo", up: true, loopback: true},
		{name: "wlan0", up: true, broadcast: true, hasRoutableIP: true},
		{name: "docker0", up: true, broadcast: true, hasRoutableIP: true},
		{name: "tailscale0", up: true, pointToPoint: true, hasRoutableIP: true},
	})
	if got := DetectVPN(); got != "tailscale0" {
		t.Errorf("DetectVPN() = %q, want tailscale0", got)
	}
}

// TestDetectVPN_FindsOddlyNamedWireGuard guards the backend-kamal case.
func TestDetectVPN_FindsOddlyNamedWireGuard(t *testing.T) {
	withIfaces(t, []iface{
		{name: "enp3s0", up: true, broadcast: true, hasRoutableIP: true},
		{name: "backend-kamal", up: true, hasRoutableIP: true}, // WireGuard: no broadcast
	})
	if got := DetectVPN(); got != "backend-kamal" {
		t.Errorf("DetectVPN() = %q, want backend-kamal", got)
	}
}

// TestDetectVPN_DownTunnelKeepsInterfaceButLosesIP is the reported bug: after
// `tailscale down` the interface stays up but has no routable IP, so it must NOT
// count as connected.
func TestDetectVPN_DownTunnelKeepsInterfaceButLosesIP(t *testing.T) {
	withIfaces(t, []iface{
		{name: "wlan0", up: true, broadcast: true, hasRoutableIP: true},
		{name: "tailscale0", up: true, pointToPoint: true, hasRoutableIP: false}, // down: no IP
	})
	if got := DetectVPN(); got != "" {
		t.Errorf("DetectVPN() = %q, want empty (tunnel up but no IP = disconnected)", got)
	}
}

// TestDetectVPN_NoneWhenOnlyPhysicalAndBridges guards the common no-VPN case.
func TestDetectVPN_NoneWhenOnlyPhysicalAndBridges(t *testing.T) {
	withIfaces(t, []iface{
		{name: "lo", up: true, loopback: true},
		{name: "wlan0", up: true, broadcast: true, hasRoutableIP: true},
		{name: "docker0", up: true, broadcast: true, hasRoutableIP: true},
		{name: "veth1234", up: true, broadcast: true, hasRoutableIP: true},
	})
	if got := DetectVPN(); got != "" {
		t.Errorf("DetectVPN() = %q, want empty (no VPN)", got)
	}
}

// TestDetectVPNs_ReportsMultiple guards that all active tunnels are returned.
func TestDetectVPNs_ReportsMultiple(t *testing.T) {
	withIfaces(t, []iface{
		{name: "wlan0", up: true, broadcast: true, hasRoutableIP: true},
		{name: "tailscale0", up: true, pointToPoint: true, hasRoutableIP: true},
		{name: "backend-kamal", up: true, hasRoutableIP: true},
	})
	got := DetectVPNs()
	if len(got) != 2 || got[0] != "tailscale0" || got[1] != "backend-kamal" {
		t.Errorf("DetectVPNs() = %v, want [tailscale0 backend-kamal]", got)
	}
	if s := DetectVPN(); s != "tailscale0, backend-kamal" {
		t.Errorf("DetectVPN() = %q, want the two joined", s)
	}
}
