#!/usr/bin/env bash
# Task-level contract for P063: chat wireframe — status + peers + text feed.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export PATH="/opt/homebrew/share/flutter/bin:${PATH:-}"

fail() {
  echo "flutter_chat_test FAIL: $*" >&2
  exit 1
}

command -v flutter >/dev/null 2>&1 || fail "flutter not on PATH"

[[ -f apps/dudka/lib/screens/chat_screen.dart ]] || fail "ChatScreen missing"
grep -q 'chat-status' apps/dudka/lib/screens/chat_screen.dart || fail "status strip key missing"
grep -q 'chat-peers' apps/dudka/lib/screens/chat_screen.dart || fail "peers pane key missing"
grep -q 'chat-feed' apps/dudka/lib/screens/chat_screen.dart || fail "feed pane key missing"
grep -q 'fetchSnapshot' apps/dudka/lib/engine/client.dart || fail "EngineClient.fetchSnapshot missing"
grep -q 'НИКОГО РЯДОМ' apps/dudka/lib/screens/chat_screen.dart || fail "alone copy missing"
grep -q 'НЕТ СЕТИ' apps/dudka/lib/screens/chat_screen.dart || fail "no_network copy missing"
grep -q 'ДУНУТЬ' apps/dudka/lib/screens/chat_screen.dart && fail "compose send is P064" || true

(
  cd apps/dudka
  flutter pub get >/dev/null
  flutter test \
    test/engine_chat_test.dart \
    test/chat_screen_test.dart \
    test/skeleton_me_test.dart \
    test/first_run_test.dart
) || fail "flutter chat wireframe tests failed"

echo "flutter_chat_test OK"
