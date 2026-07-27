// Package discovery implements LAN peer discovery (UDP announce first; register later).
package discovery

import (
	"encoding/json"
	"fmt"
)

// DefaultUDPPort is the LAN announce/register port (DUD-NET-120 / ROADMAP P020).
const DefaultUDPPort = 41777

// DefaultProtoMajor / DefaultProtoMinor are the wire protocol version in announces.
const (
	DefaultProtoMajor = 1
	DefaultProtoMinor = 0
)

// Announce is the UDP broadcast payload (DUD-NET-111).
type Announce struct {
	Type        string `json:"type"`
	PeerID      string `json:"peer_id"`
	DisplayName string `json:"display_name"`
	ProtoMajor  int    `json:"proto_major"`
	ProtoMinor  int    `json:"proto_minor"`
	TCPPort     int    `json:"tcp_port"`
	InstanceID  string `json:"instance_id"`
	IsAgent     bool   `json:"is_agent,omitempty"` // DUD-AGT-120
}

// EncodeAnnounce serializes an announce datagram.
func EncodeAnnounce(a Announce) ([]byte, error) {
	a.Type = "announce"
	if a.ProtoMajor == 0 {
		a.ProtoMajor = DefaultProtoMajor
	}
	return json.Marshal(a)
}

// DecodeAnnounce parses an announce datagram.
func DecodeAnnounce(raw []byte) (Announce, error) {
	var a Announce
	if err := json.Unmarshal(raw, &a); err != nil {
		return Announce{}, err
	}
	if a.Type != "" && a.Type != "announce" {
		return Announce{}, fmt.Errorf("discovery: unexpected type %q", a.Type)
	}
	if a.PeerID == "" || a.InstanceID == "" {
		return Announce{}, fmt.Errorf("discovery: missing peer_id/instance_id")
	}
	return a, nil
}

// FormatAnnounceRx is the stdout/log line for a heard announce (ROADMAP P020).
func FormatAnnounceRx(a Announce, from string) string {
	return fmt.Sprintf(
		"announce_rx peer_id=%s name=%s from=%s instance_id=%s",
		a.PeerID, a.DisplayName, from, a.InstanceID,
	)
}
