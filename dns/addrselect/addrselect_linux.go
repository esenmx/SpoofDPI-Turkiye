//go:build linux

package addrselect

import (
	"bufio"
	"net/netip"
	"os"
	"strconv"
	"strings"
)

func getDeprecatedAddrs() map[netip.Addr]bool {
	m := make(map[netip.Addr]bool)
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

		if (flags & 0x20) != 0 {
			// Parse IP (index 0)
			ip, err := parseIPv6Hex(fields[0])
			if err == nil {
				m[ip] = true
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
