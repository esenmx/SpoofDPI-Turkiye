//go:build linux

package addrselect

import (
	"bufio"
	"net/netip"
	"os"
	"strconv"
	"strings"
)

type addrFlags struct {
	IsDeprecated bool
	IsHome       bool
}

func getAddrFlags() map[netip.Addr]addrFlags {
	m := make(map[netip.Addr]addrFlags)
	f, err := os.Open("/proc/net/if_inet6")
	if err != nil {
		return nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		// Parse flags (index 4)
		flags, err := strconv.ParseUint(fields[4], 16, 64)
		if err != nil {
			continue
		}

		isDeprecated := (flags & 0x20) != 0
		isHome := (flags & 0x10) != 0

		if isDeprecated || isHome {
			// Parse IP (index 0)
			ip, err := parseIPv6Hex(fields[0])
			if err == nil {
				m[ip] = addrFlags{
					IsDeprecated: isDeprecated,
					IsHome:       isHome,
				}
			}
		}
	}
	return m
}

func parseIPv6Hex(s string) (netip.Addr, error) {
	if len(s) != 32 {
		return netip.Addr{}, os.ErrInvalid
	}
	var ip [16]byte
	for i := 0; i < 16; i++ {
		b, err := strconv.ParseUint(s[i*2:i*2+2], 16, 8)
		if err != nil {
			return netip.Addr{}, err
		}
		ip[i] = byte(b)
	}
	return netip.AddrFrom16(ip), nil
}
