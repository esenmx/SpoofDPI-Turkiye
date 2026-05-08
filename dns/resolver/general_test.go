package resolver

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
)

// TestGeneralResolverRespectsContext starts a fake DNS server that never
// answers, to confirm the resolver honors context cancellation (the original
// bug was Client.Exchange being used instead of ExchangeContext, which
// silently ignored caller-side timeouts).
func TestGeneralResolverRespectsContext(t *testing.T) {
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close()

	r := NewGeneralResolver(pc.LocalAddr().String())

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err = r.Resolve(ctx, "example.com", []uint16{dns.TypeA})
	elapsed := time.Since(start)
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
	if elapsed > 600*time.Millisecond {
		t.Fatalf("resolver did not honor ctx; elapsed=%v", elapsed)
	}
}

// TestGeneralResolverFallsBackToTCPOnTruncation simulates an upstream that
// returns TC=1 over UDP (a common Turkish ISP DPI tactic) and serves a real
// answer over TCP. The resolver must take the TCP path automatically.
func TestGeneralResolverFallsBackToTCPOnTruncation(t *testing.T) {
	udp, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer udp.Close()

	tcp, err := net.Listen("tcp", udp.LocalAddr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer tcp.Close()

	go func() {
		buf := make([]byte, 1500)
		for {
			n, addr, err := udp.ReadFrom(buf)
			if err != nil {
				return
			}
			req := new(dns.Msg)
			if err := req.Unpack(buf[:n]); err != nil {
				continue
			}
			resp := new(dns.Msg)
			resp.SetReply(req)
			resp.Truncated = true
			b, _ := resp.Pack()
			udp.WriteTo(b, addr)
		}
	}()

	go func() {
		for {
			conn, err := tcp.Accept()
			if err != nil {
				return
			}
			go serveTCP(conn)
		}
	}()

	r := NewGeneralResolver(udp.LocalAddr().String())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	addrs, err := r.Resolve(ctx, "example.com", []uint16{dns.TypeA})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(addrs) != 1 || !addrs[0].IP.Equal(net.ParseIP("203.0.113.1")) {
		t.Fatalf("unexpected addrs: %v", addrs)
	}
}

func serveTCP(conn net.Conn) {
	defer conn.Close()
	srv := &dns.Server{
		Listener: &singleConnListener{conn: conn},
		Net:      "tcp",
		Handler: dns.HandlerFunc(func(w dns.ResponseWriter, m *dns.Msg) {
			resp := new(dns.Msg)
			resp.SetReply(m)
			resp.Answer = append(resp.Answer, &dns.A{
				Hdr: dns.RR_Header{Name: m.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
				A:   net.ParseIP("203.0.113.1"),
			})
			_ = w.WriteMsg(resp)
		}),
	}
	_ = srv.ActivateAndServe()
}

type singleConnListener struct {
	conn net.Conn
	done bool
}

func (l *singleConnListener) Accept() (net.Conn, error) {
	if l.done {
		return nil, net.ErrClosed
	}
	l.done = true
	return l.conn, nil
}
func (l *singleConnListener) Close() error   { return nil }
func (l *singleConnListener) Addr() net.Addr { return l.conn.LocalAddr() }
