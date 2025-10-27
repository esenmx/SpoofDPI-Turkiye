package handler

import (
	"net"
	"syscall"
	"time"
)

// ConfigureTCP sets TCP options for optimal performance
func ConfigureTCP(conn *net.TCPConn) error {
	// Enable TCP_NODELAY to disable Nagle's algorithm (reduce latency)
	err := conn.SetNoDelay(true)
	if err != nil {
		return err
	}

	// Try to set TCP_QUICKACK on Linux for faster ACKs (12 = TCP_QUICKACK constant)
	if rawConn, err := conn.SyscallConn(); err == nil {
		rawConn.Write(func(fd uintptr) bool {
			// TCP_QUICKACK is Linux-specific, ignore errors on other platforms
			_ = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_TCP, 12, 1)
			return true
		})
	}

	// Set TCP keepalive for connection health
	conn.SetKeepAlive(true)
	conn.SetKeepAlivePeriod(30 * time.Second)

	return nil
}

func setConnectionTimeout(conn *net.TCPConn, timeout int) error {
	if timeout <= 0 {
		return nil
	}

	return conn.SetReadDeadline(time.Now().Add(time.Millisecond * time.Duration(timeout)))
}
