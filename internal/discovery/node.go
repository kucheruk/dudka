package discovery

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

// DialFunc opens a TCP connection; injectable for DUD-NET-101 tests (fake dialer).
type DialFunc func(network, address string, timeout time.Duration) (net.Conn, error)

// Config configures a discovery node (announce + TCP register).
type Config struct {
	PeerID      string
	DisplayName string
	InstanceID  string
	ProtoMajor  int
	ProtoMinor  int
	TCPPort     int           // advertised/listen TCP port; 0 with TCPBind :0 => ephemeral
	UDPPort     int           // used when Bind empty; default DefaultUDPPort
	Bind        string        // UDP bind, e.g. 127.0.0.1:0 or :41777
	TCPBind     string        // TCP bind; empty => :TCPPort or :UDPPort
	Target      string        // empty => UDP broadcast to 255.255.255.255:UDPPort
	Interval    time.Duration // default 2s
	PeerTTL     time.Duration // prune peers quieter than this; default 5*Interval (P034)
	Peers       *PeerStore
	OnAnnounce  func(Announce, net.Addr)
	Logf        func(format string, args ...any)
	DialTimeout time.Duration
	Dialer      DialFunc // default: net.DialTimeout
	DialHosts   []string // optional seed hosts from config; dialed after Start (LAN-only)
	// OnChatLine handles inbound NDJSON feed lines: "chat" (P030) and "file_announce" (P050).
	OnChatLine func(host string, line []byte)
	// OnTailRequest handles inbound type "tail_req" (P033); write response on conn.
	OnTailRequest func(host string, conn net.Conn)
	// OnFileChunkRequest handles inbound type "file_chunk_req" (P051); write chunks on conn.
	OnFileChunkRequest func(host string, conn net.Conn, line []byte)
	// OnPeerUpserted fires after the peer table changes (register / dial).
	OnPeerUpserted func(Peer, UpsertResult)
	// OnPeerRemoved fires when a peer is pruned after PeerTTL (P034).
	OnPeerRemoved func(Peer)
}

// Node sends periodic announces, accepts TCP register, and dials peers on announce.
type Node struct {
	cfg     Config
	mu      sync.Mutex
	conn    net.PacketConn
	tcpLn   net.Listener
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	local         net.Addr
	dialing       map[string]struct{}
	tcpPort       int
	udpPort       int
	portRelocated bool
	portNote      string
	proto         *protoBook
}

// NewNode builds a stopped discovery node.
func NewNode(cfg Config) *Node {
	if cfg.ProtoMajor == 0 {
		cfg.ProtoMajor = DefaultProtoMajor
	}
	if cfg.UDPPort == 0 && cfg.Bind == "" {
		cfg.UDPPort = DefaultUDPPort
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 2 * time.Second
	}
	if cfg.PeerTTL <= 0 {
		cfg.PeerTTL = 5 * cfg.Interval
	}
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = 2 * time.Second
	}
	if cfg.Dialer == nil {
		cfg.Dialer = net.DialTimeout
	}
	if cfg.Logf == nil {
		cfg.Logf = func(string, ...any) {}
	}
	if cfg.Peers == nil {
		cfg.Peers = NewPeerStore()
	}
	return &Node{cfg: cfg, dialing: make(map[string]struct{}), proto: newProtoBook()}
}

// LocalAddr returns the bound UDP address after Start.
func (n *Node) LocalAddr() net.Addr {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.local
}

// TCPPort returns the bound TCP register port after Start.
func (n *Node) TCPPort() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.tcpPort
}

// Peers returns the neighbor table.
func (n *Node) Peers() *PeerStore { return n.cfg.Peers }

// Status returns proto health including recent incompatible peers (P023)
// and LAN availability (P044 / DUD-NET-140).
func (n *Node) Status() Status {
	n.mu.Lock()
	major, minor := n.cfg.ProtoMajor, n.cfg.ProtoMinor
	ann, sess := n.udpPort, n.tcpPort
	reloc := n.portRelocated
	note := n.portNote
	n.mu.Unlock()
	if major == 0 {
		major = DefaultProtoMajor
	}
	network := NetworkOK
	if !HasUsableLAN() {
		network = NetworkNoNetwork
	}
	return Status{
		ProtoMajor:    major,
		ProtoMinor:    minor,
		Network:       network,
		Incompatible:  n.proto.list(),
		AnnouncePort:  ann,
		SessionPort:   sess,
		PortRelocated: reloc,
		PortNote:      note,
	}
}

func (n *Node) noteProtoMismatch(peerID string, theirs int) {
	n.mu.Lock()
	ours := n.cfg.ProtoMajor
	n.mu.Unlock()
	if ours == 0 {
		ours = DefaultProtoMajor
	}
	n.proto.note(peerID, theirs)
	n.cfg.Logf("%s", FormatProtoMismatch(peerID, theirs, ours))
}

