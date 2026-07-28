#!/usr/bin/env bash
# DUD-CHAT-121: nick and bounded history survive an engine/app replacement.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1

fail() {
  echo "reinstall_persistence_test FAIL: $*" >&2
  exit 1
}

tmpdir="$(mktemp -d)"
pid=""
trap '[[ -n "$pid" ]] && kill "$pid" 2>/dev/null || true; rm -rf "$tmpdir"' EXIT
bin="$tmpdir/dudkad"
data="$tmpdir/data"
go build -o "$bin" ./cmd/dudkad || fail "go build failed"

start_engine() {
  local log="$1"
  "$bin" -data-dir "$data" -listen "127.0.0.1:0" >"$log" 2>&1 &
  pid=$!
  for _ in $(seq 1 80); do
    if grep -q '^ready ' "$log" 2>/dev/null; then
      engine_listen="$(sed -n 's/^listen=//p' "$log" | head -1)"
      return
    fi
    if ! kill -0 "$pid" 2>/dev/null; then
      fail "engine exited: $(cat "$log")"
    fi
    sleep 0.1
  done
  fail "engine did not become ready: $(cat "$log")"
}

engine_listen=""
start_engine "$tmpdir/first.log"
listen="$engine_listen"
curl -fsS --max-time 2 -X POST "http://${listen}/nick" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Евгений"}' >/dev/null
curl -fsS --max-time 2 -X POST "http://${listen}/send" \
  -H 'Content-Type: application/json' \
  -d '{"text":"переживи переустановку"}' >/dev/null
[[ -s "$data/messages.json" ]] || fail "messages.json was not written"

kill "$pid"
wait "$pid" 2>/dev/null || true
pid=""

# Rebuild into the same executable path to model an application replacement.
go build -o "$bin" ./cmd/dudkad || fail "replacement build failed"
start_engine "$tmpdir/second.log"
listen="$engine_listen"
me="$(curl -fsS --max-time 2 "http://${listen}/me")"
messages="$(curl -fsS --max-time 2 "http://${listen}/messages")"
python3 - "$me" "$messages" <<'PY'
import json
import sys

me = json.loads(sys.argv[1])
messages = json.loads(sys.argv[2]).get("messages", [])
assert me.get("name") == "Евгений", me
assert any(m.get("text") == "переживи переустановку" for m in messages), messages
PY

echo "reinstall_persistence_test OK"
