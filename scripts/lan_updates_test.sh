#!/usr/bin/env bash
# P099: offline update blob share via loopback /updates (LAN peer can fetch file via file announce separately)
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
tmpdir="$(mktemp -d)"
trap 'kill $(jobs -p) 2>/dev/null || true; rm -rf "$tmpdir"' EXIT
go build -o "$tmpdir/dudkad" ./cmd/dudkad
"$tmpdir/dudkad" -data-dir "$tmpdir/d" -name "Pack" -listen 127.0.0.1:17931 \
  -announce-port 0 -session-port 0 >/dev/null 2>&1 &
for _ in $(seq 1 40); do curl -sf http://127.0.0.1:17931/health >/dev/null && break; sleep 0.1; done
b64=$(printf 'dudka-update-v1' | base64)
curl -sf -X POST http://127.0.0.1:17931/updates -d "{\"name\":\"dudka-note.txt\",\"content_b64\":\"$b64\"}" >/dev/null
curl -sf http://127.0.0.1:17931/updates | grep -q dudka-note.txt
[[ -f "$tmpdir/d/updates/dudka-note.txt" ]] || { echo "lan_updates_test FAIL missing file"; exit 1; }
echo "lan_updates_test OK"
