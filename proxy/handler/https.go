package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/esenmx/SpoofDPI/packet"
	"github.com/esenmx/SpoofDPI/util"
	"github.com/esenmx/SpoofDPI/util/log"
)

const (
	httpsDialTimeout = 10 * time.Second
)

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
		bufferSize:      4 * 1024,
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
		logger.Debug().Msgf("writing chunked client hello to %s", initPkt.Domain())
		chunks := splitInChunks(ctx, clientHello, h.windowSize)
		if _, err := writeChunks(rConn, chunks); err != nil {
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

func splitInChunks(ctx context.Context, raw []byte, size int) [][]byte {
	logger := log.GetCtxLogger(ctx)
	logger.Debug().Msgf("window-size: %d", size)

	if size <= 0 {
		if len(raw) < 2 {
			return [][]byte{raw}
		}
		logger.Debug().Msg("using legacy fragmentation")
		return [][]byte{raw[:1], raw[1:]}
	}

	chunks := make([][]byte, 0, (len(raw)+size-1)/size)
	for len(raw) > 0 {
		n := size
		if n > len(raw) {
			n = len(raw)
		}
		chunks = append(chunks, raw[:n])
		raw = raw[n:]
	}
	return chunks
}

func writeChunks(conn *net.TCPConn, chunks [][]byte) (int, error) {
	total := 0
	for i, c := range chunks {
		n, err := conn.Write(c)
		total += n
		if err != nil {
			return total, fmt.Errorf("chunk %d/%d: %w", i+1, len(chunks), err)
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
	wg.Add(2)

	go func() {
		defer wg.Done()
		copyWithTimeout(ctx, lConn, rConn, bufSize, timeoutMs)
		_ = rConn.CloseWrite()
		_ = lConn.CloseRead()
	}()
	go func() {
		defer wg.Done()
		copyWithTimeout(ctx, rConn, lConn, bufSize, timeoutMs)
		_ = lConn.CloseWrite()
		_ = rConn.CloseRead()
	}()

	wg.Wait()
	_ = lConn.Close()
	_ = rConn.Close()
	logger.Debug().Msgf("closed proxy connection: %s", domain)
}

func copyWithTimeout(ctx context.Context, from, to *net.TCPConn, bufSize, timeoutMs int) {
	logger := log.GetCtxLogger(ctx)
	buf := make([]byte, bufSize)
	for {
		if timeoutMs > 0 {
			if err := from.SetReadDeadline(time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)); err != nil {
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
