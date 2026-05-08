package resolver

import (
	"context"
	"net"

	"github.com/miekg/dns"
)

type SystemResolver struct {
	*net.Resolver
}

func NewSystemResolver() *SystemResolver {
	return &SystemResolver{
		&net.Resolver{PreferGo: true},
	}
}

func (r *SystemResolver) String() string {
	return "system"
}

func (r *SystemResolver) Resolve(ctx context.Context, host string, qTypes []uint16) ([]net.IPAddr, error) {
	network := networkForQTypes(qTypes)
	ips, err := r.LookupIP(ctx, network, host)
	if err != nil {
		return nil, err
	}
	addrs := make([]net.IPAddr, 0, len(ips))
	for _, ip := range ips {
		addrs = append(addrs, net.IPAddr{IP: ip})
	}
	if len(addrs) > 1 {
		sortAddrs(addrs)
	}
	return addrs, nil
}

func networkForQTypes(qTypes []uint16) string {
	has4, has6 := false, false
	for _, q := range qTypes {
		switch q {
		case dns.TypeA:
			has4 = true
		case dns.TypeAAAA:
			has6 = true
		}
	}
	switch {
	case has4 && !has6:
		return "ip4"
	case has6 && !has4:
		return "ip6"
	default:
		return "ip"
	}
}
