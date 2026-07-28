#!/usr/bin/env bash
# Task-level contract for P067: Flutter announce/fetch with progress + cancel.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1
export PATH="/opt/homebrew/share/flutter/bin:${PATH:-}"

fail() {
  echo "flutter_files_test FAIL: $*" >&2
  exit 1
}

command -v flutter >/dev/null 2>&1 || fail "flutter not on PATH"
command -v dart >/dev/null 2>&1 || fail "dart not on PATH"

grep -q 'announceFile' apps/dudka/lib/engine/client.dart || fail "announceFile missing"
grep -q 'startFetch' apps/dudka/lib/engine/client.dart || fail "startFetch missing"
grep -q 'cancelFetch' apps/dudka/lib/engine/client.dart || fail "cancelFetch missing"
grep -q 'chat-file' apps/dudka/lib/screens/chat_screen.dart || fail "ФАЙЛ button missing"
grep -q 'СКАЧАТЬ' apps/dudka/lib/screens/chat_screen.dart || fail "СКАЧАТЬ missing"
grep -q 'ОТМЕНА' apps/dudka/lib/screens/chat_screen.dart || fail "ОТМЕНА missing"

(
  cd apps/dudka
  flutter pub get >/dev/null
  flutter test test/file_transfer_test.dart test/chat_screen_test.dart
) || fail "flutter file widget/unit tests failed"

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
"$tmpdir/dudkad" -data-dir "$tmpdir/a" -name "FlutterSrc" -listen "127.0.0.1:0" \
  -announce-port "$port" -session-port 0 -announce-interval 150ms >"$log_a" 2>&1 &
pid_a=$!
"$tmpdir/dudkad" -data-dir "$tmpdir/b" -name "FlutterDst" -listen "127.0.0.1:0" \
  -announce-port "$port" -session-port 0 -announce-interval 150ms >"$log_b" 2>&1 &
pid_b=$!

listen_a=""; listen_b=""; session_a=""; session_b=""
for _ in $(seq 1 50); do
  if grep -q '^ready ' "$log_a" 2>/dev/null && grep -q '^ready ' "$log_b" 2>/dev/null; then
    listen_a="$(grep '^listen=' "$log_a" | head -n 1 | sed 's/^listen=//')"
    listen_b="$(grep '^listen=' "$log_b" | head -n 1 | sed 's/^listen=//')"
    session_a="$(grep '^session_tcp=' "$log_a" | head -n 1 | sed 's/^session_tcp=//')"
    session_b="$(grep '^session_tcp=' "$log_b" | head -n 1 | sed 's/^session_tcp=//')"
    break
  fi
  sleep 0.1
done
[[ -n "$listen_a" && -n "$listen_b" && -n "$session_a" && -n "$session_b" ]] || fail "engines not ready"

# Make this file-transfer gate deterministic on a single macOS host. Real UDP
# broadcast remains covered by the separate LAN smoke.
curl -fsS --max-time 5 -X POST "http://${listen_a}/scan" \
  -H 'Content-Type: application/json' \
  -d "{\"hosts\":[\"127.0.0.1\"],\"port\":${session_b}}" >/dev/null \
  || fail "source could not register destination"
curl -fsS --max-time 5 -X POST "http://${listen_b}/scan" \
  -H 'Content-Type: application/json' \
  -d "{\"hosts\":[\"127.0.0.1\"],\"port\":${session_a}}" >/dev/null \
  || fail "destination could not register source"

peers_ready=0
for _ in $(seq 1 60); do
  pa="$(curl -sS --max-time 1 "http://${listen_a}/peers" || true)"
  pb="$(curl -sS --max-time 1 "http://${listen_b}/peers" || true)"
  if python3 - "$pa" "$pb" <<'PY'
import json, sys
ja, jb = json.loads(sys.argv[1]), json.loads(sys.argv[2])
raise SystemExit(0 if (ja.get("peers") and jb.get("peers")) else 1)
PY
  then peers_ready=1; break; fi
  sleep 0.1
done
[[ "$peers_ready" == 1 ]] || fail "engines did not discover each other"

# Large-ish payload so progress can be observed before cancel.
python3 - <<PY
p="$tmpdir/blob.bin"
open(p,"wb").write(bytes([i % 256 for i in range(256*1024)]))
print(p)
PY
blob="$tmpdir/blob.bin"

out="$(
  cd apps/dudka
  dart run tool/live_announce.dart "http://${listen_a}" "$blob" "application/octet-stream"
)" || fail "announce failed: $out"
file_id="$(printf '%s\n' "$out" | sed -n 's/.*file_id=\([^ ]*\).*/\1/p')"
[[ -n "$file_id" ]] || fail "no file_id in $out"

# Wait announce visible on B.
announce_visible=0
for _ in $(seq 1 40); do
  msgs="$(curl -sS --max-time 1 "http://${listen_b}/messages" || true)"
  if printf '%s\n' "$msgs" | grep -Fq "$file_id"; then announce_visible=1; break; fi
  sleep 0.1
done
[[ "$announce_visible" == 1 ]] || fail "first announce did not reach destination"
curl -fsS --max-time 5 -X POST "http://${listen_b}/scan" \
  -H 'Content-Type: application/json' \
  -d "{\"hosts\":[\"127.0.0.1\"],\"port\":${session_a}}" >/dev/null \
  || fail "destination lost source before cancel fetch"

(
  cd apps/dudka
  dart run tool/live_fetch.dart "http://${listen_b}" "$file_id" --cancel
) || fail "cancel fetch path failed"

# Fresh announce + full download.
# The 150ms test interval implies a 750ms peer TTL; refresh explicit
# registration after the cancel scenario so macOS broadcast fan-out is not a
# hidden requirement for the second announce.
curl -fsS --max-time 5 -X POST "http://${listen_a}/scan" \
  -H 'Content-Type: application/json' \
  -d "{\"hosts\":[\"127.0.0.1\"],\"port\":${session_b}}" >/dev/null \
  || fail "source could not re-register destination"
curl -fsS --max-time 5 -X POST "http://${listen_b}/scan" \
  -H 'Content-Type: application/json' \
  -d "{\"hosts\":[\"127.0.0.1\"],\"port\":${session_a}}" >/dev/null \
  || fail "destination could not re-register source"
out2="$(
  cd apps/dudka
  dart run tool/live_announce.dart "http://${listen_a}" "$blob" "application/octet-stream"
)" || fail "announce2 failed"
file_id2="$(printf '%s\n' "$out2" | sed -n 's/.*file_id=\([^ ]*\).*/\1/p')"
announce2_visible=0
for _ in $(seq 1 40); do
  msgs="$(curl -sS --max-time 1 "http://${listen_b}/messages" || true)"
  if printf '%s\n' "$msgs" | grep -Fq "$file_id2"; then announce2_visible=1; break; fi
  sleep 0.1
done
[[ "$announce2_visible" == 1 ]] || fail "second announce did not reach destination"
curl -fsS --max-time 5 -X POST "http://${listen_b}/scan" \
  -H 'Content-Type: application/json' \
  -d "{\"hosts\":[\"127.0.0.1\"],\"port\":${session_a}}" >/dev/null \
  || fail "destination lost source before full fetch"
(
  cd apps/dudka
  dart run tool/live_fetch.dart "http://${listen_b}" "$file_id2" --wait
) || fail "full fetch failed"

kill "$pid_a" "$pid_b" 2>/dev/null || true
wait "$pid_a" 2>/dev/null || true
wait "$pid_b" 2>/dev/null || true
pid_a=""; pid_b=""

echo "flutter_files_test OK"
