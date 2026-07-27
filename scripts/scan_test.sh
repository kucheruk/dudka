#!/usr/bin/env bash
# Task-level contract for P024: POST /scan finds peer with UDP broadcast filtered out.
# Run: ./scripts/scan_test.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1

fail() {
  echo "scan_test FAIL: $*" >&2
  exit 1
}

go test ./internal/discovery/ ./internal/loopback/ -count=1 -run 'Scan|PostScan' >/dev/null || fail "unit tests failed"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"; [[ -n "${pid_a:-}" ]] && kill "$pid_a" 2>/dev/null || true; [[ -n "${pid_b:-}" ]] && kill "$pid_b" 2>/dev/null || true' EXIT
bin="$tmpdir/dudkad"
go build -o "$bin" ./cmd/dudkad || fail "go build failed"

# Different announce ports + long interval = UDP broadcast path filtered/disabled.
port_a="$(python3 - <<'PY'
import socket
s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
)"
port_b="$(python3 - <<'PY'
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
  -announce-port "$port_a" -session-port 0 -announce-interval 1h >"$log_a" 2>&1 &
pid_a=$!
"$bin" -data-dir "$tmpdir/b" -name "Bob" -listen "127.0.0.1:0" \
  -announce-port "$port_b" -session-port 0 -announce-interval 1h >"$log_b" 2>&1 &
pid_b=$!

listen_a=""
peer_b=""
session_b=""
for _ in $(seq 1 50); do
  if grep -q '^ready ' "$log_a" 2>/dev/null && grep -q '^ready ' "$log_b" 2>/dev/null; then
    listen_a="$(grep '^listen=' "$log_a" | head -n 1 | sed 's/^listen=//')"
    peer_b="$(sed -n 's/^peer_id=//p' "$log_b" | head -1)"
    session_b="$(grep '^session_tcp=' "$log_b" | head -n 1 | sed 's/^session_tcp=//')"
    break
  fi
  sleep 0.1
done
[[ -n "$listen_a" && -n "$session_b" ]] || fail "not ready; a=$(cat "$log_a") b=$(cat "$log_b")"

# Without scan, Alice should not know Bob (UDP filtered).
peers0="$(curl -sS --max-time 2 "http://${listen_a}/peers")"
python3 - "$peers0" "$peer_b" <<'PY'
import json, sys
d = json.loads(sys.argv[1])
peer_b = sys.argv[2]
ids = {p.get("peer_id") for p in d.get("peers") or []}
assert peer_b not in ids, d
print("pre_scan_empty_ok")
PY

scan="$(curl -sS --max-time 5 -X POST "http://${listen_a}/scan" \
  -H 'Content-Type: application/json' \
  -d "{\"hosts\":[\"127.0.0.1\"],\"port\":${session_b}}")"
python3 - "$scan" "$peer_b" <<'PY'
import json, sys
d = json.loads(sys.argv[1])
peer_b = sys.argv[2]
assert d.get("found") == 1, d
ids = {p.get("peer_id") for p in d.get("peers") or []}
assert peer_b in ids, d
print("scan_ok")
PY

peers1="$(curl -sS --max-time 2 "http://${listen_a}/peers")"
python3 - "$peers1" "$peer_b" <<'PY'
import json, sys
d = json.loads(sys.argv[1])
peer_b = sys.argv[2]
ids = {p.get("peer_id") for p in d.get("peers") or []}
assert peer_b in ids, d
print("peers_ok")
PY

grep -q "scan_hit peer_id=${peer_b}" "$log_a" || fail "missing scan_hit; a=$(cat "$log_a")"

kill "$pid_a" "$pid_b" 2>/dev/null || true
wait "$pid_a" 2>/dev/null || true
wait "$pid_b" 2>/dev/null || true
pid_a=""; pid_b=""

echo "scan_test OK"
