#!/usr/bin/env bash
# Task-level contract for P015: GET /me JSON from loopback; foreign IP rejected.
# Run: ./scripts/me_test.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1

fail() {
  echo "me_test FAIL: $*" >&2
  exit 1
}

go test ./internal/loopback/ -count=1 >/dev/null || fail "loopback unit tests failed"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"; [[ -n "${pid:-}" ]] && kill "$pid" 2>/dev/null || true' EXIT
bin="$tmpdir/dudkad"
log="$tmpdir/out.log"
data="$tmpdir/data"
go build -o "$bin" ./cmd/dudkad || fail "go build failed"

"$bin" -data-dir "$data" -name "MeNick" -listen "127.0.0.1:0" >"$log" 2>&1 &
pid=$!

listen=""
peer=""
for _ in $(seq 1 50); do
  if grep -q '^ready ' "$log" 2>/dev/null; then
    listen="$(grep '^listen=' "$log" | head -n 1 | sed 's/^listen=//')"
    peer="$(sed -n 's/^peer_id=//p' "$log" | head -1)"
    break
  fi
  if ! kill -0 "$pid" 2>/dev/null; then
    fail "dudkad exited early; log: $(cat "$log")"
  fi
  sleep 0.1
done
[[ -n "$listen" && -n "$peer" ]] || fail "missing listen/peer; log: $(cat "$log")"

body="$tmpdir/me.json"
code="$(curl -sS -o "$body" -w '%{http_code}' --max-time 2 "http://${listen}/me")"
[[ "$code" == "200" ]] || fail "GET /me status=$code body=$(cat "$body")"

python3 - "$body" "$peer" <<'PY'
import json, sys
path, peer = sys.argv[1], sys.argv[2]
with open(path, encoding="utf-8") as f:
    d = json.load(f)
if d.get("peer_id") != peer:
    raise SystemExit(f"peer_id want {peer} got {d.get('peer_id')}")
if d.get("name") != "MeNick":
    raise SystemExit(f"name want MeNick got {d.get('name')}")
print("json_ok")
PY

# Binding is loopback-only: connecting via a non-loopback local IP must fail.
lan_ip="$(python3 - <<'PY'
import socket
s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
try:
    s.connect(("192.0.2.1", 80))  # TEST-NET-1, no packets needed
    ip = s.getsockname()[0]
finally:
    s.close()
print(ip if ip and not ip.startswith("127.") else "")
PY
)"
port="${listen##*:}"
if [[ -n "$lan_ip" ]]; then
  set +e
  curl -sS --connect-timeout 1 --max-time 1 "http://${lan_ip}:${port}/me" >/dev/null 2>&1
  curl_rc=$?
  set -e
  [[ "$curl_rc" -ne 0 ]] || fail "GET /me via LAN IP ${lan_ip} unexpectedly succeeded"
fi

kill "$pid" 2>/dev/null || true
wait "$pid" 2>/dev/null || true
pid=""

echo "me_test OK"
