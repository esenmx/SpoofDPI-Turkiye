package resolver

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/miekg/dns"
)

type DOHResolver struct {
	upstream string
	host     string
	client   *http.Client
}

// NewDOHResolver builds a resolver that talks DNS-over-HTTPS to upstream.
// bootstrapIP, when non-empty, is dialed directly so the resolver does not
// depend on the system resolver to find its own server (avoids a chicken-
// and-egg problem on networks where the system resolver is poisoned).
func NewDOHResolver(upstream, bootstrapIP string) *DOHResolver {
	u, err := url.Parse(upstream)
	if err != nil || u.Host == "" {
		// Fall back to a single-host build so the call surface still works.
		u = &url.URL{Scheme: "https", Host: upstream, Path: "/dns-query"}
	}
	if u.Path == "" {
		u.Path = "/dns-query"
	}

	host := u.Hostname()
	dialer := &net.Dialer{Timeout: 3 * time.Second, KeepAlive: 30 * time.Second}

	transport := &http.Transport{
		ForceAttemptHTTP2:   true,
		MaxIdleConns:        4,
		MaxIdleConnsPerHost: 2,
		IdleConnTimeout:     30 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if bootstrapIP != "" {
				reqHost, reqPort, splitErr := net.SplitHostPort(addr)
				if splitErr == nil && reqHost == host {
					addr = net.JoinHostPort(bootstrapIP, reqPort)
				}
			}
			return dialer.DialContext(ctx, network, addr)
		},
	}

	return &DOHResolver{
		upstream: u.String(),
		host:     host,
		client:   &http.Client{Timeout: 5 * time.Second, Transport: transport},
	}
}

func (r *DOHResolver) Resolve(ctx context.Context, host string, qTypes []uint16) ([]net.IPAddr, error) {
	resultCh := lookupAllTypes(ctx, host, qTypes, r.exchange)
	return processResults(ctx, resultCh)
}

func (r *DOHResolver) String() string {
	return fmt.Sprintf("doh(%s)", r.upstream)
}

func (r *DOHResolver) exchange(ctx context.Context, msg *dns.Msg) (*dns.Msg, error) {
	pack, err := msg.Pack()
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s?dns=%s", r.upstream, base64.RawURLEncoding.EncodeToString(pack))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/dns-message")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer io.Copy(io.Discard, resp.Body)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("doh http %d", resp.StatusCode)
	}

	body := bytes.Buffer{}
	if _, err := io.Copy(&body, io.LimitReader(resp.Body, 64*1024)); err != nil {
		return nil, err
	}

	out := new(dns.Msg)
	if err := out.Unpack(body.Bytes()); err != nil {
		return nil, err
	}

	switch out.Rcode {
	case dns.RcodeSuccess:
		return out, nil
	case dns.RcodeNameError:
		return nil, errNXDomain
	default:
		return nil, fmt.Errorf("doh rcode %s", dns.RcodeToString[out.Rcode])
	}
}

var errNXDomain = errors.New("nxdomain")
