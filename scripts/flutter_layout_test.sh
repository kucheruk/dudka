#!/usr/bin/env bash
# Task-level contract for P070: wide dual-pane / narrow peer strip (DUD-UI-140).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export PATH="/opt/homebrew/share/flutter/bin:${PATH:-}"

fail() {
  echo "flutter_layout_test FAIL: $*" >&2
  exit 1
}

command -v flutter >/dev/null 2>&1 || fail "flutter not on PATH"

[[ -f apps/dudka/lib/layout/chat_layout.dart ]] || fail "chat_layout.dart missing"
grep -q 'dudkaWideBreakpoint' apps/dudka/lib/layout/chat_layout.dart || fail "breakpoint missing"
grep -q 'chat-layout-wide' apps/dudka/lib/screens/chat_screen.dart || fail "wide layout key missing"
grep -q 'chat-layout-narrow' apps/dudka/lib/screens/chat_screen.dart || fail "narrow layout key missing"
grep -q 'chat-peers-strip' apps/dudka/lib/screens/chat_screen.dart || fail "peer strip key missing"

(
  cd apps/dudka
  flutter pub get >/dev/null
  flutter test test/layout_test.dart
) || fail "flutter layout tests failed"

echo "flutter_layout_test OK"
