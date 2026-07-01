//go:build !linux

package addrselect

import "net/netip"

type addrFlags struct {
	IsDeprecated bool
	IsHome       bool
}

func getAddrFlags() map[netip.Addr]addrFlags {
	return nil
}
