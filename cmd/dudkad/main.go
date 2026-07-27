// Command dudkad is the LAN chat engine.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"dudka/internal/discovery"
	"dudka/internal/identity"
	"dudka/internal/loopback"
	"dudka/internal/version"
)

func main() {
	dataDir := flag.String("data-dir", defaultDataDir(), "directory for local peer state")
	nameFlag := flag.String("name", "", "display name (nick); overrides saved name")
	listen := flag.String("listen", "127.0.0.1:17880", "loopback HTTP listen address")
	announcePort := flag.Int("announce-port", discovery.DefaultUDPPort, "UDP announce listen/broadcast port")
	announceInterval := flag.Duration("announce-interval", 2*time.Second, "UDP announce period")
	announceTarget := flag.String("announce-target", "", "optional unicast host:port instead of broadcast (tests)")
	flag.Parse()

	fmt.Printf("dudkad %s\n", version.Version)

	peerID, err := identity.LoadOrCreate(*dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dudkad: peer_id: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("peer_id=%s\n", peerID)

	promptEnabled := *nameFlag == "" && stdinIsTTY() && os.Getenv("DUDKA_NO_PROMPT") == "" && os.Getenv("CI") == ""
	displayName, err := identity.LoadOrCreateDisplayName(*dataDir, *nameFlag, os.Stdin, promptEnabled)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dudkad: display_name: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("display_name=%s\n", displayName)

	instanceID, err := identity.NewUUIDv4()
	if err != nil {
		fmt.Fprintf(os.Stderr, "dudkad: instance_id: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("instance_id=%s\n", instanceID)

	disc := discovery.NewNode(discovery.Config{
		PeerID:      peerID,
		DisplayName: displayName,
		InstanceID:  instanceID,
		UDPPort:     *announcePort,
		TCPPort:     *announcePort, // session TCP comes in P021; advertise same port for now
		Interval:    *announceInterval,
		Target:      *announceTarget,
		Logf:        func(format string, args ...any) { fmt.Printf(format+"\n", args...) },
	})
	if err := disc.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "dudkad: discovery: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = disc.Stop() }()
	if addr := disc.LocalAddr(); addr != nil {
		fmt.Printf("announce=%s\n", addr.String())
	}

	api := loopback.New(peerID, displayName)
	api.SetPersistName(func(name string) error {
		return identity.SaveDisplayName(*dataDir, name)
	})
	ln, err := api.Listen(*listen)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dudkad: listen: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("listen=%s\n", ln.Addr().String())
	fmt.Println(loopback.FormatReady(peerID, displayName))

	if err := api.Serve(ln); err != nil {
		fmt.Fprintf(os.Stderr, "dudkad: serve: %v\n", err)
		os.Exit(1)
	}
}

func defaultDataDir() string {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		return filepath.Join(".", ".dudka")
	}
	return filepath.Join(base, "dudka")
}

func stdinIsTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
