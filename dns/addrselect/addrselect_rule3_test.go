package addrselect

import (
	"net"
	"net/netip"
	"testing"
)

func TestRule3(t *testing.T) {
	// Mock srcAddrs to control the "deprecated" status of sources
	origSrcAddrs := srcAddrs
	defer func() { srcAddrs = origSrcAddrs }()

	// DA1: source is Deprecated
	// DA2: source is Not Deprecated
	da1 := net.IPAddr{IP: net.ParseIP("2001:db8::1")}
	da2 := net.IPAddr{IP: net.ParseIP("2001:db8::2")}

	// Mock implementation
	srcAddrs = func(addrs []net.IPAddr) []srcInfo {
		res := make([]srcInfo, len(addrs))
		for i, a := range addrs {
			if a.IP.Equal(da1.IP) {
				// Deprecated source
				res[i] = srcInfo{
					Addr:         netip.MustParseAddr("2001:db8::3"),
					IsNative:     true,
					IsDeprecated: true,
				}
			} else if a.IP.Equal(da2.IP) {
				// Non-deprecated source
				res[i] = srcInfo{
					Addr:         netip.MustParseAddr("2001:db8::4"),
					IsNative:     true,
					IsDeprecated: false,
				}
			}
		}
		return res
	}

	// We expect DA2 (Not Deprecated) to be preferred over DA1 (Deprecated)
	// Assuming all other rules are equal.
	// Both are Global Unicast.

	input := []net.IPAddr{da1, da2}
	SortByRFC6724(input)

	if !input[0].IP.Equal(da2.IP) {
		t.Errorf("Expected %v (Non-deprecated) to be first, got %v", da2.IP, input[0].IP)
	}
}
