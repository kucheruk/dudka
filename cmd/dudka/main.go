// Command dudka is the Linux TUI client for the apartment LAN chat.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dudka/internal/tui"
	"dudka/internal/version"
)

func main() {
	engine := flag.String("engine", "127.0.0.1:17880", "dudkad loopback base (host:port or URL)")
	watch := flag.Bool("watch", false, "refresh status/peers until Ctrl+C (default: one frame)")
	interval := flag.Duration("interval", time.Second, "refresh interval in -watch mode")
	flag.Parse()

	fmt.Printf("dudka %s\n", version.Version)

	client := tui.NewClient(*engine)
	printFrame := func() {
		snap, err := client.Fetch()
		if err != nil {
			fmt.Print(tui.Render(tui.Snapshot{EngineOK: false, Err: err.Error()}))
			return
		}
		fmt.Print(tui.Render(snap))
	}

	printFrame()
	if !*watch {
		return
	}

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	for {
		select {
		case <-sig:
			fmt.Println()
			return
		case <-ticker.C:
			fmt.Println("---")
			printFrame()
		}
	}
}
