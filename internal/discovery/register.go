package discovery

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// Register is the TCP register request/response payload (DUD-NET-111).
type Register struct {
	Type        string `json:"type"`
	PeerID      string `json:"peer_id"`
	DisplayName string `json:"display_name"`
	ProtoMajor  int    `json:"proto_major"`
	ProtoMinor  int    `json:"proto_minor"`
	TCPPort     int    `json:"tcp_port"`
	InstanceID  string `json:"instance_id"`
	Reason      string `json:"reason,omitempty"`
}

// EncodeRegister serializes a register line (newline-delimited JSON).
func EncodeRegister(r Register) ([]byte, error) {
	if r.Type == "" {
		r.Type = "register"
	}
	if r.Type != "register_reject" && r.ProtoMajor == 0 {
		r.ProtoMajor = DefaultProtoMajor
	}
	raw, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

// DecodeRegister parses one register JSON object (optional trailing newline).
func DecodeRegister(raw []byte) (Register, error) {
	var r Register
	if err := json.Unmarshal(raw, &r); err != nil {
		return Register{}, err
	}
	switch r.Type {
	case "", "register", "register_ok", "register_reject":
	default:
		return Register{}, fmt.Errorf("discovery: unexpected register type %q", r.Type)
	}
	if r.Type == "register_reject" {
		return r, nil
	}
	if r.PeerID == "" || r.InstanceID == "" {
		return Register{}, fmt.Errorf("discovery: missing peer_id/instance_id")
	}
	return r, nil
}

func writeRegister(w io.Writer, r Register) error {
	raw, err := EncodeRegister(r)
	if err != nil {
		return err
	}
	_, err = w.Write(raw)
	return err
}

func readRegister(r *bufio.Reader) (Register, error) {
	line, err := r.ReadBytes('\n')
	if err != nil && !(err == io.EOF && len(line) > 0) {
		return Register{}, err
	}
	return DecodeRegister(line)
}

func peerFromRegister(r Register, host string) Peer {
	return Peer{
		PeerID:      r.PeerID,
		DisplayName: r.DisplayName,
		InstanceID:  r.InstanceID,
		ProtoMajor:  r.ProtoMajor,
		ProtoMinor:  r.ProtoMinor,
		Host:        host,
		TCPPort:     r.TCPPort,
		LastSeen:    time.Now().UTC(),
	}
}

func (n *Node) rememberPeer(p Peer) {
	res := n.cfg.Peers.Upsert(p)
	if res.InstanceChanged {
		n.cfg.Logf("%s", FormatPeerUpdated(p.PeerID, res.OldInstanceID, p.InstanceID))
	}
	n.mu.Lock()
	onUpsert := n.cfg.OnPeerUpserted
	n.mu.Unlock()
	if onUpsert != nil {
		onUpsert(p, res)
	}
}

func hostFromAddr(addr net.Addr) string {
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return strings.TrimSpace(addr.String())
	}
	return host
}
