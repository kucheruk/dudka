package tui

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"dudka/internal/version"
)

const (
	diagnosticLogLines = 12
	diagnosticLogBytes = 6 << 10
)

func diagnosticBundle(
	snap Snapshot,
	engineBase string,
	lastError string,
	errorAt time.Time,
	collectedAt time.Time,
	logTail string,
) string {
	network := strings.TrimSpace(snap.Network)
	if network == "" {
		network = "unknown"
	}
	errorTime := "unknown"
	if !errorAt.IsZero() {
		errorTime = errorAt.Format(time.RFC3339)
	}
	terminal := strings.TrimSpace(os.Getenv("TERM"))
	if terminal == "" {
		terminal = "unknown"
	}
	multiplexer := "none"
	switch {
	case os.Getenv("TMUX") != "":
		multiplexer = "tmux"
	case strings.HasPrefix(terminal, "screen"):
		multiplexer = "screen"
	}
	logTail = boundLogTail(sanitizeDiagnosticText(strings.TrimSpace(logTail)))
	if logTail == "" {
		logTail = "(tui.log пуст или недоступен)"
	}

	return fmt.Sprintf(`ДУДКА — ДИАГНОСТИКА ДЛЯ АГЕНТА
собрано: %s
ошибка_в: %s
версия: %s
платформа: %s/%s
терминал: %s
мультиплексор: %s
движок: %s
движок_доступен: %t
сеть: %s
соседей: %d
сообщений: %d
передач: %d
протокол: %d.%d
порты: announce=%d session=%d

ОШИБКА
%s

ХВОСТ tui.log (%d строк, не более %d байт)
%s
`,
		collectedAt.Format(time.RFC3339),
		errorTime,
		version.Version,
		runtime.GOOS,
		runtime.GOARCH,
		sanitizeDiagnosticText(terminal),
		multiplexer,
		engineKind(engineBase),
		snap.EngineOK,
		sanitizeDiagnosticText(network),
		len(snap.Peers),
		len(snap.Messages),
		len(snap.Transfers),
		snap.ProtoMajor,
		snap.ProtoMinor,
		snap.AnnouncePort,
		snap.SessionPort,
		sanitizeDiagnosticText(lastError),
		diagnosticLogLines,
		diagnosticLogBytes,
		logTail,
	)
}

func engineKind(base string) string {
	base = strings.ToLower(strings.TrimSpace(base))
	if strings.Contains(base, "127.0.0.1") || strings.Contains(base, "localhost") || strings.Contains(base, "[::1]") {
		return "loopback"
	}
	if base == "" {
		return "не задан"
	}
	return "нестандартный адрес (скрыт)"
}

func readTUILogTail() string {
	path := tuiLogPath()
	if path == "" {
		return ""
	}
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return ""
	}
	start := info.Size() - diagnosticLogBytes
	if start < 0 {
		start = 0
	}
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return ""
	}
	raw, err := io.ReadAll(io.LimitReader(f, diagnosticLogBytes))
	if err != nil {
		return ""
	}
	text := string(raw)
	if start > 0 {
		if cut := strings.IndexByte(text, '\n'); cut >= 0 {
			text = text[cut+1:]
		}
	}
	return boundLogTail(text)
}

func boundLogTail(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) > diagnosticLogLines {
		lines = lines[len(lines)-diagnosticLogLines:]
	}
	return strings.Join(lines, "\n")
}

func sanitizeDiagnosticText(text string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\n' || r == '\t':
			return r
		case r < 0x20 || r == 0x7f:
			return -1
		default:
			return r
		}
	}, text)
}
