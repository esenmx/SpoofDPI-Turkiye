package resolver

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

type fakeResolver struct {
	name  string
	delay time.Duration
	addrs []net.IPAddr
	err   error
	calls int32
}

func (f *fakeResolver) String() string { return f.name }

func (f *fakeResolver) Resolve(ctx context.Context, _ string, _ []uint16) ([]net.IPAddr, error) {
	atomic.AddInt32(&f.calls, 1)
	select {
	case <-time.After(f.delay):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return f.addrs, f.err
}

func ip(s string) net.IPAddr { return net.IPAddr{IP: net.ParseIP(s)} }

func TestChainReturnsFastestSuccess(t *testing.T) {
	slow := &fakeResolver{name: "slow", delay: 200 * time.Millisecond, addrs: []net.IPAddr{ip("8.8.8.8")}}
	fast := &fakeResolver{name: "fast", delay: 10 * time.Millisecond, addrs: []net.IPAddr{ip("1.1.1.1")}}
	c := NewChainResolver([]Resolver{slow, fast}, time.Second)

	addrs, err := c.Resolve(context.Background(), "example.com", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(addrs) != 1 || !addrs[0].IP.Equal(net.ParseIP("1.1.1.1")) {
		t.Fatalf("expected fast resolver to win, got %v", addrs)
	}
}

func TestChainFallsBackOnFailure(t *testing.T) {
	bad := &fakeResolver{name: "bad", delay: 5 * time.Millisecond, err: errors.New("hijacked")}
	good := &fakeResolver{name: "good", delay: 10 * time.Millisecond, addrs: []net.IPAddr{ip("9.9.9.9")}}
	c := NewChainResolver([]Resolver{bad, good}, time.Second)

	addrs, err := c.Resolve(context.Background(), "example.com", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(addrs) != 1 || !addrs[0].IP.Equal(net.ParseIP("9.9.9.9")) {
		t.Fatalf("expected fallback resolver to win, got %v", addrs)
	}
}

func TestChainAggregatesErrorsWhenAllFail(t *testing.T) {
	a := &fakeResolver{name: "a", err: errors.New("boom-a")}
	b := &fakeResolver{name: "b", err: errors.New("boom-b")}
	c := NewChainResolver([]Resolver{a, b}, time.Second)

	addrs, err := c.Resolve(context.Background(), "example.com", nil)
	if err == nil || addrs != nil {
		t.Fatalf("expected error and nil addrs, got %v %v", addrs, err)
	}
	if !contains(err.Error(), "boom-a") || !contains(err.Error(), "boom-b") {
		t.Fatalf("expected aggregated errors, got %v", err)
	}
}

func TestChainEmptyResultIsTreatedAsFailure(t *testing.T) {
	empty := &fakeResolver{name: "empty"}
	good := &fakeResolver{name: "good", delay: 5 * time.Millisecond, addrs: []net.IPAddr{ip("1.0.0.1")}}
	c := NewChainResolver([]Resolver{empty, good}, time.Second)

	addrs, err := c.Resolve(context.Background(), "example.com", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(addrs) != 1 || !addrs[0].IP.Equal(net.ParseIP("1.0.0.1")) {
		t.Fatalf("expected good resolver to win after empty result, got %v", addrs)
	}
}

func TestChainPerTryTimeoutCancelsSlow(t *testing.T) {
	stuck := &fakeResolver{name: "stuck", delay: time.Hour}
	good := &fakeResolver{name: "good", delay: 10 * time.Millisecond, addrs: []net.IPAddr{ip("1.0.0.1")}}
	c := NewChainResolver([]Resolver{stuck, good}, 50*time.Millisecond)

	start := time.Now()
	addrs, err := c.Resolve(context.Background(), "example.com", nil)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(addrs) != 1 {
		t.Fatalf("expected 1 addr, got %v", addrs)
	}
	if elapsed > 200*time.Millisecond {
		t.Fatalf("expected fast resolution, elapsed=%v", elapsed)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
