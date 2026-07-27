#!/usr/bin/env bash
# Task-level contract for P068: image thumbs in Flutter feed.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1
export PATH="/opt/homebrew/share/flutter/bin:${PATH:-}"

fail() {
  echo "flutter_thumbs_test FAIL: $*" >&2
  exit 1
}

command -v flutter >/dev/null 2>&1 || fail "flutter not on PATH"
command -v dart >/dev/null 2>&1 || fail "dart not on PATH"

grep -q 'thumb_b64' apps/dudka/lib/engine/client.dart || fail "thumb_b64 parsing missing"
grep -q 'file-thumb-' apps/dudka/lib/screens/chat_screen.dart || fail "file-thumb key missing"
grep -q 'FeedThumbKind' apps/dudka/lib/engine/client.dart || fail "FeedThumbKind missing"
grep -q 'HEIC' apps/dudka/lib/screens/chat_screen.dart || fail "HEIC mark missing"

(
  cd apps/dudka
  flutter pub get >/dev/null
  flutter test test/file_thumb_test.dart
) || fail "flutter thumb tests failed"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"; [[ -n "${pid:-}" ]] && kill "$pid" 2>/dev/null || true' EXIT
go build -o "$tmpdir/dudkad" ./cmd/dudkad || fail "build dudkad"

python3 - "$tmpdir/dot.jpg" <<'PY'
import sys
jpeg = bytes.fromhex(
  'ffd8ffe000104a46494600010100000100010000'
  'ffdb004300080606070605080707070909080a0c'
  '140d0c0b0b0c1912130f141d1a1f1e1d1a1c1c20'
  '242e2720222c231c1c2837292c30313434341f27'
  '393d38323c2e333432ffc0000b08000100010101'
  '1100ffc4001f0000010501010101010100000000'
  '000000000102030405060708090a0bffc400b510'
  '0002010303020403050504040000017d01020300'
  '041105122131410613516107227114328191a108'
  '2342b1c11552d1f02433627282090a161718191a'
  '25262728292a3435363738393a43444546474849'
  '4a535455565758595a636465666768696a737475'
  '767778797a838485868788898a92939495969798'
  '999aa2a3a4a5a6a7a8a9aab2b3b4b5b6b7b8b9ba'
  'c2c3c4c5c6c7c8c9cad2d3d4d5d6d7d8d9dae1e2'
  'e3e4e5e6e7e8e9eaf1f2f3f4f5f6f7f8f9faffda'
  '0008010100003f00fbd5db20a8f14500ffd9'
)
open(sys.argv[1], 'wb').write(jpeg)
PY

log="$tmpdir/d.log"
"$tmpdir/dudkad" -data-dir "$tmpdir/d" -name "ThumbPeer" -listen "127.0.0.1:0" \
  -announce-port 41791 -session-port 0 >"$log" 2>&1 &
pid=$!
listen=""
for _ in $(seq 1 50); do
  if grep -q '^ready ' "$log" 2>/dev/null; then
    listen="$(grep '^listen=' "$log" | head -n 1 | sed 's/^listen=//')"
    break
  fi
  sleep 0.1
done
[[ -n "$listen" ]] || fail "engine not ready: $(cat "$log")"

out="$(
  cd apps/dudka
  dart run tool/live_announce.dart "http://${listen}" "$tmpdir/dot.jpg" "image/jpeg"
)" || fail "announce jpeg failed: $out"
file_id="$(printf '%s\n' "$out" | sed -n 's/.*file_id=\([^ ]*\).*/\1/p')"
[[ -n "$file_id" ]] || fail "no file_id"

msgs="$(curl -sS --max-time 2 "http://${listen}/messages")"
python3 - "$msgs" "$file_id" <<'PY' || fail "messages missing thumb_b64"
import json, sys, base64
d = json.loads(sys.argv[1])
fid = sys.argv[2]
msgs = d.get("messages") or []
hit = next((m for m in msgs if m.get("file_id") == fid), None)
assert hit, msgs
b64 = hit.get("thumb_b64") or ""
assert b64, hit
raw = base64.b64decode(b64)
assert raw[:2] == b"\xff\xd8", raw[:8]
print("thumb_b64_ok", len(raw))
PY

kill "$pid" 2>/dev/null || true
wait "$pid" 2>/dev/null || true
pid=""

echo "flutter_thumbs_test OK"
