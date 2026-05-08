package resolver

import (
	"context"
	"net"
	"sync"
	"time"
	"unsafe"
)

// Cache wraps a Resolver with a small TTL+negative cache. It coalesces
// concurrent in-flight queries for the same host so a burst of requests
// (e.g. a page load that triggers dozens of subresources on the same CDN)
// produces one upstream lookup, not dozens.
type Cache struct {
	inner   Resolver
	posTTL  time.Duration
	negTTL  time.Duration
	maxSize int

	mu      sync.Mutex
	entries map[string]*cacheEntry
	pending map[string]*pendingLookup
}

type cacheEntry struct {
	addrs   []net.IPAddr
	err     error
	expires time.Time
}

type pendingLookup struct {
	done  chan struct{}
	addrs []net.IPAddr
	err   error
}

func NewCache(inner Resolver, posTTL, negTTL time.Duration, maxSize int) *Cache {
	if maxSize <= 0 {
		maxSize = 1024
	}
	return &Cache{
		inner:   inner,
		posTTL:  posTTL,
		negTTL:  negTTL,
		maxSize: maxSize,
		entries: make(map[string]*cacheEntry),
		pending: make(map[string]*pendingLookup),
	}
}

func (c *Cache) String() string {
	return "cache(" + c.inner.String() + ")"
}

func cacheKey(host string, qTypes []uint16) string {
	b := make([]byte, len(host)+2*len(qTypes))
	copy(b, host)
	for i, q := range qTypes {
		b[len(host)+2*i] = byte(q >> 8)
		b[len(host)+2*i+1] = byte(q)
	}
	// b is freshly allocated and not referenced after the conversion, so a
	// zero-copy unsafe.String is safe here and avoids the runtime.stringbytes
	// copy that string(b) would do on every lookup.
	return unsafe.String(unsafe.SliceData(b), len(b))
}

func (c *Cache) Resolve(ctx context.Context, host string, qTypes []uint16) ([]net.IPAddr, error) {
	key := cacheKey(host, qTypes)
	now := time.Now()

	c.mu.Lock()
	if e, ok := c.entries[key]; ok && e.expires.After(now) {
		c.mu.Unlock()
		if e.err != nil {
			return nil, e.err
		}
		return cloneAddrs(e.addrs), nil
	}
	if p, ok := c.pending[key]; ok {
		c.mu.Unlock()
		select {
		case <-p.done:
			if p.err != nil {
				return nil, p.err
			}
			return cloneAddrs(p.addrs), nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	p := &pendingLookup{done: make(chan struct{})}
	c.pending[key] = p
	c.mu.Unlock()

	addrs, err := c.inner.Resolve(ctx, host, qTypes)

	c.mu.Lock()
	p.addrs, p.err = addrs, err
	close(p.done)
	delete(c.pending, key)

	ttl := c.posTTL
	if err != nil || len(addrs) == 0 {
		ttl = c.negTTL
	}
	if ttl > 0 {
		c.entries[key] = &cacheEntry{
			addrs:   cloneAddrs(addrs),
			err:     err,
			expires: time.Now().Add(ttl),
		}
		c.evictLocked()
	}
	c.mu.Unlock()

	return addrs, err
}

func (c *Cache) evictLocked() {
	if len(c.entries) <= c.maxSize {
		return
	}
	now := time.Now()
	for k, e := range c.entries {
		if !e.expires.After(now) {
			delete(c.entries, k)
		}
	}
	if len(c.entries) <= c.maxSize {
		return
	}
	// still too big — drop arbitrary entries. Map iteration order is
	// already randomized, so this gives us a cheap probabilistic eviction.
	excess := len(c.entries) - c.maxSize
	for k := range c.entries {
		if excess == 0 {
			break
		}
		delete(c.entries, k)
		excess--
	}
}

func cloneAddrs(in []net.IPAddr) []net.IPAddr {
	if in == nil {
		return nil
	}
	out := make([]net.IPAddr, len(in))
	copy(out, in)
	return out
}
