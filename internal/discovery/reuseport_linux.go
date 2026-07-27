//go:build linux

package discovery

import "syscall"

func setReuseAddrPort(fd int) error {
	if err := syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		return err
	}
	// Linux SO_REUSEPORT = 15; not exported by package syscall on linux.
	const soReusePort = 0xf
	return syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, soReusePort, 1)
}
