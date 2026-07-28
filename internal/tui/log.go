package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func logTUIError(scope string, err error) {
	cache, cacheErr := os.UserCacheDir()
	if cacheErr != nil || cache == "" {
		return
	}
	dir := filepath.Join(cache, "dudka")
	if mkdirErr := os.MkdirAll(dir, 0o700); mkdirErr != nil {
		return
	}
	f, openErr := os.OpenFile(filepath.Join(dir, "tui.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if openErr != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "%s %s: %v\n", time.Now().Format(time.RFC3339), scope, err)
}
