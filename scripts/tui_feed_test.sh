#!/usr/bin/env bash
# Task-level contract for P041: TUI shows message feed from engine.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1

fail() {
  echo "tui_feed_test FAIL: $*" >&2
  exit 1
}

go test ./internal/tui/ -count=1 >/dev/null || fail "unit tests failed"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"; [[ -n "${pid:-}" ]] && kill "$pid" 2>/dev/null || true' EXIT
go build -o "$tmpdir/dudkad" ./cmd/dudkad || fail "build dudkad"
go build -o "$tmpdir/dudka" ./cmd/dudka || fail "build dudka"

log="$tmpdir/a.log"
"$tmpdir/dudkad" -data-dir "$tmpdir/a" -name "Аня" -listen "127.0.0.1:0" \
  -announce-port 0 -session-port 0 -announce-interval 1h >"$log" 2>&1 &
pid=$!

listen=""
for _ in $(seq 1 50); do
  if grep -q '^ready ' "$log" 2>/dev/null; then
    listen="$(grep '^listen=' "$log" | head -n 1 | sed 's/^listen=//')"
    break
  fi
  sleep 0.1
done
[[ -n "$listen" ]] || fail "dudkad not ready: $(cat "$log")"

curl -sS --max-time 2 -X POST "http://${listen}/send" \
  -H 'Content-Type: application/json' \
  -d '{"text":"лента из engine"}' >/dev/null || fail "POST /send failed"

frame="$("$tmpdir/dudka" -engine "$listen")"
printf '%s\n' "$frame" | grep -q 'FEED' || fail "missing FEED:\n$frame"
printf '%s\n' "$frame" | grep -q 'лента из engine' || fail "missing message text:\n$frame"
printf '%s\n' "$frame" | grep -q 'Аня' || fail "missing nick in feed:\n$frame"
printf '%s\n' "$frame" | grep -q '·' || fail "missing · separators:\n$frame"
# Alone with messages still shows empty peers copy + feed.
printf '%s\n' "$frame" | grep -q 'НИКОГО РЯДОМ' || fail "missing empty peers copy:\n$frame"

kill "$pid" 2>/dev/null || true
wait "$pid" 2>/dev/null || true
pid=""

echo "tui_feed_test OK"
