package resolver

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/miekg/dns"
)

func TestCacheServesPositiveHits(t *testing.T) {
	inner := &fakeResolver{name: "inner", addrs: []net.IPAddr{ip("1.1.1.1")}}
	c := NewCache(inner, time.Minute, 10*time.Second, 16)

	for i := 0; i < 5; i++ {
		addrs, err := c.Resolve(context.Background(), "example.com", []uint16{dns.TypeA})
		if err != nil {
			t.Fatalf("iter %d: %v", i, err)
		}
		if len(addrs) != 1 {
			t.Fatalf("iter %d: got %v", i, addrs)
		}
	}
	if got := atomic.LoadInt32(&inner.calls); got != 1 {
		t.Fatalf("expected 1 upstream call, got %d", got)
	}
}

func TestCacheServesNegativeHits(t *testing.T) {
	inner := &fakeResolver{name: "inner", err: errors.New("nxdomain")}
	c := NewCache(inner, time.Minute, time.Minute, 16)

	for i := 0; i < 3; i++ {
		_, err := c.Resolve(context.Background(), "missing.tld", []uint16{dns.TypeA})
		if err == nil {
			t.Fatalf("expected error on iter %d", i)
		}
	}
	if got := atomic.LoadInt32(&inner.calls); got != 1 {
		t.Fatalf("expected 1 upstream call (negative cache hit thereafter), got %d", got)
	}
}

func TestCacheCoalescesConcurrentLookups(t *testing.T) {
	inner := &fakeResolver{name: "inner", delay: 80 * time.Millisecond, addrs: []net.IPAddr{ip("1.1.1.1")}}
	c := NewCache(inner, time.Minute, 10*time.Second, 16)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.Resolve(context.Background(), "example.com", []uint16{dns.TypeA}); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	if got := atomic.LoadInt32(&inner.calls); got != 1 {
		t.Fatalf("expected exactly 1 upstream call (coalesced), got %d", got)
	}
}

func TestCacheReturnsFreshAfterTTL(t *testing.T) {
	inner := &fakeResolver{name: "inner", addrs: []net.IPAddr{ip("1.1.1.1")}}
	c := NewCache(inner, 30*time.Millisecond, 30*time.Millisecond, 16)

	if _, err := c.Resolve(context.Background(), "example.com", []uint16{dns.TypeA}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(60 * time.Millisecond)
	if _, err := c.Resolve(context.Background(), "example.com", []uint16{dns.TypeA}); err != nil {
		t.Fatal(err)
	}
	if got := atomic.LoadInt32(&inner.calls); got != 2 {
		t.Fatalf("expected 2 upstream calls after TTL expiry, got %d", got)
	}
}

func TestCacheKeyDistinguishesQTypes(t *testing.T) {
	inner := &fakeResolver{name: "inner", addrs: []net.IPAddr{ip("1.1.1.1")}}
	c := NewCache(inner, time.Minute, 10*time.Second, 16)

	if _, err := c.Resolve(context.Background(), "example.com", []uint16{dns.TypeA}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Resolve(context.Background(), "example.com", []uint16{dns.TypeAAAA}); err != nil {
		t.Fatal(err)
	}
	if got := atomic.LoadInt32(&inner.calls); got != 2 {
		t.Fatalf("expected separate cache entries for A vs AAAA, got %d calls", got)
	}
}
