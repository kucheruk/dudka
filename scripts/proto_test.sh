#!/usr/bin/env bash
# Task-level contract for P023: incompatible proto_major rejected; session intact.
# Run: ./scripts/proto_test.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1

fail() {
  echo "proto_test FAIL: $*" >&2
  exit 1
}

go test ./internal/discovery/ -count=1 -run 'Compatible|RegisterRejects' >/dev/null || fail "unit tests failed"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"; [[ -n "${pid:-}" ]] && kill "$pid" 2>/dev/null || true' EXIT
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

log="$tmpdir/out.log"
"$bin" -data-dir "$tmpdir/data" -name "Host" -listen "127.0.0.1:0" \
  -announce-port "$port" -session-port 0 -announce-interval 1h >"$log" 2>&1 &
pid=$!

listen=""
session=""
for _ in $(seq 1 50); do
  if grep -q '^ready ' "$log" 2>/dev/null; then
    listen="$(grep '^listen=' "$log" | head -n 1 | sed 's/^listen=//')"
    session="$(grep '^session_tcp=' "$log" | head -n 1 | sed 's/^session_tcp=//')"
    break
  fi
  if ! kill -0 "$pid" 2>/dev/null; then
    fail "dudkad exited; log=$(cat "$log")"
  fi
  sleep 0.1
done
[[ -n "$listen" && -n "$session" ]] || fail "not ready; log=$(cat "$log")"

# Seed a "good" peer via compatible register first.
python3 - "$session" <<'PY'
import json, socket, sys
port = int(sys.argv[1])
req = {
    "type": "register",
    "peer_id": "good-peer",
    "display_name": "Good",
    "proto_major": 1,
    "proto_minor": 0,
    "tcp_port": 1,
    "instance_id": "good-inst",
}
s = socket.create_connection(("127.0.0.1", port), timeout=2)
s.sendall((json.dumps(req) + "\n").encode())
line = s.makefile().readline()
s.close()
resp = json.loads(line)
assert resp.get("type") == "register_ok", resp
print("good_ok")
PY

# Incompatible register must be rejected.
python3 - "$session" <<'PY'
import json, socket, sys
port = int(sys.argv[1])
req = {
    "type": "register",
    "peer_id": "alien-peer",
    "display_name": "Alien",
    "proto_major": 99,
    "proto_minor": 0,
    "tcp_port": 9,
    "instance_id": "alien-inst",
}
s = socket.create_connection(("127.0.0.1", port), timeout=2)
s.sendall((json.dumps(req) + "\n").encode())
line = s.makefile().readline()
s.close()
resp = json.loads(line)
assert resp.get("type") == "register_reject", resp
assert resp.get("reason") == "proto_major_mismatch", resp
print("reject_ok")
PY

peers="$(curl -sS --max-time 2 "http://${listen}/peers")"
python3 - "$peers" <<'PY'
import json, sys
d = json.loads(sys.argv[1])
ids = {p.get("peer_id") for p in d.get("peers") or []}
assert "good-peer" in ids, d
assert "alien-peer" not in ids, d
print("peers_ok")
PY

status="$(curl -sS --max-time 2 "http://${listen}/status")"
python3 - "$status" <<'PY'
import json, sys
d = json.loads(sys.argv[1])
assert d.get("proto_major") == 1, d
inc = d.get("incompatible") or []
assert any(p.get("peer_id") == "alien-peer" for p in inc), d
print("status_ok")
PY

grep -q 'proto_mismatch peer_id=alien-peer' "$log" || fail "missing proto_mismatch log; $(cat "$log")"

kill "$pid" 2>/dev/null || true
wait "$pid" 2>/dev/null || true
pid=""

echo "proto_test OK"
