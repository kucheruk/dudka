#!/usr/bin/env bash
# P025 / DUD-NET-101: public seed IP from flags must log wan_refuse and not hang on WAN.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"; kill "$PID" 2>/dev/null || true' EXIT

go build -o "$TMP/dudkad" ./cmd/dudkad

"$TMP/dudkad" \
  -data-dir "$TMP/data" \
  -name Guard \
  -listen "127.0.0.1:0" \
  -announce-port 0 \
  -session-port 0 \
  -announce-interval 1h \
  -dial-hosts "8.8.8.8,1.1.1.1" \
  >"$TMP/out" 2>&1 &
PID=$!

for i in $(seq 1 50); do
  if grep -q 'wan_refuse' "$TMP/out" 2>/dev/null; then
    break
  fi
  sleep 0.1
done

if ! grep -q 'wan_refuse host=8.8.8.8' "$TMP/out"; then
  echo "FAIL: missing wan_refuse for 8.8.8.8" >&2
  cat "$TMP/out" >&2 || true
  exit 1
fi
if ! grep -q 'wan_refuse host=1.1.1.1' "$TMP/out"; then
  echo "FAIL: missing wan_refuse for 1.1.1.1" >&2
  cat "$TMP/out" >&2 || true
  exit 1
fi

echo "wan_test OK"
