#!/usr/bin/env bash
# Task-level contract for P065: alone/no_network + «ИСКАТЬ» → POST /scan.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1
export PATH="/opt/homebrew/share/flutter/bin:${PATH:-}"

fail() {
  echo "flutter_seek_test FAIL: $*" >&2
  exit 1
}

command -v flutter >/dev/null 2>&1 || fail "flutter not on PATH"
command -v dart >/dev/null 2>&1 || fail "dart not on PATH"

grep -q 'ИСКАТЬ' apps/dudka/lib/screens/chat_screen.dart || fail "ИСКАТЬ button missing"
grep -q 'chat-seek' apps/dudka/lib/screens/chat_screen.dart || fail "chat-seek key missing"
grep -q 'startScan' apps/dudka/lib/engine/client.dart || fail "EngineClient.startScan missing"

(
  cd apps/dudka
  flutter pub get >/dev/null
  flutter test \
    test/network_seek_test.dart \
    test/chat_screen_test.dart
) || fail "flutter seek/network tests failed"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"; for p in ${pid_a:-} ${pid_b:-}; do [[ -n "$p" ]] && kill "$p" 2>/dev/null || true; done' EXIT
go build -o "$tmpdir/dudkad" ./cmd/dudkad || fail "build dudkad"

# Different announce ports + long interval = UDP path filtered (same as scan_test).
port_a="$(python3 - <<'PY'
import socket
s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
)"
port_b="$(python3 - <<'PY'
import socket
s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
)"

log_a="$tmpdir/a.log"; log_b="$tmpdir/b.log"
"$tmpdir/dudkad" -data-dir "$tmpdir/a" -name "SeekA" -listen "127.0.0.1:0" \
  -announce-port "$port_a" -session-port 0 -announce-interval 1h >"$log_a" 2>&1 &
pid_a=$!
"$tmpdir/dudkad" -data-dir "$tmpdir/b" -name "SeekB" -listen "127.0.0.1:0" \
  -announce-port "$port_b" -session-port 0 -announce-interval 1h >"$log_b" 2>&1 &
pid_b=$!

listen_a=""; peer_b=""; session_b=""
for _ in $(seq 1 50); do
  if grep -q '^ready ' "$log_a" 2>/dev/null && grep -q '^ready ' "$log_b" 2>/dev/null; then
    listen_a="$(grep '^listen=' "$log_a" | head -n 1 | sed 's/^listen=//')"
    peer_b="$(sed -n 's/^peer_id=//p' "$log_b" | head -1)"
    session_b="$(grep '^session_tcp=' "$log_b" | head -n 1 | sed 's/^session_tcp=//')"
    break
  fi
  sleep 0.1
done
[[ -n "$listen_a" && -n "$session_b" && -n "$peer_b" ]] \
  || fail "not ready; a=$(cat "$log_a") b=$(cat "$log_b")"

(
  cd apps/dudka
  dart run tool/live_scan.dart "http://${listen_a}" "127.0.0.1" "$session_b"
) || fail "Flutter startScan did not find peer"

peers="$(curl -sS --max-time 2 "http://${listen_a}/peers")"
python3 - "$peers" "$peer_b" <<'PY' || fail "peers table missing scan hit"
import json, sys
d = json.loads(sys.argv[1])
peer_b = sys.argv[2]
ids = {p.get("peer_id") for p in d.get("peers") or []}
assert peer_b in ids, d
print("peers_ok")
PY

kill "$pid_a" "$pid_b" 2>/dev/null || true
wait "$pid_a" 2>/dev/null || true
wait "$pid_b" 2>/dev/null || true
pid_a=""; pid_b=""

echo "flutter_seek_test OK"
