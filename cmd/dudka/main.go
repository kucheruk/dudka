// Command dudka is the apartment LAN chat TUI (interactive by default, P046).
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

	"golang.org/x/term"
)

func main() {
	engine := flag.String("engine", defaultEngine(), "dudkad loopback base (host:port or URL)")
	watch := flag.Bool("watch", false, "legacy line-mode refresh+stdin (scripts); prefer interactive TUI")
	once := flag.Bool("once", false, "print one plain frame and exit (scripts / non-TTY)")
	send := flag.String("send", "", "send one text line to engine, print frame, exit")
	nick := flag.String("nick", "", "change display name via engine, print frame, exit")
	fetchID := flag.String("fetch", "", "start file download by file_id and print progress frames until done")
	announcePath := flag.String("announce", "", "announce a local file into the feed and print frame")
	interval := flag.Duration("interval", time.Second, "refresh interval in -watch mode")
	flag.Parse()

	explicitEngine := strings.TrimSpace(os.Getenv("DUDKA_ENGINE")) != ""
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "engine" {
			explicitEngine = true
		}
	})
	if err := ensureLocalEngine(*engine, explicitEngine); err != nil {
		fmt.Fprintf(os.Stderr, "dudka: %v\n", err)
		os.Exit(1)
	}

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
		fmt.Printf("dudka %s\n", version.Version)
		if _, err := client.SetNick(name); err != nil {
			fmt.Fprintf(os.Stderr, "dudka: nick: %v\n", err)
			os.Exit(1)
		}
		printFrame()
		return
	}

	if text := strings.TrimSpace(*send); text != "" {
		fmt.Printf("dudka %s\n", version.Version)
		if _, err := client.Send(text); err != nil {
			fmt.Fprintf(os.Stderr, "dudka: send: %v\n", err)
			os.Exit(1)
		}
		printFrame()
		return
	}

	if path := strings.TrimSpace(*announcePath); path != "" {
		fmt.Printf("dudka %s\n", version.Version)
		res, err := client.AnnouncePath(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "dudka: announce: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "announced file_id=%s name=%s mime=%s\n", res.FileID, res.Name, res.Mime)
		printFrame()
		return
	}

	if fid := strings.TrimSpace(*fetchID); fid != "" {
		fmt.Printf("dudka %s\n", version.Version)
		runFetch(client, fid, printFrame)
		return
	}

	// Script / pipe path: one plain frame (keeps protocol tests green).
	if *once || *watch || !isInteractive() {
		fmt.Printf("dudka %s\n", version.Version)
		printFrame()
		if !*watch {
			return
		}
		runWatch(client, *interval, printFrame)
		return
	}

	// Default: real interactive TUI (alt screen, fixed panels).
	if err := tui.RunInteractive(*engine); err != nil {
		fmt.Fprintf(os.Stderr, "dudka: %v\n", err)
		os.Exit(1)
	}
}

func defaultEngine() string {
	if v := strings.TrimSpace(os.Getenv("DUDKA_ENGINE")); v != "" {
		return v
	}
	return "127.0.0.1:17880"
}

func isInteractive() bool {
	return term.IsTerminal(int(os.Stdout.Fd())) && term.IsTerminal(int(os.Stdin.Fd()))
}

func runFetch(client *tui.Client, fid string, printFrame func() tui.Snapshot) {
	plan, err := client.BeginFetch(fid, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dudka: fetch: %v\n", err)
		os.Exit(1)
	}
	if plan.Warning != "" {
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

func runWatch(client *tui.Client, interval time.Duration, printFrame func() tui.Snapshot) {
	ticker := time.NewTicker(interval)
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
