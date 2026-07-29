// Command dudka-signal is the memory-only WebRTC rendezvous for browser tabs.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"dudka/internal/signaling"
	"dudka/internal/stunserver"
	"dudka/internal/version"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:5251", "HTTP listen address")
	origin := flag.String("origin", "https://zamoo.team", "exact allowed browser origin")
	webDir := flag.String("web-dir", "", "optional static web directory for local testing")
	stunListen := flag.String("stun-listen", "", "optional public STUN UDP listen address")
	flag.Parse()

	if *stunListen != "" {
		go func() {
			log.Printf("dudka-stun %s listen=%s", version.Version, *stunListen)
			if err := stunserver.ListenAndServe(*stunListen); err != nil {
				fmt.Fprintf(os.Stderr, "dudka-stun: %v\n", err)
				os.Exit(1)
			}
		}()
	}
	signalHandler := signaling.NewServer(*origin).Handler()
	handler := signalHandler
	if *webDir != "" {
		mux := http.NewServeMux()
		mux.Handle("/health", signalHandler)
		mux.HandleFunc("/dudka/signal", func(w http.ResponseWriter, r *http.Request) {
			request := r.Clone(r.Context())
			request.URL.Path = "/"
			request.URL.RawPath = ""
			signalHandler.ServeHTTP(w, request)
		})
		mux.Handle("/", http.FileServer(http.Dir(*webDir)))
		handler = mux
	}
	server := &http.Server{
		Addr:              *listen,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       75 * time.Second,
	}
	log.Printf("dudka-signal %s listen=%s", version.Version, *listen)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintf(os.Stderr, "dudka-signal: %v\n", err)
		os.Exit(1)
	}
}
