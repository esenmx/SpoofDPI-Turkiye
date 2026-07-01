package addrselect

import (
	"net"
	"net/netip"
	"testing"
)

func TestRule4(t *testing.T) {
	origSrcAddrs := srcAddrs
	defer func() { srcAddrs = origSrcAddrs }()

	da1 := net.IPAddr{IP: net.ParseIP("2001:db8::1")}
	da2 := net.IPAddr{IP: net.ParseIP("2001:db8::2")}

	srcAddrs = func(addrs []net.IPAddr) []srcInfo {
		res := make([]srcInfo, len(addrs))
		for i, a := range addrs {
			if a.IP.Equal(da1.IP) {
				// Home address
				res[i] = srcInfo{
					Addr:   netip.MustParseAddr("2001:db8::3"),
					IsHome: true,
				}
			} else if a.IP.Equal(da2.IP) {
				// Not a home address
				res[i] = srcInfo{
					Addr:   netip.MustParseAddr("2001:db8::4"),
					IsHome: false,
				}
			}
		}
		return res
	}

	input := []net.IPAddr{da2, da1}
	SortByRFC6724(input)

	if !input[0].IP.Equal(da1.IP) {
		t.Errorf("Expected %v (Home Address) to be first, got %v", da1.IP, input[0].IP)
	}
}
