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
	watch := flag.Bool("watch", false, "refresh + read stdin lines (Enter = send, /nick Имя) until Ctrl+C")
	send := flag.String("send", "", "send one text line to engine, print frame, exit")
	nick := flag.String("nick", "", "change display name via engine, print frame, exit")
	fetchID := flag.String("fetch", "", "start file download by file_id and print progress frames until done")
	interval := flag.Duration("interval", time.Second, "refresh interval in -watch mode")
	flag.Parse()

	fmt.Printf("dudka %s\n", version.Version)

	client := tui.NewClient(*engine)
	printFrame := func() tui.Snapshot {
		snap, err := client.Fetch()
		if err != nil {
			fmt.Print(tui.Render(tui.Snapshot{EngineOK: false, Err: err.Error()}))
			return tui.Snapshot{}
		}
		fmt.Print(tui.Render(snap))
		return snap
	}

	if name := strings.TrimSpace(*nick); name != "" {
		if _, err := client.SetNick(name); err != nil {
			fmt.Fprintf(os.Stderr, "dudka: nick: %v\n", err)
			os.Exit(1)
		}
		printFrame()
		return
	}

	if text := strings.TrimSpace(*send); text != "" {
		if _, err := client.Send(text); err != nil {
			fmt.Fprintf(os.Stderr, "dudka: send: %v\n", err)
			os.Exit(1)
		}
		printFrame()
		return
	}

	if fid := strings.TrimSpace(*fetchID); fid != "" {
		plan, err := client.BeginFetch(fid, false)
		if err != nil {
			fmt.Fprintf(os.Stderr, "dudka: fetch: %v\n", err)
			os.Exit(1)
		}
		if plan.Warning != "" {
			// Non-interactive: show warning, then proceed (DUD-FILE-111 — no hard block).
			fmt.Fprintln(os.Stderr, plan.Warning)
			plan, err = client.BeginFetch(fid, true)
			if err != nil {
				fmt.Fprintf(os.Stderr, "dudka: fetch: %v\n", err)
				os.Exit(1)
			}
		}
		if !plan.Started {
			fmt.Fprintf(os.Stderr, "dudka: fetch did not start\n")
			os.Exit(1)
		}
		deadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(deadline) {
			snap := printFrame()
			done := false
			for _, tr := range snap.Transfers {
				if tr.FileID == fid && (tr.Status == tui.TransferDone || tr.Status == tui.TransferError || tr.Status == tui.TransferCancelled || tr.Percent >= 100) {
					done = true
					if tr.Status == tui.TransferError {
						os.Exit(1)
					}
					break
				}
			}
			if done {
				return
			}
			fmt.Println("---")
			time.Sleep(50 * time.Millisecond)
		}
		fmt.Fprintf(os.Stderr, "dudka: fetch timeout\n")
		os.Exit(1)
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
				if _, ok := err.(*tui.ErrLargeFileWarning); ok {
					fmt.Fprintf(os.Stderr, "%s\n", err.Error())
				} else {
					fmt.Fprintf(os.Stderr, "dudka: %v\n", err)
				}
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
