// Command dudkad is the LAN chat engine (stub until discovery and loopback land).
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"dudka/internal/identity"
	"dudka/internal/version"
)

func main() {
	dataDir := flag.String("data-dir", defaultDataDir(), "directory for local peer state")
	flag.Parse()

	fmt.Printf("dudkad %s\n", version.Version)

	peerID, err := identity.LoadOrCreate(*dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dudkad: peer_id: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("peer_id=%s\n", peerID)
}

func defaultDataDir() string {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		return filepath.Join(".", ".dudka")
	}
	return filepath.Join(base, "dudka")
}
