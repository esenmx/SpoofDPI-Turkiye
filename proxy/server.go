package proxy

import (
	"context"
	"errors"
	"io"
	"net"
	"time"

	"github.com/esenmx/SpoofDPI/util"
	"github.com/esenmx/SpoofDPI/util/log"
)

const (
	BufferSize   = 64 * 1024 // 64KB buffer for optimal throughput
	TLSHeaderLen = 5
)

func ReadBytes(ctx context.Context, conn *net.TCPConn, dest []byte) ([]byte, error) {
	n, err := readBytesInternal(ctx, conn, dest)
	return dest[:n], err
}

func readBytesInternal(ctx context.Context, conn *net.TCPConn, dest []byte) (int, error) {
	totalRead, err := conn.Read(dest)
	if err != nil {
		var opError *net.OpError
		switch {
		case errors.As(err, &opError) && opError.Timeout():
			return totalRead, errors.New("timed out")
		default:
			return totalRead, err
		}
	}
	return totalRead, nil
}

func Serve(ctx context.Context, from *net.TCPConn, to *net.TCPConn, proto string, fd string, td string, timeout int) {
	ctx = util.GetCtxWithScope(ctx, proto)
	logger := log.GetCtxLogger(ctx)

	defer func() {
		from.Close()
		to.Close()

		logger.Debug().Msgf("closing proxy connection: %s -> %s", fd, td)
	}()

	// Set read deadline once for the entire connection if timeout is set
	if timeout > 0 {
		deadline := time.Now().Add(time.Millisecond * time.Duration(timeout))
		from.SetReadDeadline(deadline)
	}

	// Use io.CopyBuffer for optimal performance with large buffer
	buf := make([]byte, BufferSize)
	_, err := io.CopyBuffer(to, from, buf)

	if err != nil && err != io.EOF {
		logger.Debug().Msgf("error in copy: %s -> %s: %s", fd, td, err)
	}
}
