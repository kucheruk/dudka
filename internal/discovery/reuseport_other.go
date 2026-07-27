//go:build !linux && !darwin

package discovery

import "syscall"

func setReuseAddrPort(fd int) error {
	// Best-effort: REUSEADDR only (SO_REUSEPORT not portable here).
	return syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
}
