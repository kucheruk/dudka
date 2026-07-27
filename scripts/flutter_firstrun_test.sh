#!/usr/bin/env bash
# Task-level contract for P062: first-run nick (RU) → chat.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export PATH="/opt/homebrew/share/flutter/bin:${PATH:-}"

fail() {
  echo "flutter_firstrun_test FAIL: $*" >&2
  exit 1
}

command -v flutter >/dev/null 2>&1 || fail "flutter not on PATH"

[[ -f apps/dudka/lib/screens/first_run_nick_screen.dart ]] || fail "FirstRunNickScreen missing"
[[ -f apps/dudka/lib/screens/chat_screen.dart ]] || fail "ChatScreen missing"
[[ -f apps/dudka/lib/session/first_run_store.dart ]] || fail "FirstRunStore missing"
[[ -f apps/dudka/lib/nick/fallback.dart ]] || fail "nick fallback missing"

grep -q 'FirstRunNickScreen' apps/dudka/lib/app.dart || fail "DudkaApp must host first-run"
grep -q 'ChatScreen' apps/dudka/lib/app.dart || fail "DudkaApp must host chat after first-run"
grep -q 'Как вас зовут' apps/dudka/lib/screens/first_run_nick_screen.dart \
  || fail "first-run copy must ask for nick in RU"
grep -q 'Пропустить' apps/dudka/lib/screens/first_run_nick_screen.dart \
  || fail "first-run must allow skip"
grep -qiE 'email|телефон|аватар' apps/dudka/lib/screens/first_run_nick_screen.dart \
  && fail "first-run must not require avatar/email/phone" || true

(
  cd apps/dudka
  flutter pub get >/dev/null
  flutter test \
    test/nick_fallback_test.dart \
    test/first_run_test.dart \
    test/skeleton_me_test.dart
) || fail "flutter first-run tests failed"

echo "flutter_firstrun_test OK"
