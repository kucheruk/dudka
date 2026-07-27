#!/usr/bin/env bash
# Task-level contract for P072: GUI/TUI user-facing strings are Russian (DUD-PRD-103).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export PATH="/opt/homebrew/share/flutter/bin:${PATH:-}"

fail() {
  echo "ru_ui_test FAIL: $*" >&2
  exit 1
}

command -v go >/dev/null 2>&1 || fail "go not on PATH"
command -v flutter >/dev/null 2>&1 || fail "flutter not on PATH"

# Static guards: silkscreen labels must not stay English in UI sources.
grep -q "СОСЕДИ" apps/dudka/lib/screens/chat_screen.dart || fail "Flutter СОСЕДИ label missing"
grep -q "СОСЕДИ" internal/tui/view.go || fail "TUI СОСЕДИ header missing"
grep -q "ЛЕНТА" internal/tui/view.go || fail "TUI ЛЕНТА header missing"
grep -q "онлайн" apps/dudka/lib/engine/client.dart || fail "Flutter онлайн status missing"
grep -q "онлайн" internal/tui/view.go || fail "TUI онлайн status missing"
grep -qE "Text\\('PEERS'\\)|\"PEERS\\\\n\"" apps/dudka/lib/screens/chat_screen.dart internal/tui/view.go \
  && fail "English PEERS label still present" || true
if grep -nE "Text\\('PEERS'\\)" apps/dudka/lib/screens/chat_screen.dart >/dev/null; then
  fail "Flutter still has Text('PEERS')"
fi
if grep -nE 'WriteString\("PEERS' internal/tui/view.go >/dev/null; then
  fail "TUI still writes PEERS"
fi
if grep -nE 'online %d|ENGINE OFFLINE|"FILE %s' internal/tui/view.go >/dev/null; then
  fail "TUI still has English online/ENGINE OFFLINE/FILE format"
fi
if grep -nE "online \\\$n|FILE \\\$" apps/dudka/lib/engine/client.dart >/dev/null; then
  fail "Flutter formatters still English"
fi

go test ./internal/tui/ -run 'TestRenderUserFacingRussian|TestRenderOfflineAndAloneRussian|TestRenderTransferMarkersRussian' -count=1 \
  || fail "Go RU UI tests failed"

(
  cd apps/dudka
  flutter pub get >/dev/null
  flutter test test/ru_ui_test.dart
) || fail "Flutter RU UI tests failed"

echo "ru_ui_test OK"
