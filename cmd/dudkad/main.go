// Command dudkad is the direct WebRTC chat engine.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"

	"dudka/internal/agent"
	"dudka/internal/chat"
	"dudka/internal/discovery"
	"dudka/internal/files"
	"dudka/internal/identity"
	"dudka/internal/loopback"
	"dudka/internal/rtcmesh"
	"dudka/internal/version"
)

func main() {
	dataDir := flag.String("data-dir", defaultDataDir(), "directory for local peer state")
	nameFlag := flag.String("name", "", "display name (nick); overrides saved name")
	listen := flag.String("listen", "127.0.0.1:17880", "loopback HTTP listen address")
	signalURL := flag.String("signal-url", "wss://zamoo.team/dudka/signal", "Studio signaling WebSocket")
	signalOrigin := flag.String("signal-origin", "https://zamoo.team", "signaling Origin header")
	stunURL := flag.String("stun-url", "stun:zamoo.team:3478", "Studio STUN service")
	asAgent := flag.Bool("agent", false, "this process is a home agent (requires triple-prefix nick)")
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
	if *asAgent {
		if err := agent.ValidateAgentNick(displayName); err != nil {
			fmt.Fprintf(os.Stderr, "dudkad: agent nick: %v (want agent·model·host)\n", err)
			os.Exit(1)
		}
	} else if agent.LooksLikeAgentNick(displayName) {
		fmt.Fprintf(os.Stderr, "dudkad: human nick must not use agent triple-prefix; pass -agent if this is an agent\n")
		os.Exit(1)
	}

	peers := discovery.NewPeerStore()
	msgs, err := chat.NewPersistentStore(filepath.Join(*dataDir, "messages.json"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "dudkad: history: %v\n", err)
		os.Exit(1)
	}
	blobs, err := files.NewStore(filepath.Join(*dataDir, "blobs"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "dudkad: blobs: %v\n", err)
		os.Exit(1)
	}
	var mesh *rtcmesh.Client
	hub := chat.NewHub(chat.Config{
		PeerID:    peerID,
		Name:      displayName,
		Store:     msgs,
		Peers:     peers,
		Blobs:     blobs,
		InboxDir:  filepath.Join(*dataDir, "inbox"),
		ThumbsDir: filepath.Join(*dataDir, "thumbs"),
		Logf:      func(format string, args ...any) { fmt.Printf(format+"\n", args...) },
		Broadcast: func(message chat.Message) int {
			if mesh == nil {
				return 0
			}
			return mesh.Broadcast(message)
		},
	})
	mesh = rtcmesh.New(rtcmesh.Config{
		PeerID: peerID, Name: displayName, Peers: peers, Blobs: blobs,
		History: hub.Messages, OnMessage: func(raw []byte) {
			hub.HandleChatLine("webrtc", raw)
		},
		SignalURL: *signalURL, Origin: *signalOrigin, STUNURL: *stunURL,
		Logf: func(format string, args ...any) { fmt.Printf(format+"\n", args...) },
	})
	defer mesh.Stop()

	api := loopback.New(peerID, displayName)
	api.SetPersistName(func(name string) error {
		if err := identity.SaveDisplayName(*dataDir, name); err != nil {
			return err
		}
		mesh.SetName(name)
		return nil
	})
	api.SetPeers(peers)
	api.SetChat(hub)
	api.SetIsAgent(*asAgent)
	api.SetUpdatesDir(filepath.Join(*dataDir, "updates"))
	api.SetStatusProvider(func() discovery.Status {
		return discovery.Status{
			ProtoMajor: discovery.DefaultProtoMajor,
			ProtoMinor: discovery.DefaultProtoMinor,
			Network:    "ok", Incompatible: []discovery.IncompatiblePeer{},
		}
	})
	api.SetScanProvider(func(_ context.Context, _ discovery.ScanRequest) (discovery.ScanResult, error) {
		mesh.Restart()
		return discovery.ScanResult{Peers: peers.List(), Found: len(peers.List())}, nil
	})
	consentPath := filepath.Join(*dataDir, "internet_consent")
	var internetEnabled atomic.Bool
	enableInternet := func() error {
		if internetEnabled.Load() {
			return nil
		}
		if err := os.MkdirAll(*dataDir, 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(consentPath, []byte("studio-signaling-and-stun\n"), 0o600); err != nil {
			return err
		}
		internetEnabled.Store(true)
		mesh.Start(context.Background())
		return nil
	}
	api.SetInternetConsent(internetEnabled.Load, enableInternet)
	if _, err := os.Stat(consentPath); err == nil {
		if err := enableInternet(); err != nil {
			fmt.Fprintf(os.Stderr, "dudkad: internet consent: %v\n", err)
		}
	}
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
