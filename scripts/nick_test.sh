#!/usr/bin/env bash
# Task-level contract for P016: POST /nick updates GET /me (and disk).
# Run: ./scripts/nick_test.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1

fail() {
  echo "nick_test FAIL: $*" >&2
  exit 1
}

go test ./internal/loopback/ -count=1 -run Nick >/dev/null || fail "nick unit tests failed"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"; [[ -n "${pid:-}" ]] && kill "$pid" 2>/dev/null || true' EXIT
bin="$tmpdir/dudkad"
log="$tmpdir/out.log"
data="$tmpdir/data"
go build -o "$bin" ./cmd/dudkad || fail "go build failed"

"$bin" -data-dir "$data" -name "Старый" -listen "127.0.0.1:0" >"$log" 2>&1 &
pid=$!

listen=""
for _ in $(seq 1 50); do
  if grep -q '^ready ' "$log" 2>/dev/null; then
    listen="$(grep '^listen=' "$log" | head -n 1 | sed 's/^listen=//')"
    break
  fi
  if ! kill -0 "$pid" 2>/dev/null; then
    fail "dudkad exited early; log: $(cat "$log")"
  fi
  sleep 0.1
done
[[ -n "$listen" ]] || fail "no listen; log: $(cat "$log")"

code="$(curl -sS -o "$tmpdir/nick.json" -w '%{http_code}' --max-time 2 \
  -X POST "http://${listen}/nick" \
  -H 'Content-Type: application/json' \
  -d '{"name":"НовыйНик"}')"
[[ "$code" == "200" ]] || fail "POST /nick status=$code body=$(cat "$tmpdir/nick.json")"

me="$(curl -sS --max-time 2 "http://${listen}/me")"
python3 - "$me" <<'PY'
import json, sys
d = json.loads(sys.argv[1])
if d.get("name") != "НовыйНик":
    raise SystemExit(f"GET /me name={d.get('name')!r} want НовыйНик")
print("me_ok")
PY

disk="$(tr -d '[:space:]' <"$data/display_name")"
[[ "$disk" == "НовыйНик" ]] || fail "disk display_name=$disk want НовыйНик"

# Restart must keep the new nick (no -name override).
kill "$pid" 2>/dev/null || true
wait "$pid" 2>/dev/null || true
pid=""
log2="$tmpdir/out2.log"
"$bin" -data-dir "$data" -listen "127.0.0.1:0" >"$log2" 2>&1 &
pid=$!
listen2=""
for _ in $(seq 1 50); do
  if grep -q '^ready ' "$log2" 2>/dev/null; then
    listen2="$(grep '^listen=' "$log2" | head -n 1 | sed 's/^listen=//')"
    break
  fi
  if ! kill -0 "$pid" 2>/dev/null; then
    fail "restart exited early; log: $(cat "$log2")"
  fi
  sleep 0.1
done
me2="$(curl -sS --max-time 2 "http://${listen2}/me")"
python3 - "$me2" <<'PY'
import json, sys
d = json.loads(sys.argv[1])
if d.get("name") != "НовыйНик":
    raise SystemExit(f"after restart name={d.get('name')!r}")
print("persist_ok")
PY

kill "$pid" 2>/dev/null || true
wait "$pid" 2>/dev/null || true
pid=""

echo "nick_test OK"
