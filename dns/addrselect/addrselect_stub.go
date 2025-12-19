//go:build !linux

package addrselect

import "net/netip"

func getDeprecatedAddrs() map[netip.Addr]bool {
	return nil
}
