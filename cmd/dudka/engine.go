package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func ensureLocalEngine(engine string, explicit bool) error {
	if explicit || strings.TrimRight(engine, "/") != "127.0.0.1:17880" {
		return nil
	}
	if engineReady(engine) {
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	dudkad := filepath.Join(filepath.Dir(exe), "dudkad")
	if _, err := os.Stat(dudkad); err != nil {
		return fmt.Errorf("движок не запущен; рядом с dudka не найден dudkad")
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		cache = os.TempDir()
	}
	logDir := filepath.Join(cache, "dudka")
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return err
	}
	log, err := os.OpenFile(filepath.Join(logDir, "dudkad.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer log.Close()
	cmd := exec.Command(dudkad)
	cmd.Env = append(os.Environ(), "DUDKA_NO_PROMPT=1")
	cmd.Stdin = nil
	cmd.Stdout = log
	cmd.Stderr = log
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("запуск dudkad: %w", err)
	}
	_ = cmd.Process.Release()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		if engineReady(engine) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("dudkad не ответил за 5 секунд; лог: %s", filepath.Join(logDir, "dudkad.log"))
}

func engineReady(engine string) bool {
	url := engine
	if !strings.Contains(url, "://") {
		url = "http://" + url
	}
	client := http.Client{Timeout: 250 * time.Millisecond}
	res, err := client.Get(strings.TrimRight(url, "/") + "/me")
	if err != nil {
		return false
	}
	_ = res.Body.Close()
	return res.StatusCode >= 200 && res.StatusCode < 300
}
