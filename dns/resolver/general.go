package resolver

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/miekg/dns"
)

type GeneralResolver struct {
	udp    *dns.Client
	tcp    *dns.Client
	server string
}

func NewGeneralResolver(server string) *GeneralResolver {
	return &GeneralResolver{
		udp:    &dns.Client{Net: "udp", Timeout: 2 * time.Second},
		tcp:    &dns.Client{Net: "tcp", Timeout: 3 * time.Second},
		server: server,
	}
}

func (r *GeneralResolver) Resolve(ctx context.Context, host string, qTypes []uint16) ([]net.IPAddr, error) {
	resultCh := lookupAllTypes(ctx, host, qTypes, r.exchange)
	return processResults(ctx, resultCh)
}

func (r *GeneralResolver) String() string {
	return fmt.Sprintf("general(%s)", r.server)
}

func (r *GeneralResolver) exchange(ctx context.Context, msg *dns.Msg) (*dns.Msg, error) {
	resp, _, err := r.udp.ExchangeContext(ctx, msg, r.server)
	if err != nil {
		return nil, err
	}
	if resp != nil && resp.Truncated {
		resp, _, err = r.tcp.ExchangeContext(ctx, msg, r.server)
		if err != nil {
			return nil, err
		}
	}
	return resp, nil
}
