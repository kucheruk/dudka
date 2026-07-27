package discovery

import (
	"context"
	"fmt"
	"net"
	"syscall"
)

func listenUDP(addr string) (net.PacketConn, error) {
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var opErr error
			if err := c.Control(func(fd uintptr) {
				// SO_REUSEPORT (where available) lets two dudkad on one host share the announce port.
				opErr = setReuseAddrPort(fd)
			}); err != nil {
				return err
			}
			return opErr
		},
	}
	conn, err := lc.ListenPacket(context.Background(), "udp4", addr)
	if err != nil {
		return nil, fmt.Errorf("discovery: listen %s: %w", addr, err)
	}
	return conn, nil
}

func setBroadcast(conn net.PacketConn) error {
	sc, ok := conn.(syscall.Conn)
	if !ok {
		return nil
	}
	raw, err := sc.SyscallConn()
	if err != nil {
		return err
	}
	var opErr error
	if err := raw.Control(func(fd uintptr) {
		opErr = setSockBroadcast(fd)
	}); err != nil {
		return err
	}
	return opErr
}