// SetTarget updates the UDP announce destination (tests).
func (n *Node) SetTarget(target string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.cfg.Target = target
}

// SetDisplayName updates the nick used in announce/register (DUD-CHAT-111 / P043).
func (n *Node) SetDisplayName(name string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.cfg.DisplayName = strings.TrimSpace(name)
}

// Start binds TCP+UDP, begins announce/register loops, then dials DialHosts (LAN-only).
func (n *Node) Start() error {
	n.mu.Lock()
	if n.conn != nil {
		n.mu.Unlock()
		return errors.New("discovery: already started")
	}
	if n.cfg.PeerID == "" || n.cfg.InstanceID == "" {
		n.mu.Unlock()
		return errors.New("discovery: peer_id and instance_id required")
	}

	var (
		tcpLn        net.Listener
		tcpPort      int
		tcpRelocated bool
		err          error
	)
	if n.cfg.TCPBind != "" {
		tcpLn, err = net.Listen("tcp", n.cfg.TCPBind)
		if err != nil {
			n.mu.Unlock()
			return fmt.Errorf("discovery: tcp listen: %w", err)
		}
		tcpPort = tcpLn.Addr().(*net.TCPAddr).Port
	} else {
		tcpLn, tcpPort, tcpRelocated, err = listenTCPWithFallback(n.cfg.TCPPort, DefaultPortSpan)
		if err != nil {
			n.mu.Unlock()
			return fmt.Errorf("discovery: tcp listen: %w", err)
		}
	}
	n.tcpPort = tcpPort
	n.cfg.TCPPort = tcpPort

	var (
		conn         net.PacketConn
		udpPort      int
		udpRelocated bool
	)
	if n.cfg.Bind != "" {
		conn, err = listenUDP(n.cfg.Bind)
		if err != nil {
			_ = tcpLn.Close()
			n.mu.Unlock()
			return err
		}
		if ua, ok := conn.LocalAddr().(*net.UDPAddr); ok {
			udpPort = ua.Port
		}
	} else {
		preferredUDP := n.cfg.UDPPort
		if preferredUDP == 0 {
			preferredUDP = DefaultUDPPort
		}
		conn, udpPort, udpRelocated, err = listenUDPWithFallback(preferredUDP, DefaultPortSpan)
		if err != nil {
			_ = tcpLn.Close()
			n.mu.Unlock()
			return err
		}
		// Keep cfg.UDPPort as the announce *destination* port (default 41777).
		// Listen may relocate; status exposes actual bind via udpPort (P091).
	}
	if err := setBroadcast(conn); err != nil {
		_ = conn.Close()
		_ = tcpLn.Close()
		n.mu.Unlock()
		return err
	}

	n.udpPort = udpPort
	n.portRelocated = tcpRelocated || udpRelocated
	if n.portRelocated {
		n.portNote = fmt.Sprintf("порт %d занят — слушаем announce=%d session=%d", DefaultUDPPort, udpPort, tcpPort)
		n.cfg.Logf("port_relocated announce=%d session=%d note=%q", udpPort, tcpPort, n.portNote)
	}

	ctx, cancel := context.WithCancel(context.Background())
	n.conn = conn
	n.tcpLn = tcpLn
	n.cancel = cancel
	n.local = conn.LocalAddr()
	seeds := append([]string{}, n.cfg.DialHosts...)

	n.wg.Add(4)
	go n.readLoop(ctx, conn)
	go n.announceLoop(ctx)
	go n.acceptLoop(ctx, tcpLn)
	go n.pruneLoop(ctx)
	n.mu.Unlock()

	n.dialConfiguredHosts(seeds)
	return nil
}

func (n *Node) pruneLoop(ctx context.Context) {
	defer n.wg.Done()
	n.mu.Lock()
	ttl := n.cfg.PeerTTL
	interval := n.cfg.Interval
	n.mu.Unlock()
	if ttl <= 0 {
		return
	}
	tick := ttl / 2
	if tick <= 0 {
		tick = interval
	}
	if tick > time.Second {
		tick = time.Second
	}
	t := time.NewTicker(tick)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			n.pruneStalePeers()
		}
	}
}

func (n *Node) pruneStalePeers() {
	n.mu.Lock()
	ttl := n.cfg.PeerTTL
	peers := n.cfg.Peers
	onRemoved := n.cfg.OnPeerRemoved
	n.mu.Unlock()
	if ttl <= 0 || peers == nil {
		return
	}
	removed := peers.PruneOlderThan(time.Now().UTC().Add(-ttl))
	for _, p := range removed {
		n.cfg.Logf("%s", FormatPeerGone(p.PeerID))
		if onRemoved != nil {
			onRemoved(p)
		}
	}
}

