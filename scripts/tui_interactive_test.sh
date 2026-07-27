#!/usr/bin/env bash
# Task-level contract for P046: interactive TUI (bubbletea) layout + smoke.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail() {
  echo "tui_interactive_test FAIL: $*" >&2
  exit 1
}

command -v go >/dev/null 2>&1 || fail "go not on PATH"

grep -q 'bubbletea' go.mod || fail "bubbletea not in go.mod"
grep -q 'lipgloss' go.mod || fail "lipgloss not in go.mod"
grep -q 'RunInteractive' internal/tui/app.go || fail "RunInteractive missing"
grep -q 'RenderScreen' internal/tui/screen.go || fail "RenderScreen missing"
grep -q 'isInteractive' cmd/dudka/main.go || fail "TTY gate missing in main"
grep -q 'P046' ROADMAP.md || fail "P046 not in ROADMAP"

go test ./internal/tui/ -run 'TestLayoutFor|TestRenderScreen|TestNewModelInit' -count=1 \
  || fail "interactive TUI unit tests failed"

# Non-TTY still dumps plain frame (script contract).
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
go build -o "$tmpdir/dudkad" ./cmd/dudkad
go build -o "$tmpdir/dudka" ./cmd/dudka
"$tmpdir/dudkad" -data-dir "$tmpdir/d" -name "TUI" -listen "127.0.0.1:0" \
  -announce-port 0 -session-port 0 -announce-interval 1h >"$tmpdir/e.log" 2>&1 &
pid=$!
listen=""
for _ in $(seq 1 50); do
  if grep -q '^ready ' "$tmpdir/e.log" 2>/dev/null; then
    listen="$(grep '^listen=' "$tmpdir/e.log" | head -n1 | sed 's/^listen=//')"
    break
  fi
  sleep 0.1
done
[[ -n "$listen" ]] || fail "engine not ready"
frame="$("$tmpdir/dudka" -engine "$listen" -once)"
printf '%s\n' "$frame" | grep -q 'СОСЕДИ\|ДУДКА' || fail "once frame missing RU labels:\n$frame"
kill "$pid" 2>/dev/null || true
wait "$pid" 2>/dev/null || true

echo "tui_interactive_test OK"
