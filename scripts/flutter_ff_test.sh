#!/usr/bin/env bash
# Task-level contract for P071: Flutter↔Flutter text+file on two peers (two devices).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1
export PATH="/opt/homebrew/share/flutter/bin:${PATH:-}"

fail() {
  echo "flutter_ff_test FAIL: $*" >&2
  exit 1
}

command -v flutter >/dev/null 2>&1 || fail "flutter not on PATH"
command -v dart >/dev/null 2>&1 || fail "dart not on PATH"

grep -q 'sendText' apps/dudka/lib/engine/client.dart || fail "EngineClient.sendText missing"
grep -q 'announceFile' apps/dudka/lib/engine/client.dart || fail "announceFile missing"
grep -q 'startFetch' apps/dudka/lib/engine/client.dart || fail "startFetch missing"
[[ -f apps/dudka/tool/live_wait_text.dart ]] || fail "live_wait_text.dart missing"
[[ -f apps/dudka/tool/live_send.dart ]] || fail "live_send.dart missing"
[[ -f apps/dudka/tool/live_announce.dart ]] || fail "live_announce.dart missing"
[[ -f apps/dudka/tool/live_fetch.dart ]] || fail "live_fetch.dart missing"

(
  cd apps/dudka
  flutter pub get >/dev/null
) || fail "flutter pub get failed"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"; for p in ${pid_a:-} ${pid_b:-}; do [[ -n "$p" ]] && kill "$p" 2>/dev/null || true; done' EXIT
go build -o "$tmpdir/dudkad" ./cmd/dudkad || fail "build dudkad"

port="$(python3 - <<'PY'
import socket
s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
)"

log_a="$tmpdir/a.log"; log_b="$tmpdir/b.log"
"$tmpdir/dudkad" -data-dir "$tmpdir/a" -name "FlutterA" -listen "127.0.0.1:0" \
  -announce-port "$port" -session-port 0 -announce-interval 150ms >"$log_a" 2>&1 &
pid_a=$!
"$tmpdir/dudkad" -data-dir "$tmpdir/b" -name "FlutterB" -listen "127.0.0.1:0" \
  -announce-port "$port" -session-port 0 -announce-interval 150ms >"$log_b" 2>&1 &
pid_b=$!

listen_a=""; listen_b=""
for _ in $(seq 1 50); do
  if grep -q '^ready ' "$log_a" 2>/dev/null && grep -q '^ready ' "$log_b" 2>/dev/null; then
    listen_a="$(grep '^listen=' "$log_a" | head -n 1 | sed 's/^listen=//')"
    listen_b="$(grep '^listen=' "$log_b" | head -n 1 | sed 's/^listen=//')"
    break
  fi
  sleep 0.1
done
[[ -n "$listen_a" && -n "$listen_b" ]] || fail "engines not ready"

for _ in $(seq 1 60); do
  pa="$(curl -sS --max-time 1 "http://${listen_a}/peers" || true)"
  pb="$(curl -sS --max-time 1 "http://${listen_b}/peers" || true)"
  if python3 - "$pa" "$pb" <<'PY'
import json, sys
ja, jb = json.loads(sys.argv[1]), json.loads(sys.argv[2])
raise SystemExit(0 if (ja.get("peers") and jb.get("peers")) else 1)
PY
  then break; fi
  sleep 0.1
done

# --- text Flutter A → Flutter B ---
MSG_AB="ff-ab $(date +%s)-$$"
(
  cd apps/dudka
  dart run tool/live_send.dart "http://${listen_a}" "$MSG_AB"
) || fail "Flutter A sendText failed"
(
  cd apps/dudka
  dart run tool/live_wait_text.dart "http://${listen_b}" "$MSG_AB" --timeout-ms 8000
) || fail "Flutter B missing A text"

# --- text Flutter B → Flutter A ---
MSG_BA="ff-ba $(date +%s)-$$"
(
  cd apps/dudka
  dart run tool/live_send.dart "http://${listen_b}" "$MSG_BA"
) || fail "Flutter B sendText failed"
(
  cd apps/dudka
  dart run tool/live_wait_text.dart "http://${listen_a}" "$MSG_BA" --timeout-ms 8000
) || fail "Flutter A missing B text"

# --- file Flutter A announce → Flutter B fetch ---
python3 - <<PY
p="$tmpdir/ff-payload.bin"
open(p,"wb").write(bytes([i % 256 for i in range(64 * 1024)]))
PY
blob="$tmpdir/ff-payload.bin"
out="$(
  cd apps/dudka
  dart run tool/live_announce.dart "http://${listen_a}" "$blob" "application/octet-stream"
)" || fail "Flutter A announce failed: $out"
file_id="$(printf '%s\n' "$out" | sed -n 's/.*file_id=\([^ ]*\).*/\1/p')"
[[ -n "$file_id" ]] || fail "no file_id in $out"

(
  cd apps/dudka
  dart run tool/live_wait_text.dart "http://${listen_b}" "$file_id" --timeout-ms 8000
) || fail "Flutter B missing file announce"

(
  cd apps/dudka
  dart run tool/live_fetch.dart "http://${listen_b}" "$file_id" --wait
) || fail "Flutter B fetch failed"

kill "$pid_a" "$pid_b" 2>/dev/null || true
wait "$pid_a" 2>/dev/null || true
wait "$pid_b" 2>/dev/null || true
pid_a=""; pid_b=""

echo "flutter_ff_test OK"
