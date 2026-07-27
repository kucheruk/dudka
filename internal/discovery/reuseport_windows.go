//go:build windows

package discovery

import "syscall"

func setReuseAddrPort(fd uintptr) error {
	// Windows: REUSEADDR only (no portable SO_REUSEPORT in Go syscall).
	return syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
}

func setSockBroadcast(fd uintptr) error {
	return syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_BROADCAST, 1)
}
