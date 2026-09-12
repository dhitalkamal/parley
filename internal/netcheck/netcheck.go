// Package netcheck does best-effort, read-only detection of whether a VPN or
// tunnel interface is currently up on this machine. It never changes network
// state and never runs privileged commands - it only reads the OS interface
// list (net.Interfaces), so it works cross-platform with no configuration.
package netcheck

import (
	"net"
	"strings"
)

// iface is a testable snapshot of one network interface. Real values come from
// net.Interfaces; tests inject their own via listInterfaces.
type iface struct {
	name          string
	up            bool
	loopback      bool
	pointToPoint  bool
	broadcast     bool
	hasRoutableIP bool // has a non-loopback, non-link-local unicast address
}

// listInterfaces is swapped out in tests.
var listInterfaces = defaultListInterfaces

func defaultListInterfaces() []iface {
	netifs, err := net.Interfaces()
	if err != nil {
		return nil
	}
	out := make([]iface, 0, len(netifs))
	for _, ni := range netifs {
		out = append(out, iface{
			name:          ni.Name,
			up:            ni.Flags&net.FlagUp != 0,
			loopback:      ni.Flags&net.FlagLoopback != 0,
			pointToPoint:  ni.Flags&net.FlagPointToPoint != 0,
			broadcast:     ni.Flags&net.FlagBroadcast != 0,
			hasRoutableIP: hasRoutableAddr(ni),
		})
	}
	return out
}

// hasRoutableAddr reports whether the interface has at least one routable
// unicast IP (not loopback, not link-local). A VPN/tunnel that's been brought
// down often keeps its interface but loses its IP (e.g. `tailscale down` leaves
// tailscale0 up but strips the 100.x address), so "has an IP" is what actually
// distinguishes connected from disconnected.
func hasRoutableAddr(ni net.Interface) bool {
	addrs, err := ni.Addrs()
	if err != nil {
		return false
	}
	for _, a := range addrs {
		var ip net.IP
		switch v := a.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			continue
		}
		if ip.IsGlobalUnicast() {
			return true
		}
	}
	return false
}

// nonVPNPrefixes are interface-name prefixes that are physical NICs, bridges,
// or container/virtualization plumbing - never a user VPN.
var nonVPNPrefixes = []string{
	"lo", "eth", "en", "em", "eno", "ens", "enp", "wl", "wlan", "wlp", "wlo",
	"docker", "br-", "br0", "bond", "veth", "virbr", "vmnet", "vboxnet",
	"cni", "flannel", "cali", "cilium", "kube", "vnet",
}

// vpnPrefixes are names that are unmistakably a VPN/tunnel.
var vpnPrefixes = []string{
	"tun", "tap", "wg", "utun", "ppp", "tailscale", "nordlynx", "proton",
	"ipsec", "wireguard", "vpn", "gpd", "zt",
}

func hasAnyPrefix(name string, prefixes []string) bool {
	name = strings.ToLower(name)
	for _, p := range prefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

func isVPNLike(i iface) bool {
	if !i.up || i.loopback || !i.hasRoutableIP {
		return false
	}
	if hasAnyPrefix(i.name, nonVPNPrefixes) {
		return false
	}
	return i.pointToPoint || !i.broadcast || hasAnyPrefix(i.name, vpnPrefixes)
}

// DetectVPNs returns the names of every active VPN/tunnel interface (up,
// non-loopback, with a routable IP, and tunnel-shaped). Empty when none.
func DetectVPNs() []string {
	var names []string
	for _, i := range listInterfaces() {
		if isVPNLike(i) {
			names = append(names, i.name)
		}
	}
	return names
}

