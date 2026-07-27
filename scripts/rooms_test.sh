#!/usr/bin/env bash
# P097: channels API
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
tmpdir="$(mktemp -d)"
trap 'kill $(jobs -p) 2>/dev/null || true; rm -rf "$tmpdir"' EXIT
go build -o "$tmpdir/dudkad" ./cmd/dudkad
"$tmpdir/dudkad" -data-dir "$tmpdir/d" -name "Вася" -listen 127.0.0.1:17911 \
  -announce-port 0 -session-port 0 >/dev/null 2>&1 &
for _ in $(seq 1 40); do curl -sf http://127.0.0.1:17911/health >/dev/null && break; sleep 0.1; done
curl -sf -X POST http://127.0.0.1:17911/channels -d '{"name":"кухня"}' >/dev/null
curl -sf -X POST http://127.0.0.1:17911/send -d '{"text":"hi","channel":"кухня"}' >/dev/null
curl -sf -X POST http://127.0.0.1:17911/send -d '{"text":"общий-msg"}' >/dev/null
n_k=$(curl -sf 'http://127.0.0.1:17911/messages?channel=%D0%BA%D1%83%D1%85%D0%BD%D1%8F' | python3 -c 'import sys,json; print(len(json.load(sys.stdin)["messages"]))')
n_o=$(curl -sf 'http://127.0.0.1:17911/messages?channel=%D0%BE%D0%B1%D1%89%D0%B8%D0%B9' | python3 -c 'import sys,json; print(len(json.load(sys.stdin)["messages"]))')
chs=$(curl -sf http://127.0.0.1:17911/channels)
echo "$chs" | grep -q 'кухня'
[[ "$n_k" -ge 1 && "$n_o" -ge 1 ]] || { echo "rooms_test FAIL n_k=$n_k n_o=$n_o"; exit 1; }
echo "rooms_test OK"
