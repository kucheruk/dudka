#!/usr/bin/env bash
# Task-level contract for P061: macOS Flutter skeleton builds/runs and shows /me.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1
export PATH="/opt/homebrew/share/flutter/bin:${PATH:-}"

fail() {
  echo "flutter_skeleton_test FAIL: $*" >&2
  exit 1
}

command -v flutter >/dev/null 2>&1 || fail "flutter not on PATH"
command -v dart >/dev/null 2>&1 || fail "dart not on PATH"

[[ -f apps/dudka/lib/app.dart ]] || fail "lib/app.dart missing (skeleton)"
[[ -f apps/dudka/lib/screens/me_screen.dart ]] || fail "MeScreen missing"
[[ -f apps/dudka/lib/engine/host.dart ]] || fail "EngineHost missing"
[[ -f apps/dudka/macos/Runner.xcodeproj/project.pbxproj ]] || fail "macOS target missing"
grep -q 'MeScreen' apps/dudka/lib/app.dart || fail "DudkaApp must host MeScreen"
grep -q 'EngineHost' apps/dudka/lib/main.dart || fail "main must wire EngineHost spawn path"
grep -qiE 'macOS|macos|desktop' docs/design/flutter-bind.md \
  || fail "ADR must name macOS/desktop as first skeleton target"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

go build -o "$tmpdir/dudkad" ./cmd/dudkad || fail "go build dudkad failed"

(
  cd apps/dudka
  flutter pub get >/dev/null
  flutter test test/engine_host_test.dart test/engine_me_test.dart test/skeleton_me_test.dart
) || fail "flutter unit/widget tests failed"

# Live: EngineHost spawns dudkad and UI client reads /me.
(
  cd apps/dudka
  dart run tool/live_host_me.dart "$tmpdir/dudkad"
) || fail "live_host_me failed"

# Cheapest first target: macOS desktop build must succeed.
(
  cd apps/dudka
  flutter build macos --debug
) || fail "flutter build macos failed"

app="$(find apps/dudka/build/macos -name 'dudka.app' -type d 2>/dev/null | head -n 1)"
[[ -n "$app" ]] || fail "dudka.app not produced by macos build"
[[ -x "$app/Contents/MacOS/dudka" ]] || fail "macos binary missing in $app"

echo "flutter_skeleton_test OK app=$app"
