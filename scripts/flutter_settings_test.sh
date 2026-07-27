#!/usr/bin/env bash
# Task-level contract for P066: mini-settings — nick only.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export PATH="/opt/homebrew/share/flutter/bin:${PATH:-}"

fail() {
  echo "flutter_settings_test FAIL: $*" >&2
  exit 1
}

command -v flutter >/dev/null 2>&1 || fail "flutter not on PATH"

[[ -f apps/dudka/lib/screens/settings_nick_screen.dart ]] || fail "SettingsNickScreen missing"
grep -q 'chat-settings' apps/dudka/lib/screens/chat_screen.dart || fail "chat settings entry missing"
grep -q 'settings-nick-field' apps/dudka/lib/screens/settings_nick_screen.dart || fail "nick field missing"
grep -q 'setNick' apps/dudka/lib/screens/settings_nick_screen.dart || fail "must call setNick"
# Forbid profile fields as UI labels (comments may mention them).
if grep -nE "labelText:.*(email|телефон|аватар|пароль)|Text\\('.*(email|телефон|аватар|пароль)" \
  apps/dudka/lib/screens/settings_nick_screen.dart >/dev/null; then
  fail "settings must stay nick-only"
fi

(
  cd apps/dudka
  flutter pub get >/dev/null
  flutter test test/settings_nick_test.dart
) || fail "flutter settings tests failed"

# Live: change nick via EngineClient (same path as settings save).
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"; [[ -n "${pid:-}" ]] && kill "$pid" 2>/dev/null || true' EXIT
go build -o "$tmpdir/dudkad" ./cmd/dudkad || fail "build dudkad"
log="$tmpdir/d.log"
"$tmpdir/dudkad" -data-dir "$tmpdir/d" -name "Before" -listen "127.0.0.1:0" \
  -announce-port 41790 -session-port 0 >"$log" 2>&1 &
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

(
  cd apps/dudka
  dart run tool/live_nick.dart "http://${listen}" "AfterNick"
) || fail "live_nick failed"

me="$(curl -sS --max-time 2 "http://${listen}/me")"
printf '%s\n' "$me" | grep -q 'AfterNick' || fail "GET /me missing new nick: $me"

kill "$pid" 2>/dev/null || true
wait "$pid" 2>/dev/null || true
pid=""

echo "flutter_settings_test OK"
