#!/usr/bin/env bash
# Task-level contract for P030: POST /send → second peer GET /messages ≤ 2s.
# Run: ./scripts/send_test.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1

fail() {
  echo "send_test FAIL: $*" >&2
  exit 1
}

go test ./internal/chat/ ./internal/discovery/ ./internal/loopback/ -count=1 >/dev/null || fail "unit tests failed"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"; [[ -n "${pid_a:-}" ]] && kill "$pid_a" 2>/dev/null || true; [[ -n "${pid_b:-}" ]] && kill "$pid_b" 2>/dev/null || true' EXIT
bin="$tmpdir/dudkad"
go build -o "$bin" ./cmd/dudkad || fail "go build failed"

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
  -announce-port "$port_a" -announce-target "127.0.0.1:${port_b}" \
  -session-port 0 -announce-interval 150ms >"$log_a" 2>&1 &
pid_a=$!
"$bin" -data-dir "$tmpdir/b" -name "Bob" -listen "127.0.0.1:0" \
  -announce-port "$port_b" -announce-target "127.0.0.1:${port_a}" \
  -session-port 0 -announce-interval 150ms >"$log_b" 2>&1 &
pid_b=$!

listen_a=""
listen_b=""
peer_a=""
for _ in $(seq 1 50); do
  if grep -q '^ready ' "$log_a" 2>/dev/null && grep -q '^ready ' "$log_b" 2>/dev/null; then
    listen_a="$(grep '^listen=' "$log_a" | head -n 1 | sed 's/^listen=//')"
    listen_b="$(grep '^listen=' "$log_b" | head -n 1 | sed 's/^listen=//')"
    peer_a="$(sed -n 's/^peer_id=//p' "$log_a" | head -1)"
    break
  fi
  sleep 0.1
done
[[ -n "$listen_a" && -n "$listen_b" ]] || fail "not ready; a=$(cat "$log_a") b=$(cat "$log_b")"

# Wait until peers see each other.
found=0
for _ in $(seq 1 60); do
  pa="$(curl -sS --max-time 1 "http://${listen_a}/peers" || true)"
  pb="$(curl -sS --max-time 1 "http://${listen_b}/peers" || true)"
  if python3 - "$pa" "$pb" <<'PY'
import json, sys
try:
    ja, jb = json.loads(sys.argv[1]), json.loads(sys.argv[2])
except Exception:
    raise SystemExit(1)
raise SystemExit(0 if (ja.get("peers") and jb.get("peers")) else 1)
PY
  then
    found=1
    break
  fi
  sleep 0.1
done
[[ "$found" -eq 1 ]] || fail "peers empty; a=$(curl -sS "http://${listen_a}/peers") b=$(curl -sS "http://${listen_b}/peers")"

send_body='{"text":"p030 hello from alice"}'
curl -sS --max-time 2 -X POST "http://${listen_a}/send" \
  -H 'Content-Type: application/json' \
  -d "$send_body" >"$tmpdir/send.json" || fail "POST /send failed"
python3 - "$tmpdir/send.json" <<'PY' || fail "send not accepted/queued"
import json, sys
d = json.load(open(sys.argv[1]))
assert d.get("status") in ("accepted", "queued"), d
assert "queued" in d, d
assert "delivered" not in json.dumps(d).lower(), d
assert d.get("message", {}).get("text") == "p030 hello from alice", d
PY

# Bob must see the message within 2s.
seen=0
deadline_ms=2000
step_ms=50
steps=$((deadline_ms / step_ms))
for _ in $(seq 1 "$steps"); do
  mb="$(curl -sS --max-time 1 "http://${listen_b}/messages" || true)"
  if python3 - "$mb" "$peer_a" <<'PY'
import json, sys
try:
    data = json.loads(sys.argv[1])
except Exception:
    raise SystemExit(1)
want_peer = sys.argv[2]
for m in data.get("messages") or []:
    if m.get("text") == "p030 hello from alice" and m.get("peer_id") == want_peer:
        raise SystemExit(0)
raise SystemExit(1)
PY
  then
    seen=1
    break
  fi
  sleep 0.05
done
[[ "$seen" -eq 1 ]] || fail "Bob missing message ≤2s; messages=$(curl -sS "http://${listen_b}/messages") logs a=$(cat "$log_a") b=$(cat "$log_b")"

kill "$pid_a" "$pid_b" 2>/dev/null || true
wait "$pid_a" 2>/dev/null || true
wait "$pid_b" 2>/dev/null || true
pid_a=""; pid_b=""

echo "send_test OK"
