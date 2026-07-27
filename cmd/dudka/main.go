// Command dudka is the Linux TUI client for the apartment LAN chat.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"dudka/internal/tui"
	"dudka/internal/version"
)

func main() {
	engine := flag.String("engine", "127.0.0.1:17880", "dudkad loopback base (host:port or URL)")
	watch := flag.Bool("watch", false, "refresh + read stdin lines (Enter = send) until Ctrl+C")
	send := flag.String("send", "", "send one text line to engine, print frame, exit")
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

	if text := strings.TrimSpace(*send); text != "" {
		if _, err := client.Send(text); err != nil {
			fmt.Fprintf(os.Stderr, "dudka: send: %v\n", err)
			os.Exit(1)
		}
		printFrame()
		return
	}

	printFrame()
	if !*watch {
		return
	}

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	lines := make(chan string, 1)
	go func() {
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			lines <- sc.Text()
		}
		close(lines)
	}()

	fmt.Fprint(os.Stderr, "> ")
	for {
		select {
		case <-sig:
			fmt.Println()
			return
		case line, ok := <-lines:
			if !ok {
				return
			}
			if err := tui.HandleComposeLine(client, line); err != nil {
				fmt.Fprintf(os.Stderr, "dudka: send: %v\n", err)
			}
			fmt.Println("---")
			printFrame()
			fmt.Fprint(os.Stderr, "> ")
		case <-ticker.C:
			fmt.Println("---")
			printFrame()
			fmt.Fprint(os.Stderr, "> ")
		}
	}
}
