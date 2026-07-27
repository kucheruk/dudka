#!/usr/bin/env bash
# P098: optional want_ack yields ack message type
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
tmpdir="$(mktemp -d)"
trap 'kill $(jobs -p) 2>/dev/null || true; rm -rf "$tmpdir"' EXIT
go build -o "$tmpdir/dudkad" ./cmd/dudkad
ANN=41781
"$tmpdir/dudkad" -data-dir "$tmpdir/a" -name "A" -listen 127.0.0.1:17921 \
  -announce-port "$ANN" -session-port 0 -announce-interval 200ms >"$tmpdir/a.log" 2>&1 &
"$tmpdir/dudkad" -data-dir "$tmpdir/b" -name "B" -listen 127.0.0.1:17922 \
  -announce-port "$ANN" -session-port 0 -announce-interval 200ms >"$tmpdir/b.log" 2>&1 &
for _ in $(seq 1 50); do
  pa=$(curl -sf http://127.0.0.1:17921/peers | python3 -c 'import sys,json; print(len(json.load(sys.stdin).get("peers",[])))' 2>/dev/null || echo 0)
  [[ "$pa" -ge 1 ]] && break
  sleep 0.15
done
[[ "$pa" -ge 1 ]] || { echo "ack_test FAIL no peers"; exit 1; }
curl -sf -X POST http://127.0.0.1:17921/send -d '{"text":"ping-ack","want_ack":true}' >/dev/null
ok=0
for _ in $(seq 1 40); do
  # Receiver (B) emits local ack; fan-out to sender is best-effort.
  if curl -sf http://127.0.0.1:17922/messages | python3 -c 'import sys,json; ms=json.load(sys.stdin)["messages"];
sys.exit(0 if any(m.get("type")=="ack" and m.get("ack_for") for m in ms) else 1)'; then
    ok=1; break
  fi
  sleep 0.1
done
[[ "$ok" -eq 1 ]] || { echo "ack_test FAIL no ack"; exit 1; }
echo "ack_test OK"
