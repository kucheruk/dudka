#!/usr/bin/env bash
# Task-level contract for P020: second process sees UDP announce (announce_rx).
# Run: ./scripts/announce_test.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1

fail() {
  echo "announce_test FAIL: $*" >&2
  exit 1
}

go test ./internal/discovery/ -count=1 >/dev/null || fail "discovery unit tests failed"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"; [[ -n "${pid_a:-}" ]] && kill "$pid_a" 2>/dev/null || true; [[ -n "${pid_b:-}" ]] && kill "$pid_b" 2>/dev/null || true' EXIT
bin="$tmpdir/dudkad"
go build -o "$bin" ./cmd/dudkad || fail "go build failed"

# Ephemeral announce port avoids clashing with a real LAN :41777.
port="$(python3 - <<'PY'
import socket
s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
)"

log_a="$tmpdir/a.log"
log_b="$tmpdir/b.log"
"$bin" -data-dir "$tmpdir/a" -name "Alice" -listen "127.0.0.1:0" \
  -announce-port "$port" -session-port 0 -announce-interval 200ms >"$log_a" 2>&1 &
pid_a=$!
"$bin" -data-dir "$tmpdir/b" -name "Bob" -listen "127.0.0.1:0" \
  -announce-port "$port" -session-port 0 -announce-interval 200ms >"$log_b" 2>&1 &
pid_b=$!

peer_a=""
for _ in $(seq 1 50); do
  if grep -q '^ready ' "$log_a" 2>/dev/null && grep -q '^ready ' "$log_b" 2>/dev/null; then
    peer_a="$(sed -n 's/^peer_id=//p' "$log_a" | head -1)"
    break
  fi
  sleep 0.1
done
[[ -n "$peer_a" ]] || fail "not ready; a=$(cat "$log_a") b=$(cat "$log_b")"

found=0
for _ in $(seq 1 50); do
  if grep -q "announce_rx peer_id=${peer_a}" "$log_b" 2>/dev/null; then
    found=1
    break
  fi
  sleep 0.1
done
[[ "$found" -eq 1 ]] || fail "Bob never saw Alice announce; b=$(cat "$log_b") a=$(cat "$log_a")"

kill "$pid_a" "$pid_b" 2>/dev/null || true
wait "$pid_a" 2>/dev/null || true
wait "$pid_b" 2>/dev/null || true
pid_a=""; pid_b=""

echo "announce_test OK"