func (n *Node) dialConfiguredHosts(hosts []string) {
	n.mu.Lock()
	port := n.tcpPort
	if port == 0 {
		port = n.cfg.TCPPort
	}
	n.mu.Unlock()
	if port <= 0 {
		port = DefaultUDPPort
	}
	for _, host := range hosts {
		host = strings.TrimSpace(host)
		if host == "" {
			continue
		}
		_, _ = n.dialRegister(host, port)
	}
}

// Stop closes sockets and waits for goroutines.
func (n *Node) Stop() error {
	n.mu.Lock()
	cancel := n.cancel
	conn := n.conn
	tcpLn := n.tcpLn
	n.cancel = nil
	n.conn = nil
	n.tcpLn = nil
	n.local = nil
	n.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	var err error
	if conn != nil {
		err = conn.Close()
	}
	if tcpLn != nil {
		if e := tcpLn.Close(); e != nil && err == nil {
			err = e
		}
	}
	n.wg.Wait()
	return err
}

func (n *Node) acceptLoop(ctx context.Context, ln net.Listener) {
	defer n.wg.Done()
	for {
		conn, err := ln.Accept()
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}
		go n.handleSessionConn(conn)
	}
}

func (n *Node) handleSessionConn(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(n.cfg.DialTimeout))
	br := bufio.NewReader(conn)
	line, err := br.ReadBytes('\n')
	if err != nil && !(err == io.EOF && len(line) > 0) {
		return
	}
	typ := peekJSONType(line)
	host := hostFromAddr(conn.RemoteAddr())
	switch typ {
	case "chat", "file_announce":
		n.mu.Lock()
		onChat := n.cfg.OnChatLine
		n.mu.Unlock()
		if onChat != nil {
			onChat(host, line)
		}
		return
	case "tail_req":
		n.mu.Lock()
		onTail := n.cfg.OnTailRequest
		n.mu.Unlock()
		if onTail != nil {
			onTail(host, conn)
		}
		return
	case "file_chunk_req":
		n.mu.Lock()
		onFile := n.cfg.OnFileChunkRequest
		n.mu.Unlock()
		if onFile != nil {
			onFile(host, conn, line)
		}
		return
	}

	req, err := DecodeRegister(line)
	if err != nil {
		return
	}

	n.mu.Lock()
	oursMajor := n.cfg.ProtoMajor
	self := Register{
		Type:        "register_ok",
		PeerID:      n.cfg.PeerID,
		DisplayName: n.cfg.DisplayName,
		ProtoMajor:  n.cfg.ProtoMajor,
		ProtoMinor:  n.cfg.ProtoMinor,
		TCPPort:     n.tcpPort,
		InstanceID:  n.cfg.InstanceID,
	}
	n.mu.Unlock()
	if oursMajor == 0 {
		oursMajor = DefaultProtoMajor
		self.ProtoMajor = oursMajor
	}

	if !CompatibleProto(oursMajor, req.ProtoMajor) {
		n.noteProtoMismatch(req.PeerID, req.ProtoMajor)
		_ = writeRegister(conn, Register{
			Type:       "register_reject",
			PeerID:     self.PeerID,
			ProtoMajor: oursMajor,
			ProtoMinor: self.ProtoMinor,
			InstanceID: self.InstanceID,
			Reason:     "proto_major_mismatch",
		})
		return
	}

	n.rememberPeer(peerFromRegister(req, host))
	n.cfg.Logf("register_rx peer_id=%s name=%s from=%s", req.PeerID, req.DisplayName, conn.RemoteAddr().String())
	_ = writeRegister(conn, self)
}

func (n *Node) readLoop(ctx context.Context, conn net.PacketConn) {
	defer n.wg.Done()
	buf := make([]byte, 65535)
	for {
		_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		nb, addr, err := conn.ReadFrom(buf)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			if ctx.Err() != nil {
				return
			}
			continue
		}
		a, err := DecodeAnnounce(buf[:nb])
		if err != nil {
			continue
		}
		if a.PeerID == n.cfg.PeerID {
			continue
		}
		n.cfg.Logf("%s", FormatAnnounceRx(a, addr.String()))
		if n.cfg.OnAnnounce != nil {
			n.cfg.OnAnnounce(a, addr)
		}
		n.maybeRegister(a, addr)
	}
}

