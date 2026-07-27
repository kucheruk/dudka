#!/usr/bin/env bash
# Task-level contract for P031: oversized POST /send → 4xx + clear error.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1

fail() {
  echo "send_length_test FAIL: $*" >&2
  exit 1
}

go test ./internal/chat/ ./internal/loopback/ -run 'OverMax|Oversized|ExactMax' -count=1 >/dev/null \
  || fail "unit tests failed"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"; [[ -n "${pid:-}" ]] && kill "$pid" 2>/dev/null || true' EXIT
bin="$tmpdir/dudkad"
go build -o "$bin" ./cmd/dudkad || fail "go build failed"

log="$tmpdir/out.log"
"$bin" -data-dir "$tmpdir/data" -name "Len" -listen "127.0.0.1:0" \
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
[[ -n "$listen" ]] || fail "not ready: $(cat "$log")"

python3 - "$tmpdir/payload.json" <<'PY'
import json, sys
with open(sys.argv[1], "w", encoding="utf-8") as f:
    json.dump({"text": "a" * 4001}, f)
PY

code="$(curl -sS -o "$tmpdir/body.txt" -w "%{http_code}" --max-time 2 \
  -X POST "http://${listen}/send" \
  -H 'Content-Type: application/json' \
  --data-binary @"$tmpdir/payload.json")"

[[ "$code" =~ ^4[0-9][0-9]$ ]] || fail "want 4xx got $code body=$(cat "$tmpdir/body.txt")"
grep -q '4000' "$tmpdir/body.txt" || fail "body missing limit: $(cat "$tmpdir/body.txt")"

msgs="$(curl -sS --max-time 1 "http://${listen}/messages")"
python3 - "$msgs" <<'PY' || fail "oversized stored: $msgs"
import json, sys
data = json.loads(sys.argv[1])
assert len(data.get("messages") or []) == 0
PY

kill "$pid" 2>/dev/null || true
wait "$pid" 2>/dev/null || true
pid=""

echo "send_length_test OK"
