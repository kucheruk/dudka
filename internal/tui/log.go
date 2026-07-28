package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func tuiLogPath() string {
	cache, err := os.UserCacheDir()
	if err != nil || cache == "" {
		return ""
	}
	return filepath.Join(cache, "dudka", "tui.log")
}

func logTUIError(scope string, err error) {
	path := tuiLogPath()
	if path == "" {
		return
	}
	if mkdirErr := os.MkdirAll(filepath.Dir(path), 0o700); mkdirErr != nil {
		return
	}
	f, openErr := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if openErr != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "%s %s: %v\n", time.Now().Format(time.RFC3339), scope, err)
}
