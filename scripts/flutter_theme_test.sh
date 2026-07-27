#!/usr/bin/env bash
# Task-level contract for P069: DESIGN.md charcoal / silkscreen / step-progress theme.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export PATH="/opt/homebrew/share/flutter/bin:${PATH:-}"

fail() {
  echo "flutter_theme_test FAIL: $*" >&2
  exit 1
}

command -v flutter >/dev/null 2>&1 || fail "flutter not on PATH"

[[ -f apps/dudka/lib/theme/dudka_theme.dart ]] || fail "dudka_theme.dart missing"
[[ -f apps/dudka/lib/widgets/step_progress.dart ]] || fail "step_progress.dart missing"
grep -q 'buildDudkaTheme' apps/dudka/lib/app.dart || fail "DudkaApp must apply theme"
grep -q 'StepProgress' apps/dudka/lib/screens/chat_screen.dart || fail "chat must use StepProgress"
grep -q '0xFF1A1A1A' apps/dudka/lib/theme/dudka_theme.dart || fail "panel charcoal missing"
grep -qiE 'scanline|crt|glow' apps/dudka/lib/theme/dudka_theme.dart \
  && fail "CRT fluff in theme" || true

(
  cd apps/dudka
  flutter pub get >/dev/null
  flutter test test/theme_test.dart
) || fail "flutter theme tests failed"

echo "flutter_theme_test OK"
