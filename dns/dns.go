package dns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/esenmx/SpoofDPI-Turkiye/dns/resolver"
	"github.com/esenmx/SpoofDPI-Turkiye/util"
	"github.com/esenmx/SpoofDPI-Turkiye/util/log"
	"github.com/miekg/dns"
)

const scopeDNS = "DNS"

type Resolver interface {
	Resolve(ctx context.Context, host string, qTypes []uint16) ([]net.IPAddr, error)
	String() string
}

type Dns struct {
	systemClient Resolver
	bypassClient Resolver
	qTypes       []uint16
	totalBudget  time.Duration
}

func NewDns(config *util.Config) *Dns {
	qTypes := []uint16{dns.TypeAAAA, dns.TypeA}
	if config.DnsIPv4Only {
		qTypes = []uint16{dns.TypeA}
	}

	port := strconv.Itoa(config.DnsPort)

	var chain []resolver.Resolver
	seen := make(map[string]struct{})
	addPlain := func(addr string) {
		addr = strings.TrimSpace(addr)
		if addr == "" {
			return
		}
		server := net.JoinHostPort(addr, port)
		if _, dup := seen[server]; dup {
			return
		}
		seen[server] = struct{}{}
		chain = append(chain, resolver.NewGeneralResolver(server))
	}

	addPlain(config.DnsAddr)
	for _, fb := range config.DnsFallback {
		addPlain(fb)
	}

	if config.EnableDoh && config.DohUrl != "" {
		chain = append(chain, resolver.NewDOHResolver(config.DohUrl, config.DohBootstrapIp))
	}

	// IMPORTANT: do NOT add the system resolver to this chain. ChainResolver
	// races every entry in parallel and returns the fastest non-empty answer.
	// On Turkish ISPs the local system resolver is the closest hop and would
	// win the race for blocked domains with the ISP's poisoned IP, defeating
	// the entire bypass. The system resolver is used only on the
	// useSystemDns=true branch (non-bypass traffic) below.
	bypass := resolver.NewChainResolver(chain, 1500*time.Millisecond)
	cached := resolver.NewCache(bypass, 5*time.Minute, 10*time.Second, 4096)

	return &Dns{
		systemClient: resolver.NewSystemResolver(),
		bypassClient: cached,
		qTypes:       qTypes,
		totalBudget:  3 * time.Second,
	}
}

func (d *Dns) ResolveHost(ctx context.Context, host string, _enableDoh bool, useSystemDns bool) (string, error) {
	ctx = util.GetCtxWithScope(ctx, scopeDNS)
	logger := log.GetCtxLogger(ctx)

	if ip, err := parseIpAddr(host); err == nil {
		return ip.String(), nil
	}

	clt := d.bypassClient
	if useSystemDns {
		clt = d.systemClient
	}

	ctx, cancel := context.WithTimeout(ctx, d.totalBudget)
	defer cancel()

	logger.Debug().Msgf("resolving %s using %s", host, clt)
	t := time.Now()

	addrs, err := clt.Resolve(ctx, host, d.qTypes)
	if err != nil {
		return "", fmt.Errorf("%s: %w", clt, err)
	}
	if len(addrs) == 0 {
		return "", fmt.Errorf("%s returned no addresses for %s", clt, host)
	}

	logger.Debug().Msgf("resolved %s -> %s (%d candidates) in %d ms",
		host, addrs[0].String(), len(addrs), time.Since(t).Milliseconds())
	return addrs[0].String(), nil
}

func parseIpAddr(addr string) (*net.IPAddr, error) {
	ip := net.ParseIP(addr)
	if ip == nil {
		return nil, errors.New("not an ip address")
	}
	return &net.IPAddr{IP: ip}, nil
}
