package proxy

import (
	"context"
	"errors"
	"net"
	"regexp"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/esenmx/SpoofDPI/dns"
	"github.com/esenmx/SpoofDPI/packet"
	"github.com/esenmx/SpoofDPI/proxy/handler"
	"github.com/esenmx/SpoofDPI/util"
	"github.com/esenmx/SpoofDPI/util/log"
)

const scopeProxy = "PROXY"

type Proxy struct {
	addr           string
	port           int
	timeout        int
	resolver       *dns.Dns
	windowSize     int
	allowedPattern []*regexp.Regexp
}

type Handler interface {
	Serve(ctx context.Context, lConn *net.TCPConn, pkt *packet.HttpRequest, ip string)
}

func New(config *util.Config) *Proxy {
	return &Proxy{
		addr:           config.Addr,
		port:           config.Port,
		timeout:        config.Timeout,
		windowSize:     config.WindowSize,
		allowedPattern: config.AllowedPatterns,
		resolver:       dns.NewDns(config),
	}
}

func (pxy *Proxy) Start(ctx context.Context) error {
	ctx = util.GetCtxWithScope(ctx, scopeProxy)
	logger := log.GetCtxLogger(ctx)

	l, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP(pxy.addr), Port: pxy.port})
	if err != nil {
		return err
	}
	defer l.Close()

	if pxy.timeout > 0 {
		logger.Info().Msgf("connection timeout is set to %d ms", pxy.timeout)
	}
	logger.Info().Msgf("created a listener on port %d", pxy.port)
	if len(pxy.allowedPattern) > 0 {
		logger.Info().Msgf("number of white-listed pattern: %d", len(pxy.allowedPattern))
	}

	go func() {
		<-ctx.Done()
		_ = l.Close()
	}()

	var tempDelay time.Duration
	for {
		conn, err := l.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := time.Second; tempDelay > max {
					tempDelay = max
				}
				logger.Warn().Msgf("transient accept error: %s; retrying in %s", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			logger.Error().Msgf("accept: %s", err)
			return err
		}
		tempDelay = 0

		go pxy.handle(ctx, conn)
	}
}

func (pxy *Proxy) handle(ctx context.Context, conn net.Conn) {
	ctx = util.GetCtxWithTraceId(ctx)
	logger := log.GetCtxLogger(ctx)

	defer func() {
		if r := recover(); r != nil {
			logger.Error().Msgf("panic in connection handler: %v\n%s", r, debug.Stack())
			_ = conn.Close()
		}
	}()

	pkt, err := packet.ReadHttpRequest(conn)
	if err != nil {
		logger.Debug().Msgf("error parsing request: %s", err)
		_ = conn.Close()
		return
	}
	pkt.Tidy()

	logger.Debug().Msgf("request from %s for %s", conn.RemoteAddr(), pkt.Domain())

	if !pkt.IsValidMethod() {
		logger.Debug().Msgf("unsupported method: %s", pkt.Method())
		_ = conn.Close()
		return
	}

	matched := pxy.patternMatches([]byte(pkt.Domain()))
	useSystemDns := !matched

	ip, err := pxy.resolver.ResolveHost(ctx, pkt.Domain(), false, useSystemDns)
	if err != nil {
		logger.Debug().Msgf("dns lookup %s: %s", pkt.Domain(), err)
		_, _ = conn.Write([]byte(pkt.Version() + " 502 Bad Gateway\r\n\r\n"))
		_ = conn.Close()
		return
	}

	if pkt.Port() == strconv.Itoa(pxy.port) && isLoopedRequest(ctx, net.ParseIP(ip)) {
		logger.Error().Msg("looped request detected, aborting")
		_ = conn.Close()
		return
	}

	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		logger.Error().Msgf("expected *net.TCPConn, got %T", conn)
		_ = conn.Close()
		return
	}

	var h Handler
	if pkt.IsConnectMethod() {
		h = handler.NewHttpsHandler(pxy.timeout, pxy.windowSize, pxy.allowedPattern, matched)
	} else {
		h = handler.NewHttpHandler(pxy.timeout)
	}
	h.Serve(ctx, tcpConn, pkt, ip)
}

func (pxy *Proxy) patternMatches(b []byte) bool {
	if pxy.allowedPattern == nil {
		return true
	}
	for _, pattern := range pxy.allowedPattern {
		if pattern.Match(b) {
			return true
		}
	}
	return false
}

func isLoopedRequest(ctx context.Context, ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() {
		return true
	}
	logger := log.GetCtxLogger(ctx)
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		logger.Error().Msgf("interface addrs: %s", err)
		return false
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && ipnet.IP.Equal(ip) {
			return true
		}
	}
	return false
}
