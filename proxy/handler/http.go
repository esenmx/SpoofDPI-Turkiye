package handler

import (
	"context"
	"io"
	"net"
	"strconv"

	"github.com/esenmx/SpoofDPI-Turkiye/packet"
	"github.com/esenmx/SpoofDPI-Turkiye/util/log"
)

type HttpHandler struct {
	bufferSize int
	protocol   string
	timeout    int
}

func NewHttpHandler(timeout int) *HttpHandler {
	return &HttpHandler{
		bufferSize: 4 * 1024,
		protocol:   "HTTP",
		timeout:    timeout,
	}
}

func (h *HttpHandler) Serve(ctx context.Context, lConn *net.TCPConn, lReader io.Reader, pkt *packet.HttpRequest, ip string) {
	ctx = log.GetCtxWithScope(ctx, h.protocol)
	logger := log.GetCtxLogger(ctx)

	port := 80
	if pkt.Port() != "" {
		parsed, err := strconv.Atoi(pkt.Port())
		if err != nil || parsed <= 0 || parsed > 65535 {
			logger.Debug().Msgf("invalid port %q for %s, aborting", pkt.Port(), pkt.Domain())
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

	logger.Debug().Msgf("new connection to the server %s -> %s", rConn.LocalAddr(), pkt.Domain())

	if _, err := rConn.Write(pkt.Raw()); err != nil {
		logger.Debug().Msgf("error sending request to %s: %s", pkt.Domain(), err)
		lConn.Close()
		rConn.Close()
		return
	}

	pipe(ctx, lReader, lConn, rConn, h.bufferSize, h.timeout, pkt.Domain())
}
