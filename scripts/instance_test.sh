#!/usr/bin/env bash
# Task-level contract for P022: restart → peer_updated, no zombie duplicates.
# Run: ./scripts/instance_test.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1

fail() {
  echo "instance_test FAIL: $*" >&2
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
log_b="$tmpdir/b1.log"
"$bin" -data-dir "$tmpdir/a" -name "Alice" -listen "127.0.0.1:0" \
  -announce-port "$port" -session-port 0 -announce-interval 120ms >"$log_a" 2>&1 &
pid_a=$!
"$bin" -data-dir "$tmpdir/b" -name "Bob" -listen "127.0.0.1:0" \
  -announce-port "$port" -session-port 0 -announce-interval 120ms >"$log_b" 2>&1 &
pid_b=$!

listen_a=""
peer_b=""
inst_b1=""
for _ in $(seq 1 50); do
  if grep -q '^ready ' "$log_a" 2>/dev/null && grep -q '^ready ' "$log_b" 2>/dev/null; then
    listen_a="$(grep '^listen=' "$log_a" | head -n 1 | sed 's/^listen=//')"
    peer_b="$(sed -n 's/^peer_id=//p' "$log_b" | head -1)"
    inst_b1="$(sed -n 's/^instance_id=//p' "$log_b" | head -1)"
    break
  fi
  sleep 0.1
done
[[ -n "$listen_a" && -n "$peer_b" ]] || fail "not ready"

# Wait until Alice knows Bob.
for _ in $(seq 1 60); do
  if curl -sS --max-time 1 "http://${listen_a}/peers" | grep -q "$peer_b"; then
    break
  fi
  sleep 0.1
done

kill "$pid_b" 2>/dev/null || true
wait "$pid_b" 2>/dev/null || true
pid_b=""

log_b2="$tmpdir/b2.log"
"$bin" -data-dir "$tmpdir/b" -name "Bob" -listen "127.0.0.1:0" \
  -announce-port "$port" -session-port 0 -announce-interval 120ms >"$log_b2" 2>&1 &
pid_b=$!

inst_b2=""
for _ in $(seq 1 50); do
  if grep -q '^ready ' "$log_b2" 2>/dev/null; then
    inst_b2="$(sed -n 's/^instance_id=//p' "$log_b2" | head -1)"
    break
  fi
  sleep 0.1
done
[[ -n "$inst_b2" && "$inst_b2" != "$inst_b1" ]] || fail "instance_id did not change on restart ($inst_b1 / $inst_b2)"

ok=0
for _ in $(seq 1 80); do
  body="$(curl -sS --max-time 1 "http://${listen_a}/peers" || true)"
  if python3 - "$body" "$peer_b" "$inst_b2" <<'PY'
import json, sys
body, peer_b, inst = sys.argv[1:4]
try:
    d = json.loads(body)
except Exception:
    raise SystemExit(1)
peers = d.get("peers") or []
if len(peers) != 1:
    raise SystemExit(1)
p = peers[0]
if p.get("peer_id") != peer_b:
    raise SystemExit(1)
if p.get("instance_id") != inst:
    raise SystemExit(1)
if not p.get("updated"):
    raise SystemExit(1)
raise SystemExit(0)
PY
  then
    ok=1
    break
  fi
  sleep 0.1
done
[[ "$ok" -eq 1 ]] || fail "Alice peers not updated; body=$(curl -sS "http://${listen_a}/peers") log=$(cat "$log_a")"

grep -q "peer_updated peer_id=${peer_b}" "$log_a" || fail "missing peer_updated log; a=$(cat "$log_a")"

kill "$pid_a" "$pid_b" 2>/dev/null || true
wait "$pid_a" 2>/dev/null || true
wait "$pid_b" 2>/dev/null || true
pid_a=""; pid_b=""

echo "instance_test OK"
