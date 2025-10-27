package dns

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/esenmx/SpoofDPI/dns/resolver"
	"github.com/esenmx/SpoofDPI/util"
	"github.com/esenmx/SpoofDPI/util/log"
	"github.com/miekg/dns"
)

const scopeDNS = "DNS"

// cacheEntry holds a cached DNS result
type cacheEntry struct {
	ip        string
	expiresAt time.Time
}

// dnsCache is a simple LRU cache for DNS lookups
type dnsCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	maxSize int
}

var globalCache = &dnsCache{
	entries: make(map[string]*cacheEntry),
	maxSize: 1000, // Cache up to 1000 entries
}

func (c *dnsCache) get(host string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, found := c.entries[host]
	if !found {
		return "", false
	}

	// Check if expired
	if time.Now().After(entry.expiresAt) {
		return "", false
	}

	return entry.ip, true
}

func (c *dnsCache) set(host, ip string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If at max size, remove oldest entry
	if len(c.entries) >= c.maxSize && len(c.entries) > 0 {
		// Remove first entry (simple FIFO eviction)
		for k := range c.entries {
			delete(c.entries, k)
			break
		}
	}

	c.entries[host] = &cacheEntry{
		ip:        ip,
		expiresAt: time.Now().Add(ttl),
	}
}

type Resolver interface {
	Resolve(ctx context.Context, host string, qTypes []uint16) ([]net.IPAddr, error)
	String() string
}

type Dns struct {
	host          string
	port          string
	systemClient  Resolver
	generalClient Resolver
	dohClient     Resolver
	qTypes        []uint16
}

func NewDns(config *util.Config) *Dns {
	addr := config.DnsAddr
	port := strconv.Itoa(config.DnsPort)
	var qTypes []uint16
	if config.DnsIPv4Only {
		qTypes = []uint16{dns.TypeA}
	} else {
		qTypes = []uint16{dns.TypeAAAA, dns.TypeA}
	}
	return &Dns{
		host:          config.DnsAddr,
		port:          port,
		systemClient:  resolver.NewSystemResolver(),
		generalClient: resolver.NewGeneralResolver(net.JoinHostPort(addr, port)),
		dohClient:     resolver.NewDOHResolver(addr),
		qTypes:        qTypes,
	}
}

func (d *Dns) ResolveHost(ctx context.Context, host string, enableDoh bool, useSystemDns bool) (string, error) {
	ctx = util.GetCtxWithScope(ctx, scopeDNS)
	logger := log.GetCtxLogger(ctx)

	if ip, err := parseIpAddr(host); err == nil {
		return ip.String(), nil
	}

	// Check cache first
	if cached, found := globalCache.get(host); found {
		logger.Debug().Msgf("cache hit for %s -> %s", host, cached)
		return cached, nil
	}

	clt := d.clientFactory(enableDoh, useSystemDns)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	logger.Debug().Msgf("resolving %s using %s", host, clt)

	t := time.Now()

	addrs, err := clt.Resolve(ctx, host, d.qTypes)
	if err != nil {
		return "", fmt.Errorf("%s: %w", clt, err)
	}

	if len(addrs) > 0 {
		duration := time.Since(t).Milliseconds()
		ip := addrs[0].String()
		
		// Cache the result with 5 minute TTL
		globalCache.set(host, ip, 5*time.Minute)
		
		logger.Debug().Msgf("resolved %s from %s in %d ms", ip, host, duration)
		return ip, nil
	}

	return "", fmt.Errorf("could not resolve %s using %s", host, clt)
}

func (d *Dns) clientFactory(enableDoh bool, useSystemDns bool) Resolver {
	if useSystemDns {
		return d.systemClient
	}

	if enableDoh {
		return d.dohClient
	}

	return d.generalClient
}

func parseIpAddr(addr string) (*net.IPAddr, error) {
	ip := net.ParseIP(addr)
	if ip == nil {
		return nil, fmt.Errorf("%s is not an ip address", addr)
	}

	ipAddr := &net.IPAddr{
		IP: ip,
	}

	return ipAddr, nil
}
