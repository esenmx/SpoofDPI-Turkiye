package resolver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// ChainResolver runs every configured resolver in parallel and returns the
// first non-empty answer. This makes lookups robust to a single upstream
// being throttled, hijacked, or RST-injected — a common failure mode on
// Turkish ISPs (Türk Telekom, Turkcell, Vodafone) which selectively poison
// or drop UDP/53 to certain resolvers.
type ChainResolver struct {
	resolvers []Resolver
	perTry    time.Duration
}

func NewChainResolver(resolvers []Resolver, perTry time.Duration) *ChainResolver {
	return &ChainResolver{resolvers: resolvers, perTry: perTry}
}

func (c *ChainResolver) String() string {
	names := make([]string, len(c.resolvers))
	for i, r := range c.resolvers {
		names[i] = r.String()
	}
	return "chain[" + strings.Join(names, "|") + "]"
}

type chainResult struct {
	addrs []net.IPAddr
	err   error
	from  string
}

func (c *ChainResolver) Resolve(ctx context.Context, host string, qTypes []uint16) ([]net.IPAddr, error) {
	if len(c.resolvers) == 0 {
		return nil, errors.New("chain: no resolvers configured")
	}

	cctx, cancel := context.WithCancel(ctx)
	defer cancel()

	out := make(chan chainResult, len(c.resolvers))
	for _, r := range c.resolvers {
		go func(r Resolver) {
			tctx := cctx
			if c.perTry > 0 {
				var tcancel context.CancelFunc
				tctx, tcancel = context.WithTimeout(cctx, c.perTry)
				defer tcancel()
			}
			addrs, err := r.Resolve(tctx, host, qTypes)
			select {
			case out <- chainResult{addrs: addrs, err: err, from: r.String()}:
			case <-cctx.Done():
			}
		}(r)
	}

	var errs []error
	for i := 0; i < len(c.resolvers); i++ {
		select {
		case res := <-out:
			if len(res.addrs) > 0 {
				return res.addrs, nil
			}
			if res.err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", res.from, res.err))
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return nil, errors.Join(errs...)
}
