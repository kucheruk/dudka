#!/usr/bin/env bash
# Task-level contract for P021: TCP register → GET /peers shows neighbor.
# Run: ./scripts/peers_test.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1

fail() {
  echo "peers_test FAIL: $*" >&2
  exit 1
}

go test ./internal/discovery/ ./internal/loopback/ -count=1 >/dev/null || fail "unit tests failed"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"; [[ -n "${pid_a:-}" ]] && kill "$pid_a" 2>/dev/null || true; [[ -n "${pid_b:-}" ]] && kill "$pid_b" 2>/dev/null || true' EXIT
bin="$tmpdir/dudkad"
go build -o "$bin" ./cmd/dudkad || fail "go build failed"

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
  -announce-port "$port" -session-port 0 -announce-interval 150ms >"$log_a" 2>&1 &
pid_a=$!
"$bin" -data-dir "$tmpdir/b" -name "Bob" -listen "127.0.0.1:0" \
  -announce-port "$port" -session-port 0 -announce-interval 150ms >"$log_b" 2>&1 &
pid_b=$!

listen_a=""
listen_b=""
peer_a=""
peer_b=""
for _ in $(seq 1 50); do
  if grep -q '^ready ' "$log_a" 2>/dev/null && grep -q '^ready ' "$log_b" 2>/dev/null; then
    listen_a="$(grep '^listen=' "$log_a" | head -n 1 | sed 's/^listen=//')"
    listen_b="$(grep '^listen=' "$log_b" | head -n 1 | sed 's/^listen=//')"
    peer_a="$(sed -n 's/^peer_id=//p' "$log_a" | head -1)"
    peer_b="$(sed -n 's/^peer_id=//p' "$log_b" | head -1)"
    break
  fi
  sleep 0.1
done
[[ -n "$listen_a" && -n "$listen_b" ]] || fail "not ready; a=$(cat "$log_a") b=$(cat "$log_b")"

found=0
for _ in $(seq 1 60); do
  pa="$(curl -sS --max-time 1 "http://${listen_a}/peers" || true)"
  pb="$(curl -sS --max-time 1 "http://${listen_b}/peers" || true)"
  if python3 - "$pa" "$pb" "$peer_a" "$peer_b" <<'PY'
import json, sys
a, b, id_a, id_b = sys.argv[1:5]
try:
    ja, jb = json.loads(a), json.loads(b)
except Exception:
    raise SystemExit(1)
ids_a = {p.get("peer_id") for p in ja.get("peers") or []}
ids_b = {p.get("peer_id") for p in jb.get("peers") or []}
raise SystemExit(0 if (id_b in ids_a and id_a in ids_b) else 1)
PY
  then
    found=1
    break
  fi
  sleep 0.1
done
[[ "$found" -eq 1 ]] || fail "peers not mutual; a_peers=$(curl -sS "http://${listen_a}/peers") b_peers=$(curl -sS "http://${listen_b}/peers") logs a=$(cat "$log_a") b=$(cat "$log_b")"

kill "$pid_a" "$pid_b" 2>/dev/null || true
wait "$pid_a" 2>/dev/null || true
wait "$pid_b" 2>/dev/null || true
pid_a=""; pid_b=""

echo "peers_test OK"
