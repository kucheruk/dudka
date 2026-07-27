#!/usr/bin/env bash
# Task-level contract for P064: Flutter compose «ОТПРАВИТЬ» → TUI peer sees text.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1
export PATH="/opt/homebrew/share/flutter/bin:${PATH:-}"

fail() {
  echo "flutter_send_test FAIL: $*" >&2
  exit 1
}

command -v flutter >/dev/null 2>&1 || fail "flutter not on PATH"
command -v dart >/dev/null 2>&1 || fail "dart not on PATH"

grep -q 'ОТПРАВИТЬ' apps/dudka/lib/screens/chat_screen.dart || fail "ОТПРАВИТЬ button missing"
grep -q 'sendText' apps/dudka/lib/engine/client.dart || fail "EngineClient.sendText missing"

(
  cd apps/dudka
  flutter pub get >/dev/null
  flutter test test/compose_send_test.dart test/chat_screen_test.dart
) || fail "flutter compose unit/widget tests failed"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"; for p in ${pid_a:-} ${pid_b:-}; do [[ -n "$p" ]] && kill "$p" 2>/dev/null || true; done' EXIT
go build -o "$tmpdir/dudkad" ./cmd/dudkad || fail "build dudkad"
go build -o "$tmpdir/dudka" ./cmd/dudka || fail "build dudka"

port="$(python3 - <<'PY'
import socket
s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
)"

log_a="$tmpdir/a.log"; log_b="$tmpdir/b.log"
"$tmpdir/dudkad" -data-dir "$tmpdir/a" -name "FlutterPeer" -listen "127.0.0.1:0" \
  -announce-port "$port" -session-port 0 -announce-interval 150ms >"$log_a" 2>&1 &
pid_a=$!
"$tmpdir/dudkad" -data-dir "$tmpdir/b" -name "TuiPeer" -listen "127.0.0.1:0" \
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

MSG="привет из flutter $(date +%s)"
(
  cd apps/dudka
  dart run tool/live_send.dart "http://${listen_a}" "$MSG"
) || fail "Flutter EngineClient.sendText failed"

seen=0
for _ in $(seq 1 40); do
  frame_b="$("$tmpdir/dudka" -engine "$listen_b")"
  if printf '%s\n' "$frame_b" | grep -Fq "$MSG"; then
    seen=1
    break
  fi
  sleep 0.1
done
[[ "$seen" -eq 1 ]] || fail "TUI missing Flutter text: $($tmpdir/dudka -engine "$listen_b")"

# Reply TUI → Flutter peer messages.
"$tmpdir/dudka" -engine "$listen_b" -send "ответ tui" >/dev/null
seen_a=0
for _ in $(seq 1 40); do
  body="$(curl -sS --max-time 1 "http://${listen_a}/messages" || true)"
  if printf '%s\n' "$body" | grep -Fq 'ответ tui'; then
    seen_a=1
    break
  fi
  sleep 0.1
done
[[ "$seen_a" -eq 1 ]] || fail "Flutter peer missing TUI reply"

kill "$pid_a" "$pid_b" 2>/dev/null || true
wait "$pid_a" 2>/dev/null || true
wait "$pid_b" 2>/dev/null || true
pid_a=""; pid_b=""

echo "flutter_send_test OK"
