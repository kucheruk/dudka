// Package stunserver exposes the smallest useful RFC 5389 binding service.
// It reports a browser's UDP address and never relays application data.
package stunserver

import (
	"errors"
	"net"
	"time"

	"github.com/pion/stun/v3"
)

const (
	maxPacketSize       = 1500
	maxRequestsPerIPSec = 20
	maxIPsPerWindow     = 4096
)

// ListenAndServe answers unauthenticated STUN binding requests over UDP.
func ListenAndServe(address string) error {
	udpAddress, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", udpAddress)
	if err != nil {
		return err
	}
	defer conn.Close()
	return Serve(conn)
}

// Serve answers STUN binding requests on an existing UDP socket.
func Serve(conn *net.UDPConn) error {
	if conn == nil {
		return errors.New("nil UDP connection")
	}
	buffer := make([]byte, maxPacketSize)
	windowStart := time.Now()
	requests := make(map[string]int)

	for {
		count, remote, err := conn.ReadFromUDP(buffer)
		if err != nil {
			return err
		}
		now := time.Now()
		if now.Sub(windowStart) >= time.Second {
			windowStart = now
			clear(requests)
		}
		ipKey := remote.IP.String()
		if requests[ipKey] >= maxRequestsPerIPSec {
			continue
		}
		if _, known := requests[ipKey]; !known && len(requests) >= maxIPsPerWindow {
			continue
		}
		requests[ipKey]++

		raw := append([]byte(nil), buffer[:count]...)
		if !stun.IsMessage(raw) {
			continue
		}
		request := &stun.Message{Raw: raw}
		if err := request.Decode(); err != nil || request.Type != stun.BindingRequest {
			continue
		}
		mapped := &stun.XORMappedAddress{IP: remote.IP, Port: remote.Port}
		response, err := stun.Build(
			stun.NewTransactionIDSetter(request.TransactionID),
			stun.BindingSuccess,
			mapped,
			stun.Fingerprint,
		)
		if err != nil {
			continue
		}
		_, _ = conn.WriteToUDP(response.Raw, remote)
	}
}
