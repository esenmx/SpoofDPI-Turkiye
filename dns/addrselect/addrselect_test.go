package addrselect

import (
	"net"
	"net/netip"
	"testing"
)

func TestRule7(t *testing.T) {
	// Mock srcAddrs to control the "native" status of sources
	origSrcAddrs := srcAddrs
	defer func() { srcAddrs = origSrcAddrs }()

	// Define our inputs
	// DA1: reached via Native
	// DA2: reached via Tunnel
	da1 := net.IPAddr{IP: net.ParseIP("2001:db8::1")}
	da2 := net.IPAddr{IP: net.ParseIP("2001:db8::2")}

	// Mock implementation
	srcAddrs = func(addrs []net.IPAddr) []srcInfo {
		res := make([]srcInfo, len(addrs))
		for i, a := range addrs {
			if a.IP.Equal(da1.IP) {
				// Native source
				res[i] = srcInfo{
					Addr:     netip.MustParseAddr("2001:db8::3"),
					IsNative: true,
				}
			} else if a.IP.Equal(da2.IP) {
				// Tunnel source
				res[i] = srcInfo{
					Addr:     netip.MustParseAddr("2001:db8::4"),
					IsNative: false,
				}
			}
		}
		return res
	}

	// We expect DA1 (Native) to be preferred over DA2 (Tunnel)
	// Assuming all other rules are equal.
	// Both are Global Unicast (Scope Global, Precedence 40, Label 1)
	// Scope matches.

	input := []net.IPAddr{da2, da1}
	SortByRFC6724(input)

	if !input[0].IP.Equal(da1.IP) {
		t.Errorf("Expected %v (Native) to be first, got %v", da1.IP, input[0].IP)
	}
}