func (n *Node) maybeRegister(a Announce, from net.Addr) {
	if a.TCPPort <= 0 {
		return
	}
	n.mu.Lock()
	ours := n.cfg.ProtoMajor
	n.mu.Unlock()
	if !CompatibleProto(ours, a.ProtoMajor) {
		n.noteProtoMismatch(a.PeerID, a.ProtoMajor)
		return
	}
	host := hostFromAddr(from)
	n.mu.Lock()
	if _, busy := n.dialing[a.PeerID]; busy {
		n.mu.Unlock()
		return
	}
	// Skip dial if already known with same instance; refresh LastSeen (P034 TTL).
	for _, p := range n.cfg.Peers.List() {
		if p.PeerID == a.PeerID && p.InstanceID == a.InstanceID {
			n.mu.Unlock()
			_ = n.cfg.Peers.Touch(a.PeerID)
			return
		}
	}
	n.dialing[a.PeerID] = struct{}{}
	n.mu.Unlock()

	go func() {
		defer func() {
			n.mu.Lock()
			delete(n.dialing, a.PeerID)
			n.mu.Unlock()
		}()
		_, _ = n.dialRegister(host, a.TCPPort)
	}()
}

func (n *Node) dialRegister(host string, port int) (*Peer, error) {
	if err := CheckDialHost(host); err != nil {
		n.cfg.Logf("%s", FormatWanRefuse(host))
		return nil, err
	}
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	n.mu.Lock()
	dial := n.cfg.Dialer
	timeout := n.cfg.DialTimeout
	n.mu.Unlock()
	if dial == nil {
		dial = net.DialTimeout
	}
	conn, err := dial("tcp", addr, timeout)
	if err != nil {
		n.cfg.Logf("register_dial_err addr=%s err=%v", addr, err)
		return nil, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	n.mu.Lock()
	req := Register{
		Type:        "register",
		PeerID:      n.cfg.PeerID,
		DisplayName: n.cfg.DisplayName,
		ProtoMajor:  n.cfg.ProtoMajor,
		ProtoMinor:  n.cfg.ProtoMinor,
		TCPPort:     n.tcpPort,
		InstanceID:  n.cfg.InstanceID,
	}
	n.mu.Unlock()

	if err := writeRegister(conn, req); err != nil {
		return nil, err
	}
	br := bufio.NewReader(conn)
	resp, err := readRegister(br)
	if err != nil {
		return nil, err
	}
	if resp.Type == "register_reject" {
		peerID := resp.PeerID
		if peerID == "" {
			peerID = host
		}
		n.noteProtoMismatch(peerID, resp.ProtoMajor)
		return nil, fmt.Errorf("discovery: register rejected")
	}
	n.mu.Lock()
	ours := n.cfg.ProtoMajor
	n.mu.Unlock()
	if !CompatibleProto(ours, resp.ProtoMajor) {
		n.noteProtoMismatch(resp.PeerID, resp.ProtoMajor)
		return nil, fmt.Errorf("discovery: proto mismatch")
	}
	p := peerFromRegister(resp, host)
	n.rememberPeer(p)
	n.cfg.Logf("register_ok peer_id=%s name=%s addr=%s", resp.PeerID, resp.DisplayName, addr)
	return &p, nil
}

func (n *Node) announceLoop(ctx context.Context) {
	defer n.wg.Done()
	t := time.NewTicker(n.cfg.Interval)
	defer t.Stop()
	n.sendOnce()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			n.sendOnce()
		}
	}
}

func (n *Node) sendOnce() {
	n.mu.Lock()
	conn := n.conn
	cfg := n.cfg
	tcpPort := n.tcpPort
	n.mu.Unlock()
	if conn == nil {
		return
	}
	raw, err := EncodeAnnounce(Announce{
		PeerID:      cfg.PeerID,
		DisplayName: cfg.DisplayName,
		ProtoMajor:  cfg.ProtoMajor,
		ProtoMinor:  cfg.ProtoMinor,
		TCPPort:     tcpPort,
		InstanceID:  cfg.InstanceID,
	})
	if err != nil {
		return
	}
	dst, err := n.destination()
	if err != nil {
		return
	}
	_, _ = conn.WriteTo(raw, dst)
}

func peekJSONType(line []byte) string {
	var head struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(line, &head); err != nil {
		return ""
	}
	return head.Type
}

func (n *Node) destination() (net.Addr, error) {
	n.mu.Lock()
	target := n.cfg.Target
	udpPort := n.cfg.UDPPort
	local := n.local
	n.mu.Unlock()
	if target != "" {
		ua, err := net.ResolveUDPAddr("udp4", target)
		if err != nil {
			return nil, err
		}
		// Broadcast/multicast stay allowed; unicast must be LAN (DUD-NET-101).
		if ua.IP != nil && !ua.IP.IsMulticast() && !ua.IP.Equal(net.IPv4bcast) {
			if err := CheckDialHost(ua.IP.String()); err != nil {
				n.cfg.Logf("%s", FormatWanRefuse(ua.IP.String()))
				return nil, err
			}
		}
		return ua, nil
	}
	port := udpPort
	if port == 0 {
		if ua, ok := local.(*net.UDPAddr); ok && ua.Port != 0 {
			port = ua.Port
		} else {
			port = DefaultUDPPort
		}
	}
	return &net.UDPAddr{IP: net.IPv4bcast, Port: port}, nil
}
