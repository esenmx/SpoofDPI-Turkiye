package resolver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/esenmx/SpoofDPI/dns/addrselect"
	"github.com/miekg/dns"
)

type Resolver interface {
	Resolve(ctx context.Context, host string, qTypes []uint16) ([]net.IPAddr, error)
	String() string
}

type exchangeFunc = func(ctx context.Context, msg *dns.Msg) (*dns.Msg, error)

type DNSResult struct {
	msg *dns.Msg
	err error
}

func recordTypeIDToName(id uint16) string {
	switch id {
	case dns.TypeA:
		return "A"
	case dns.TypeAAAA:
		return "AAAA"
	}
	return strconv.FormatUint(uint64(id), 10)
}

func parseAddrsFromMsg(msg *dns.Msg) []net.IPAddr {
	if msg == nil {
		return nil
	}
	var addrs []net.IPAddr
	for _, record := range msg.Answer {
		switch ipRecord := record.(type) {
		case *dns.A:
			addrs = append(addrs, net.IPAddr{IP: ipRecord.A})
		case *dns.AAAA:
			addrs = append(addrs, net.IPAddr{IP: ipRecord.AAAA})
		}
	}
	return addrs
}

func sortAddrs(addrs []net.IPAddr) {
	addrselect.SortByRFC6724(addrs)
}

func lookupAllTypes(ctx context.Context, host string, qTypes []uint16, exchange exchangeFunc) <-chan *DNSResult {
	var wg sync.WaitGroup
	resCh := make(chan *DNSResult, len(qTypes))

	for _, qType := range qTypes {
		wg.Add(1)
		go func(qType uint16) {
			defer wg.Done()
			res := lookupType(ctx, host, qType, exchange)
			select {
			case resCh <- res:
			case <-ctx.Done():
			}
		}(qType)
	}

	go func() {
		wg.Wait()
		close(resCh)
	}()

	return resCh
}

func lookupType(ctx context.Context, host string, queryType uint16, exchange exchangeFunc) *DNSResult {
	msg := newMsg(host, queryType)
	resp, err := exchange(ctx, msg)
	if err != nil {
		return &DNSResult{err: fmt.Errorf("%s %s: %w", recordTypeIDToName(queryType), host, err)}
	}
	return &DNSResult{msg: resp}
}

func newMsg(host string, qType uint16) *dns.Msg {
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(host), qType)
	msg.RecursionDesired = true
	return msg
}

func processResults(ctx context.Context, resCh <-chan *DNSResult) ([]net.IPAddr, error) {
	var errs []error
	var addrs []net.IPAddr

	for {
		select {
		case res, ok := <-resCh:
			if !ok {
				if len(addrs) == 0 {
					return nil, errors.Join(errs...)
				}
				if len(addrs) > 1 {
					sortAddrs(addrs)
				}
				return addrs, nil
			}
			if res.err != nil {
				errs = append(errs, res.err)
				continue
			}
			addrs = append(addrs, parseAddrsFromMsg(res.msg)...)
		case <-ctx.Done():
			if len(addrs) > 0 {
				if len(addrs) > 1 {
					sortAddrs(addrs)
				}
				return addrs, nil
			}
			return nil, ctx.Err()
		}
	}
}
