package discovery

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

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
	Peers       *PeerStore
	OnAnnounce  func(Announce, net.Addr)
	Logf        func(format string, args ...any)
	DialTimeout time.Duration
}

// Node sends periodic announces, accepts TCP register, and dials peers on announce.
type Node struct {
	cfg     Config
	mu      sync.Mutex
	conn    net.PacketConn
	tcpLn   net.Listener
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	local   net.Addr
	dialing map[string]struct{}
	tcpPort int
	proto   *protoBook
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
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = 2 * time.Second
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

// Status returns proto health including recent incompatible peers (P023).
func (n *Node) Status() Status {
	n.mu.Lock()
	major, minor := n.cfg.ProtoMajor, n.cfg.ProtoMinor
	n.mu.Unlock()
	if major == 0 {
		major = DefaultProtoMajor
	}
	return Status{
		ProtoMajor:   major,
		ProtoMinor:   minor,
		Incompatible: n.proto.list(),
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

// Start binds TCP+UDP, begins announce/register loops.
func (n *Node) Start() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.conn != nil {
		return errors.New("discovery: already started")
	}
	if n.cfg.PeerID == "" || n.cfg.InstanceID == "" {
		return errors.New("discovery: peer_id and instance_id required")
	}

	tcpBind := n.cfg.TCPBind
	if tcpBind == "" {
		if n.cfg.TCPPort == 0 {
			tcpBind = ":0"
		} else {
			tcpBind = fmt.Sprintf(":%d", n.cfg.TCPPort)
		}
	}
	tcpLn, err := net.Listen("tcp", tcpBind)
	if err != nil {
		return fmt.Errorf("discovery: tcp listen: %w", err)
	}
	tcpAddr := tcpLn.Addr().(*net.TCPAddr)
	n.tcpPort = tcpAddr.Port
	n.cfg.TCPPort = n.tcpPort

	bind := n.cfg.Bind
	if bind == "" {
		bind = fmt.Sprintf(":%d", n.cfg.UDPPort)
	}
	conn, err := listenUDP(bind)
	if err != nil {
		_ = tcpLn.Close()
		return err
	}
	if err := setBroadcast(conn); err != nil {
		_ = conn.Close()
		_ = tcpLn.Close()
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	n.conn = conn
	n.tcpLn = tcpLn
	n.cancel = cancel
	n.local = conn.LocalAddr()

	n.wg.Add(3)
	go n.readLoop(ctx, conn)
	go n.announceLoop(ctx)
	go n.acceptLoop(ctx, tcpLn)
	return nil
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
		go n.handleRegisterConn(conn)
	}
}

func (n *Node) handleRegisterConn(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(n.cfg.DialTimeout))
	br := bufio.NewReader(conn)
	req, err := readRegister(br)
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

	host := hostFromAddr(conn.RemoteAddr())
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
	// Skip dial if already known with same instance.
	for _, p := range n.cfg.Peers.List() {
		if p.PeerID == a.PeerID && p.InstanceID == a.InstanceID {
			n.mu.Unlock()
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
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", addr, n.cfg.DialTimeout)
	if err != nil {
		n.cfg.Logf("register_dial_err addr=%s err=%v", addr, err)
		return nil, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(n.cfg.DialTimeout))

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

func (n *Node) destination() (net.Addr, error) {
	n.mu.Lock()
	target := n.cfg.Target
	udpPort := n.cfg.UDPPort
	local := n.local
	n.mu.Unlock()
	if target != "" {
		return net.ResolveUDPAddr("udp4", target)
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
