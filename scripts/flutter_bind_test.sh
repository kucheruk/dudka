#!/usr/bin/env bash
# Task-level contract for P060: Flutter↔dudkad bind ADR + hello GET /me.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1
export PATH="/opt/homebrew/share/flutter/bin:${PATH:-}"

fail() {
  echo "flutter_bind_test FAIL: $*" >&2
  exit 1
}

ADR=docs/design/flutter-bind.md
[[ -f "$ADR" ]] || fail "$ADR missing"
grep -qiE 'subprocess|loopback' "$ADR" || fail "ADR must choose subprocess/loopback bind"
grep -qiE 'GET /me|/me' "$ADR" || fail "ADR must mention GET /me proof"
grep -qiE 'Accepted|принято|решение' "$ADR" || fail "ADR must record a decision"

[[ -f apps/dudka/pubspec.yaml ]] || fail "apps/dudka/pubspec.yaml missing"
[[ -f apps/dudka/lib/engine.dart ]] || fail "apps/dudka/lib/engine.dart missing"
[[ -f apps/dudka/lib/main.dart ]] || fail "apps/dudka/lib/main.dart missing"
grep -q 'fetchMe' apps/dudka/lib/engine.dart || fail "EngineClient.fetchMe missing"
grep -q 'MeHelloScreen' apps/dudka/lib/main.dart || fail "hello /me screen missing"
grep -q 'GET /me' apps/dudka/lib/main.dart || fail "screen must label GET /me"

command -v flutter >/dev/null 2>&1 || fail "flutter not on PATH (install SDK)"
command -v dart >/dev/null 2>&1 || fail "dart not on PATH"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"; [[ -n "${pid:-}" ]] && kill "$pid" 2>/dev/null || true' EXIT

go build -o "$tmpdir/dudkad" ./cmd/dudkad || fail "go build dudkad failed"

log="$tmpdir/e.log"
port="$(python3 - <<'PY'
import socket
s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
)"
"$tmpdir/dudkad" -data-dir "$tmpdir/data" -name "Spike" -listen "127.0.0.1:0" \
  -announce-port "$port" -session-port 0 -announce-interval 1h >"$log" 2>&1 &
pid=$!

listen=""
for _ in $(seq 1 50); do
  if grep -q '^ready ' "$log" 2>/dev/null; then
    listen="$(grep '^listen=' "$log" | head -n 1 | sed 's/^listen=//')"
    break
  fi
  sleep 0.1
done
[[ -n "$listen" ]] || fail "dudkad not ready: $(cat "$log")"

me="$(curl -sS --max-time 2 "http://${listen}/me")"
echo "$me" | grep -q '"peer_id"' || fail "/me missing peer_id: $me"
echo "$me" | grep -q 'Spike' || fail "/me missing name Spike: $me"

(
  cd apps/dudka
  flutter pub get >/dev/null
  flutter test test/engine_me_test.dart
) || fail "flutter test engine_me_test failed"

# Live bind: same EngineClient the UI uses, against real dudkad.
(
  cd apps/dudka
  dart run tool/live_me.dart "http://${listen}"
) || fail "live EngineClient.fetchMe against dudkad failed"

kill "$pid" 2>/dev/null || true
wait "$pid" 2>/dev/null || true
pid=""

echo "flutter_bind_test OK"
