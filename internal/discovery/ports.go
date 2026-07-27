package discovery

import (
	"errors"
	"fmt"
	"net"
	"syscall"
)

// DefaultPortSpan is how many ports above the preferred UDP/TCP port to try (P091).
const DefaultPortSpan = 10

// IsAddrInUse reports whether err is EADDRINUSE (or wrapped).
func IsAddrInUse(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, syscall.EADDRINUSE) {
		return true
	}
	var op *net.OpError
	if errors.As(err, &op) {
		return IsAddrInUse(op.Err)
	}
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return errno == syscall.EADDRINUSE
	}
	return false
}

// listenTCPWithFallback binds TCP on preferred port, then preferred+1…+span on EADDRINUSE.
func listenTCPWithFallback(preferred int, span int) (net.Listener, int, bool, error) {
	if preferred <= 0 {
		ln, err := net.Listen("tcp", ":0")
		if err != nil {
			return nil, 0, false, err
		}
		port := ln.Addr().(*net.TCPAddr).Port
		return ln, port, false, nil
	}
	if span <= 0 {
		span = DefaultPortSpan
	}
	var last error
	for i := 0; i <= span; i++ {
		port := preferred + i
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			return ln, port, i > 0, nil
		}
		last = err
		if !IsAddrInUse(err) {
			return nil, 0, false, err
		}
	}
	return nil, 0, false, fmt.Errorf("discovery: tcp ports %d-%d busy: %w", preferred, preferred+span, last)
}

// listenUDPWithFallback tries preferred then +1…+span when the first bind fails.
func listenUDPWithFallback(preferred int, span int) (net.PacketConn, int, bool, error) {
	if preferred <= 0 {
		conn, err := listenUDP(":0")
		if err != nil {
			return nil, 0, false, err
		}
		port := conn.LocalAddr().(*net.UDPAddr).Port
		return conn, port, false, nil
	}
	if span <= 0 {
		span = DefaultPortSpan
	}
	var last error
	for i := 0; i <= span; i++ {
		port := preferred + i
		conn, err := listenUDP(fmt.Sprintf(":%d", port))
		if err == nil {
			return conn, port, i > 0, nil
		}
		last = err
		if !IsAddrInUse(err) {
			return nil, 0, false, err
		}
	}
	return nil, 0, false, fmt.Errorf("discovery: udp ports %d-%d busy: %w", preferred, preferred+span, last)
}
