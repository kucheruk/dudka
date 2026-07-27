package discovery

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

// Config configures a discovery node (announce tx/rx).
type Config struct {
	PeerID      string
	DisplayName string
	InstanceID  string
	ProtoMajor  int
	ProtoMinor  int
	TCPPort     int
	UDPPort     int           // used when Bind empty; default DefaultUDPPort
	Bind        string        // e.g. 127.0.0.1:0 or :41777; empty => 0.0.0.0:UDPPort
	Target      string        // empty => UDP broadcast to 255.255.255.255:UDPPort
	Interval    time.Duration // default 2s
	OnAnnounce  func(Announce, net.Addr)
	Logf        func(format string, args ...any)
}

// Node sends periodic announces and listens for peers' announces.
type Node struct {
	cfg    Config
	mu     sync.Mutex
	conn   net.PacketConn
	cancel context.CancelFunc
	wg     sync.WaitGroup
	local  net.Addr
}

// NewNode builds a stopped discovery node.
func NewNode(cfg Config) *Node {
	if cfg.ProtoMajor == 0 {
		cfg.ProtoMajor = DefaultProtoMajor
	}
	if cfg.UDPPort == 0 && cfg.Bind == "" {
		cfg.UDPPort = DefaultUDPPort
	}
	if cfg.TCPPort == 0 {
		cfg.TCPPort = cfg.UDPPort
		if cfg.TCPPort == 0 {
			cfg.TCPPort = DefaultUDPPort
		}
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 2 * time.Second
	}
	if cfg.Logf == nil {
		cfg.Logf = func(string, ...any) {}
	}
	return &Node{cfg: cfg}
}

// LocalAddr returns the bound UDP address after Start.
func (n *Node) LocalAddr() net.Addr {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.local
}

// Start binds UDP, begins receive loop and periodic announce.
func (n *Node) Start() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.conn != nil {
		return errors.New("discovery: already started")
	}
	if n.cfg.PeerID == "" || n.cfg.InstanceID == "" {
		return errors.New("discovery: peer_id and instance_id required")
	}

	bind := n.cfg.Bind
	if bind == "" {
		bind = fmt.Sprintf(":%d", n.cfg.UDPPort)
	}
	conn, err := listenUDP(bind)
	if err != nil {
		return err
	}
	if err := setBroadcast(conn); err != nil {
		_ = conn.Close()
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	n.conn = conn
	n.cancel = cancel
	n.local = conn.LocalAddr()

	n.wg.Add(2)
	go n.readLoop(ctx, conn)
	go n.announceLoop(ctx)
	return nil
}

// Stop closes the UDP socket and waits for goroutines.
func (n *Node) Stop() error {
	n.mu.Lock()
	cancel := n.cancel
	conn := n.conn
	n.cancel = nil
	n.conn = nil
	n.local = nil
	n.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	var err error
	if conn != nil {
		err = conn.Close()
	}
	n.wg.Wait()
	return err
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
			continue // ignore self
		}
		n.cfg.Logf("%s", FormatAnnounceRx(a, addr.String()))
		if n.cfg.OnAnnounce != nil {
			n.cfg.OnAnnounce(a, addr)
		}
	}
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
	n.mu.Unlock()
	if conn == nil {
		return
	}
	raw, err := EncodeAnnounce(Announce{
		PeerID:      cfg.PeerID,
		DisplayName: cfg.DisplayName,
		ProtoMajor:  cfg.ProtoMajor,
		ProtoMinor:  cfg.ProtoMinor,
		TCPPort:     cfg.TCPPort,
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
	if n.cfg.Target != "" {
		return net.ResolveUDPAddr("udp4", n.cfg.Target)
	}
	port := n.cfg.UDPPort
	if port == 0 {
		if ua, ok := n.LocalAddr().(*net.UDPAddr); ok && ua.Port != 0 {
			port = ua.Port
		} else {
			port = DefaultUDPPort
		}
	}
	return &net.UDPAddr{IP: net.IPv4bcast, Port: port}, nil
}
