package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"net"
	"regexp"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/esenmx/SpoofDPI/packet"
	"github.com/esenmx/SpoofDPI/util"
	"github.com/esenmx/SpoofDPI/util/log"
)

const (
	httpsDialTimeout = 10 * time.Second
	defaultBufSize   = 4 * 1024
)

// bufPool reuses the per-direction read buffer in pipe(). Each accepted
// connection used to allocate a fresh 4 KiB slice for each direction.
var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, defaultBufSize)
		return &b
	},
}

type HttpsHandler struct {
	bufferSize      int
	protocol        string
	timeout         int
	windowSize      int
	exploit         bool
	allowedPatterns []*regexp.Regexp
}

func NewHttpsHandler(timeout int, windowSize int, allowedPatterns []*regexp.Regexp, exploit bool) *HttpsHandler {
	return &HttpsHandler{
		bufferSize:      defaultBufSize,
		protocol:        "HTTPS",
		timeout:         timeout,
		windowSize:      windowSize,
		allowedPatterns: allowedPatterns,
		exploit:         exploit,
	}
}

func (h *HttpsHandler) Serve(ctx context.Context, lConn *net.TCPConn, initPkt *packet.HttpRequest, ip string) {
	ctx = util.GetCtxWithScope(ctx, h.protocol)
	logger := log.GetCtxLogger(ctx)

	port := 443
	if initPkt.Port() != "" {
		parsed, err := strconv.Atoi(initPkt.Port())
		if err != nil || parsed <= 0 || parsed > 65535 {
			logger.Debug().Msgf("invalid port %q for %s, aborting", initPkt.Port(), initPkt.Domain())
			lConn.Close()
			return
		}
		port = parsed
	}

	rConn, err := dialTCP(ctx, ip, port)
	if err != nil {
		logger.Debug().Msgf("dial %s:%d: %s", ip, port, err)
		lConn.Close()
		return
	}

	logger.Debug().Msgf("new connection to the server %s -> %s", rConn.LocalAddr(), initPkt.Domain())

	if _, err := lConn.Write([]byte(initPkt.Version() + " 200 Connection Established\r\n\r\n")); err != nil {
		logger.Debug().Msgf("error sending 200 to client: %s", err)
		lConn.Close()
		rConn.Close()
		return
	}

	m, err := packet.ReadTLSMessage(lConn)
	if err != nil || !m.IsClientHello() {
		logger.Debug().Msgf("error reading client hello from %s: %v", lConn.RemoteAddr(), err)
		lConn.Close()
		rConn.Close()
		return
	}
	clientHello := m.Raw

	logger.Debug().Msgf("client sent hello %d bytes", len(clientHello))

	if h.exploit {
		logger.Debug().Msgf("writing chunked client hello to %s (window=%d)", initPkt.Domain(), h.windowSize)
		if _, err := writeChunks(rConn, splitInChunks(clientHello, h.windowSize)); err != nil {
			logger.Debug().Msgf("error writing chunked client hello to %s: %s", initPkt.Domain(), err)
			lConn.Close()
			rConn.Close()
			return
		}
	} else {
		logger.Debug().Msgf("writing plain client hello to %s", initPkt.Domain())
		if _, err := rConn.Write(clientHello); err != nil {
			logger.Debug().Msgf("error writing plain client hello to %s: %s", initPkt.Domain(), err)
			lConn.Close()
			rConn.Close()
			return
		}
	}

	pipe(ctx, lConn, rConn, h.bufferSize, h.timeout, initPkt.Domain())
}

// splitInChunks returns an iterator over fragments of raw. With size > 0 it
// emits fixed-size chunks (final remainder allowed); with size <= 0 it falls
// back to the legacy "1 byte then the rest" split that the original SpoofDPI
// shipped with.
func splitInChunks(raw []byte, size int) iter.Seq[[]byte] {
	if size <= 0 {
		return func(yield func([]byte) bool) {
			switch {
			case len(raw) == 0:
				return
			case len(raw) < 2:
				yield(raw)
			default:
				if !yield(raw[:1]) {
					return
				}
				yield(raw[1:])
			}
		}
	}
	return slices.Chunk(raw, size)
}

func writeChunks(conn *net.TCPConn, chunks iter.Seq[[]byte]) (int, error) {
	total, i := 0, 0
	for c := range chunks {
		i++
		n, err := conn.Write(c)
		total += n
		if err != nil {
			return total, fmt.Errorf("chunk %d: %w", i, err)
		}
	}
	return total, nil
}

// pipe runs both directions of the bidirectional copy and tears the
// connection down with TCP half-close so an EOF in one direction does not
// truncate an in-flight response in the other.
func pipe(ctx context.Context, lConn, rConn *net.TCPConn, bufSize, timeoutMs int, domain string) {
	logger := log.GetCtxLogger(ctx)

	var wg sync.WaitGroup
	wg.Go(func() {
		copyWithTimeout(ctx, lConn, rConn, bufSize, timeoutMs)
		_ = rConn.CloseWrite()
		_ = lConn.CloseRead()
	})
	wg.Go(func() {
		copyWithTimeout(ctx, rConn, lConn, bufSize, timeoutMs)
		_ = lConn.CloseWrite()
		_ = rConn.CloseRead()
	})

	wg.Wait()
	_ = lConn.Close()
	_ = rConn.Close()
	logger.Debug().Msgf("closed proxy connection: %s", domain)
}

func copyWithTimeout(ctx context.Context, from, to *net.TCPConn, bufSize, timeoutMs int) {
	logger := log.GetCtxLogger(ctx)

	var buf []byte
	var pooled *[]byte
	if bufSize == defaultBufSize {
		pooled = bufPool.Get().(*[]byte)
		defer bufPool.Put(pooled)
		buf = *pooled
	} else {
		buf = make([]byte, bufSize)
	}

	timeout := time.Duration(timeoutMs) * time.Millisecond
	for {
		if timeoutMs > 0 {
			if err := from.SetReadDeadline(time.Now().Add(timeout)); err != nil {
				logger.Debug().Msgf("setReadDeadline: %s", err)
				return
			}
		}
		n, err := from.Read(buf)
		if n > 0 {
			if _, werr := to.Write(buf[:n]); werr != nil {
				logger.Debug().Msgf("write: %s", werr)
				return
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				logger.Debug().Msgf("read: %s", err)
			}
			return
		}
	}
}

func dialTCP(ctx context.Context, ip string, port int) (*net.TCPConn, error) {
	d := &net.Dialer{Timeout: httpsDialTimeout, KeepAlive: 30 * time.Second}
	c, err := d.DialContext(ctx, "tcp", net.JoinHostPort(ip, strconv.Itoa(port)))
	if err != nil {
		return nil, err
	}
	tcp, ok := c.(*net.TCPConn)
	if !ok {
		c.Close()
		return nil, fmt.Errorf("expected *net.TCPConn, got %T", c)
	}
	return tcp, nil
}
